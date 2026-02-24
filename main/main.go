package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eterverda/fpvladder/internal/db"
	"github.com/eterverda/fpvladder/internal/elo"
	"github.com/eterverda/fpvladder/internal/model"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const DBPath = "./data"

func main() {
	var rootCmd = &cobra.Command{Use: "droon"}
	var date string
	var class string

	// Kоманда install
	var installCmd = &cobra.Command{
		Use:   "install [file_path]",
		Short: "Добавить событие в БД и пересчитать все рейтинги",
		Args:  cobra.ExactArgs(1),
		Run:   handleInstall,
	}

	// Kоманда pilot
	var pilotCmd = &cobra.Command{
		Use:   "pilot [name]",
		Short: "Создать карточку нового пилота",
		Args:  cobra.ExactArgs(1),
		Run:   func(cmd *cobra.Command, args []string) { handlePilotAdd(cmd, args, date) },
	}

	// Добавляем флаг --date (или короткий -d)
	pilotCmd.Flags().StringVarP(&date, "date", "d", model.Today().String(), "Дата регистрации пилота (формат YYYY-MM-DD)")

	var csvCmd = &cobra.Command{
		Use:   "csv [filename]",
		Short: "Экспорт рейтингов в csv",
		Args:  cobra.ExactArgs(1),
		Run:   func(cmd *cobra.Command, args []string) { handleExportCsv(cmd, args, class) },
	}

	csvCmd.Flags().StringVarP(&class, "class", "c", "drone-racing > 75mm", "Класс")

	rootCmd.AddCommand(installCmd, pilotCmd, csvCmd)

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
		Careers: []model.Career{}, // Пустой слайс превратится в []
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

	err = recalculateRatings(event)
	if err != nil {
		log.Fatalln("[✕] Не удалось пересчитать рейтинги")
	}

}

