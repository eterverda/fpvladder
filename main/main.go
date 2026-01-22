package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eterverda/fpvladder/internal/db"
	"github.com/eterverda/fpvladder/internal/model"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const DBPath = "./data"

func main() {
	var rootCmd = &cobra.Command{Use: "droon"}
	var date string

	var eventCmd = &cobra.Command{
		Use:   "event",
		Short: "Работа с эвентами",
	}

	// Подкоманда event draft
	var eventDraftCmd = &cobra.Command{
		Use:   "draft [file_path]",
		Short: "Интерактивное создание черновика эвента",
		Args:  cobra.ExactArgs(1),
		Run:   func(cmd *cobra.Command, args []string) { handleDraft(cmd, args, date) },
	}

	// Подкоманда event install
	var eventInstallCmd = &cobra.Command{
		Use:   "install [file_path]",
		Short: "Добавить событие в БД и пересчитать все рейтинги",
		Args:  cobra.ExactArgs(1),
		Run:   handleInstall,
	}

	eventCmd.AddCommand(eventInstallCmd)

	// Добавляем флаг --date (или короткий -d)
	eventDraftCmd.Flags().StringVarP(&date, "date", "d", model.Today().String(), "Дата проведения эвента (формат YYYY-MM-DD)")

	eventCmd.AddCommand(eventDraftCmd)

	var pilotCmd = &cobra.Command{
		Use:   "pilot",
		Short: "Работа с пилотами",
	}

	// Подкоманда pilot add
	var pilotAddCmd = &cobra.Command{
		Use:   "add [name]",
		Short: "Создать карточку нового пилота",
		Args:  cobra.ExactArgs(1),
		Run:   func(cmd *cobra.Command, args []string) { handlePilotAdd(cmd, args, date) },
	}

	// Добавляем флаг --date (или короткий -d)
	pilotAddCmd.Flags().StringVarP(&date, "date", "d", model.Today().String(), "Дата регистрации пилота (формат YYYY-MM-DD)")

	pilotCmd.AddCommand(pilotAddCmd)
	rootCmd.AddCommand(eventCmd, pilotCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func handlePilotAdd(cmd *cobra.Command, args []string, date string) {
	name := args[0]

	targetDate := model.Today()
	if date != "" {
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			log.Fatalf("[✕] Ошибка: дата должна быть в формате YYYY-MM-DD (получено: %s)", date)
		}
		targetDate = model.Date(parsedDate)
	}

	// Передаем targetDate в CalculateNextId, чтобы он искал в нужной папке
	// Согласно твоей логике: год берется из даты, месяц-день — тоже.
	newId, err := db.GenerateNextId(DBPath, "pilot", targetDate)
	if err != nil {
		log.Fatalf("[✕] Ошибка генерации ID: %v", err)
	}

	// Строим путь: data/pilot/YYYY/MM-DD/N.yaml
	parts := strings.Split(string(newId), "/")
	targetPath := filepath.Join(DBPath, "pilot", parts[0], parts[1], parts[2]+".yaml")

	if err := createPilotFile(targetPath, newId, name); err != nil {
		log.Fatalf("[✕] Ошибка записи файла: %v", err)
	}

	data, _ := os.ReadFile(targetPath)
	fmt.Printf("[✓] Пилот добавлен: %s\n", targetPath)
	fmt.Printf("------------\n%s------------\n", string(data))
}

func createPilotFile(path string, id model.Id, name string) error {
	// Гарантируем наличие папок
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Используем твою структуру из пакета model
	newPilot := model.Pilot{
		Id:      model.Id(id),
		Name:    name,
		Ratings: []model.RatingSummary{}, // Пустой слайс превратится в []
	}

	// Маршалим в YAML
	data, err := yaml.Marshal(newPilot)
	if err != nil {
		return fmt.Errorf("[✕] Ошибка маршалинга: %w", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return err
}

func handleInstall(cmd *cobra.Command, args []string) {
	path := args[0]

	err := validateEvent(path)
	if err != nil {
		log.Fatalf("[✕] Файл невалиден и не готов к работе: %s\n", err)
	}

	event, err := readEvent(path)
	if err != nil {
		log.Fatalf("[✕] Не удалось прочитать эвент: %s\n", err)
	}

	presumedId, err := db.GenerateNextId(DBPath, "event", event.Date)
	if event.Id != presumedId {
		log.Fatalln("[✕] Неверный Id")
	}
	targetPath := db.ResolveIdPath(DBPath, "event", event.Id)

	// 3. Скопировали файл
	err = copyFile(path, targetPath)
	if err != nil {
		log.Fatalln("[✕] Не получилось скопировать файл")
	}

	err = recalculateRatings(*event)
	if err != nil {
		log.Fatalln("[✕] Не удалось пересчитать рейтинги")
	}
}

func recalculateRatings(event model.Event) error {
	var pilots = make(map[model.Id]*model.Pilot)
	var records = make(map[model.Id]*model.PilotRecord)

	for _, stage := range event.Stages {
		class := stage.Class

		for _, entry := range stage.Pilots {
			id := entry.Id
			if _, ok := pilots[id]; !ok {
				pilot, err := readPilot(id)
				if err != nil {
					return err
				}
				pilots[id] = pilot
				records[id] = &model.PilotRecord{
					Id:   pilot.Id,
					Name: pilot.Name,
				}
			}
		}

		for class != "" {
			fmt.Printf("[ ] Обработка этапа \"%s\" для класса %s\n", stage.Name, string(class))

			var inputs = make([]pilotInput, 0, len(stage.Pilots))
			var originIds = make([]model.Id, 0, len(stage.Pilots))
			for _, entry := range stage.Pilots {
				pilot := pilots[entry.Id]
				oldRatingValue := 1200
				var originId model.Id
				for _, summary := range pilot.Ratings {
					if class == summary.Class {
						oldRatingValue = summary.Value
						originId = summary.OriginId
					}
				}
				input := pilotInput{position: entry.Position, oldRatingValue: oldRatingValue}
				inputs = append(inputs, input)
				originIds = append(originIds, originId)
			}

			for i, entry := range stage.Pilots {
				input := inputs[i]

				record := records[entry.Id]
				pilot := pilots[entry.Id]

				delta := recalculateRating(inputs, i)

				newRatingValue := input.oldRatingValue + delta

				var originId model.Id
				if len(record.Ratings) == 0 {
					originId = originIds[i]
				}

				rating := model.RatingAssignment{
					Class:     class,
					StageName: stage.Name,
					OriginId:  originId,
					OldValue:  input.oldRatingValue,
					Algorithm: model.Algorithm("elo > k30"),
					Delta:     delta,
					NewValue:  newRatingValue,
				}
				record.Ratings = append(record.Ratings, rating)

				summary := model.RatingSummary{
					Class:    rating.Class,
					Value:    rating.NewValue,
					OriginId: event.Id,
					Date:     event.Date,
					Qty:      1,
				}

				var updatedSummary = false
				for j, oldSummary := range pilot.Ratings {
					if summary.Class == oldSummary.Class {
						summary.Qty += oldSummary.Qty
						pilot.Ratings[j] = summary
						updatedSummary = true
					}
				}
				if !updatedSummary {
					pilot.Ratings = append(pilot.Ratings, summary)
				}

				fmt.Printf("    %s -> %v\n", pilot.Name, newRatingValue)
			}

			class = class.Parent()
		}
	}

	journal := &model.Journal{
		EventId:     event.Id,
		Description: fmt.Sprintf("Пересчет рейтингов по системе Elo по результатам события %s", event.Name),
		Date:        event.Date,
	}
	for _, record := range records {
		journal.Pilots = append(journal.Pilots, *record)
	}
	journalData, _ := yaml.Marshal(journal)
	journalFile := db.ResolveIdPath(DBPath, "journal", journal.EventId)

	err := os.MkdirAll(filepath.Dir(journalFile), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(journalFile, journalData, 0644)
	if err != nil {
		return err
	}

	for _, pilot := range pilots {
		pilotData, _ := yaml.Marshal(pilot)
		pilotFile := db.ResolveIdPath(DBPath, "pilot", pilot.Id)

		err = os.WriteFile(pilotFile, pilotData, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	// 0. Создали иерархию папок (0755 - стандартные права)
	err := os.MkdirAll(filepath.Dir(dst), 0755)
	if err != nil {
		return fmt.Errorf("не удалось создать папки: %w", err)
	}

	// 1. Открываем исходный файл
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// 2. Создаем целевой файл (или перезаписываем существующий)
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// 3. Копируем содержимое
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// 4. Фиксируем запись на диске
	return destFile.Sync()
}

func handleDraft(cmd *cobra.Command, args []string, date string) {
	targetPath := args[0]

	targetDate := model.Today()
	if date != "" {
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			log.Fatalf("[✕] Ошибка: дата должна быть в формате YYYY-MM-DD (получено: %s)", date)
		}
		targetDate = model.Date(parsedDate)
	}

	// 1. Инициализация файла, если его нет
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		// Генерируем новый ID через твой пакет db
		newId, err := db.GenerateNextId(DBPath, "event", targetDate)
		if err != nil {
			log.Fatalf("[✕] Ошибка вычисления ID: %v", err)
		}

		if err := initializeEventFile(targetPath, newId, targetDate); err != nil {
			log.Fatalf("[✕] Ошибка создания файла: %v", err)
		}
		fmt.Printf("[✓] Создан новый черновик события с ID: %s\n", newId)
	}

	// 2. Цикл редактирования
	runEditLoop(targetPath)
}

func initializeEventFile(path string, id model.Id, date model.Date) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Твой шаблон с комментариями
	template := fmt.Sprintf(
		`# Заполните данные о мероприятии
id: %s # id мероприятия (присваивается автоматически)
date: %s # дата проведения мероприятия
name: ~ # название мероприятия
organizer:
  name: ~ # имя организатора или название организации
stages:
  - name: Главный этап # Название этапа
    class: drone-racing > 75mm > individual # класс гонки
    pilots:
      - position: 1
        id: ~ # id пилота
        name: Иван Иванов # имя пилота
      - position: 2
        id: ~ # id пилота
        name: Иван Петров-Сидоров # имя пилота
`,
		id, date.String(),
	)

	return os.WriteFile(path, []byte(template), 0644)
}

func runEditLoop(path string) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = "vim"
	}

	for {
		editCmd := exec.Command(editor, path)
		editCmd.Stdin = os.Stdin
		editCmd.Stdout = os.Stdout
		editCmd.Stderr = os.Stderr

		if err := editCmd.Run(); err != nil {
			log.Printf("[!] Предупреждение: Редактор закрылся с ошибкой: %v", err)
		}

		// 3. Валидация прямо здесь
		err := validateEvent(path)
		if err == nil {
			fmt.Println("[✓] Файл валиден и готов к работе.")
			break
		}

		fmt.Printf("\n[✕] Ошибка в структуре эвента: %v\n", err)
		fmt.Print("[?] Вернуться в редактор для исправления? (Y/n): ")

		var input string
		fmt.Scanln(&input)
		if strings.ToLower(input) == "n" {
			fmt.Println("[!] Файл сохранен с ошибками и может не работать.")
			break
		}
	}
}

func readEvent(path string) (*model.Event, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл: %w", err)
	}

	var event model.Event
	if err := yaml.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("ошибка синтаксиса YAML: %w", err)
	}

	return &event, nil
}

