package main

import (
	"fmt"
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

	// Подкоманда pilot add
	var eventDraftCmd = &cobra.Command{
		Use:   "draft [file_path]",
		Short: "Интерактивное создание черновика эвента",
		Args:  cobra.ExactArgs(1),
		Run:   func(cmd *cobra.Command, args []string) { handleDraft(cmd, args, date) },
	}

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

// validateEvent проверяет файл на соответствие модели Event
func validateEvent(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("не удалось прочитать файл: %w", err)
	}

	var event model.Event
	if err := yaml.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("ошибка синтаксиса YAML: %w", err)
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
		// Используем твой метод для получения пути к файлу пилота
		pilotPath := db.ResolveIdPath(DBPath, "pilot", model.Id(id))

		pData, err := os.ReadFile(pilotPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("пилот [%s] не найден в БД (путь: %s)", id, pilotPath)
			}
			return fmt.Errorf("ошибка доступа к БД пилотов: %w", err)
		}

		var dbPilot model.Pilot
		if err := yaml.Unmarshal(pData, &dbPilot); err != nil {
			return fmt.Errorf("ошибка структуры файла пилота %s: %w", id, err)
		}

		// Сверка имен (предупреждение, если не совпадают)
		if !strings.EqualFold(dbPilot.Name, p.Name) {
			fmt.Printf("[!] Расхождение имен для ID %s:, %s vs %s\n", id, dbPilot.Name, p.Name)
		}
	}

	return nil
}