func recalculateRatings(event *model.Event) error {
	var simpleEvent = model.Event{
		Id:   event.Id,
		Date: event.Date,
		Name: event.Name,
	}

	var pilots = make([]*model.Pilot, len(event.Pilots))

	class := event.Class

	for i, entry := range event.Pilots {
		id := entry.Id
		pilot, err := readPilot(id)
		if err != nil {
			return err
		}
		pilots[i] = pilot
	}

	for class != "" {
		fmt.Printf("[ ] Обработка этапа класса %s\n", string(class))

		// сначала собираем все данные
		var inputs = make([]elo.Input, len(event.Pilots))
		var originIds = make([]model.Id, len(event.Pilots))
		for i, entry := range event.Pilots {
			pilot := pilots[i]
			oldRatingValue := 1200
			var originId model.Id
			for _, career := range pilot.Careers {
				if class == career.Class {
					lastRating := career.Ratings[len(career.Ratings)-1]
					oldRatingValue = lastRating.Value
					originId = lastRating.Event.Id
					break
				}
			}
			inputs[i] = elo.Input{Position: entry.Position, Rating: oldRatingValue}
			originIds[i] = originId
		}

		// потом пересчитываем пакетно
		deltas := elo.GroupKCalc(inputs)

		// потом раскладываем выходные данные
		for i := range event.Pilots {
			input := inputs[i]
			originId := originIds[i]
			delta := deltas[i]

			pilot := pilots[i]

			oldRatingValue := input.Rating
			if originId == "" {
				oldRatingValue = 0
			}
			newRatingValue := input.Rating + delta

			rating := model.RatingAssignment{
				Class:     class,
				OriginId:  originId,
				OldValue:  oldRatingValue,
				Algorithm: model.Algorithm(elo.Algorithm),
				Delta:     delta,
				NewValue:  newRatingValue,
			}
			event.Pilots[i].Ratings = append(event.Pilots[i].Ratings, rating)

			summary := model.RatingSummary{
				Num:   1,
				Event: simpleEvent,
				Position: model.Position{
					Numerator:   inputs[i].Position,
					Denominator: len(event.Pilots),
				},
				Delta: delta,
				Value: newRatingValue,
			}

			var updatedSummary = false
			for j, career := range pilot.Careers {
				if career.Class == class {
					summary.Num += career.Ratings[len(career.Ratings)-1].Num
					pilot.Careers[j].Ratings = append(career.Ratings, summary)
					updatedSummary = true
					break
				}
			}
			if !updatedSummary {
				career := model.Career{
					Class:   class,
					Ratings: []model.RatingSummary{summary},
				}
				pilot.Careers = append(pilot.Careers, career)
			}

			fmt.Printf("    %s: %v -> %v\n", pilot.Name, input.Rating, newRatingValue)
		}

		class = class.Parent()
	}

	eventData, _ := yaml.Marshal(event)
	eventFile := db.ResolveIdPath(DBPath, "event", event.Id)

	err := os.MkdirAll(filepath.Dir(eventFile), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(eventFile, eventData, 0644)
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

	posCounts := make(map[int]int)
	maxPos := 0

	// Карта для проверки уникальности пилотов в рамках ОДНОГО эвента
	eventPilotIds := make(map[string]bool)

	for _, p := range event.Pilots {
		// А. Проверка наличия ID (Обязательно по твоему требованию)
		idStr := string(p.Id)
		if idStr == "" || idStr == "~" {
			return fmt.Errorf("у пилота '%s' не указан ID (id обязателен)", p.Name)
		}

		// Б. Уникальность ID внутри этапа
		if eventPilotIds[idStr] {
			return fmt.Errorf("дубликат пилота с ID %s", idStr)
		}
		eventPilotIds[idStr] = true

		// Собираем для верификации по базе (общий список по всему файлу)
		allUniquePilots[idStr] = p

		// В. Сбор данных по позициям
		if p.Position <= 0 {
			return fmt.Errorf("позиция пилота %s должна быть > 0", p.Name)
		}
		posCounts[p.Position]++
		if p.Position > maxPos {
			maxPos = p.Position
		}
	}

	// Г. Логика "командности" и последовательности мест
	teamSize := posCounts[1]
	if teamSize == 0 {
		return fmt.Errorf("отсутствует 1-е место")
	}

	for i := 1; i <= maxPos; i++ {
		count, ok := posCounts[i]
		if !ok {
			return fmt.Errorf("пропущено место %d (последовательность прервана)", i)
		}
		if count != teamSize {
			return fmt.Errorf("неровные составы. На 1-м месте %d чел, а на %d-м — %d",
				teamSize, i, count)
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
			fmt.Printf("[!] Расхождение имен для ID %s: %s vs %s\n", id, dbPilot.Name, p.Name)
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

func handleExportCsv(cmd *cobra.Command, args []string, class string) {
	// 1. Собираем данные
	type record struct {
		Name   string
		Rating int
	}
	var results []record

	// Рекурсивно обходим папку с пилотами
	pilotDir := db.ResolveTypePath(DBPath, "pilot")
	err := filepath.Walk(pilotDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".yaml" {
			return nil
		}

		// Читаем файл пилота
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		} // Пропускаем битые файлы

		var pilot model.Pilot
		if err := yaml.Unmarshal(data, &pilot); err != nil {
			log.Fatalf("[✕] Ошибка при чтении файлов: %v", err)
		}

		// Ищем нужный класс в карточке пилота
		for _, r := range pilot.Careers {
			if string(r.Class) == class {
				results = append(results, record{Name: pilot.Name, Rating: r.Ratings[len(r.Ratings)-1].Value})
				break
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("[✕] Ошибка при чтении файлов: %v", err)
	}

	// 2. Сортируем по рейтингу (от большего к меньшему)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Rating > results[j].Rating
	})

	dest, err := os.Create(args[0])
	if err != nil {
		log.Fatalf("[✕] Ошибка при создании файла: %v", err)
	}
	defer dest.Close()

	// 3. Выводим в CSV

	// Заголовки (без мест, как ты и просил)
	_, err = fmt.Fprintf(dest, "rank, name, result\n")
	if err != nil {
		log.Fatalf("[✕] Ошибка при записи файла: %v", err)
	}

	for i, res := range results {
		_, err = fmt.Fprintf(dest, "%v, %s, %v\n", i+1, res.Name, res.Rating)
		if err != nil {
			log.Fatalf("[✕] Ошибка при записи файла: %v", err)
		}
	}
}