// validateEvent проверяет файл на соответствие модели Event
func validateEvent(path string) error {

	event, err := readEvent(path)
	if err != nil {
		return err
	}

	// 1. Базовая проверка заголовка
	if event.Name == "" || event.Name == "~" {
		return fmt.Errorf("поле 'name' эвента не заполнено")
	}

	// Карта для сбора уникальных пилотов всего эвента (ID -> Name)
	// Нужна для второго цикла верификации по файловой базе
	allUniquePilots := make(map[string]model.PilotEntry)

	// --- ЦИКЛ 1: Внутренняя логика этапов ---
	for _, stage := range event.Stages {
		if len(stage.Pilots) == 0 {
			continue
		}

		posCounts := make(map[int]int)
		maxPos := 0

		// Карта для проверки уникальности пилотов в рамках ОДНОГО этапа
		stagePilotIds := make(map[string]bool)

		for _, p := range stage.Pilots {
			// А. Проверка наличия ID (Обязательно по твоему требованию)
			idStr := string(p.Id)
			if idStr == "" || idStr == "~" {
				return fmt.Errorf("этап '%s': у пилота '%s' не указан ID (id обязателен)", stage.Name, p.Name)
			}

			// Б. Уникальность ID внутри этапа
			if stagePilotIds[idStr] {
				return fmt.Errorf("этап '%s': дубликат пилота с ID %s", stage.Name, idStr)
			}
			stagePilotIds[idStr] = true

			// Собираем для верификации по базе (общий список по всему файлу)
			allUniquePilots[idStr] = p

			// В. Сбор данных по позициям
			if p.Position <= 0 {
				return fmt.Errorf("этап '%s': позиция пилота %s должна быть > 0", stage.Name, p.Name)
			}
			posCounts[p.Position]++
			if p.Position > maxPos {
				maxPos = p.Position
			}
		}

		// Г. Логика "командности" и последовательности мест
		teamSize := posCounts[1]
		if teamSize == 0 {
			return fmt.Errorf("этап '%s': отсутствует 1-е место", stage.Name)
		}

		for i := 1; i <= maxPos; i++ {
			count, ok := posCounts[i]
			if !ok {
				return fmt.Errorf("этап '%s': пропущено место %d (последовательность прервана)", stage.Name, i)
			}
			if count != teamSize {
				return fmt.Errorf("этап '%s': неровные составы. На 1-м месте %d чел, а на %d-м — %d",
					stage.Name, teamSize, i, count)
			}
		}
	}

	// --- ЦИКЛ 2: Верификация пилотов по внешней базе данных ---

	for id, p := range allUniquePilots {
		dbPilot, err := readPilot(model.Id(id))
		if err != nil {
			return err
		}
		// Сверка имен (предупреждение, если не совпадают)
		if !strings.EqualFold(dbPilot.Name, p.Name) {
			fmt.Printf("[!] Расхождение имен для ID %s:, %s vs %s\n", id, dbPilot.Name, p.Name)
		}
	}

	return nil
}

func readPilot(id model.Id) (*model.Pilot, error) {
	// Используем твой метод для получения пути к файлу пилота
	pilotPath := db.ResolveIdPath(DBPath, "pilot", model.Id(id))

	pData, err := os.ReadFile(pilotPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("пилот [%s] не найден в БД (путь: %s)", id, pilotPath)
		}
		return nil, fmt.Errorf("ошибка доступа к БД пилотов: %w", err)
	}

	var dbPilot model.Pilot
	if err := yaml.Unmarshal(pData, &dbPilot); err != nil {
		return nil, fmt.Errorf("ошибка структуры файла пилота %s: %w", id, err)
	}
	return &dbPilot, nil
}
