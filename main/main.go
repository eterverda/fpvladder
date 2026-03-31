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
	"github.com/eterverda/fpvladder/internal/prepare"
	"github.com/eterverda/fpvladder/internal/site"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	DbPath   = "./data"
	SitePath = "./build"
)

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
		Use:   "pilot [name]+",
		Short: "Создать карточку нового пилота",
		Args:  cobra.MinimumNArgs(1),
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

	var genCmd = &cobra.Command{
		Use:   "generate",
		Short: "Сгенерировать сайт",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return site.Generate(DbPath, SitePath) },
	}

	csvCmd.Flags().StringVarP(&class, "class", "c", "drone-racing > 75mm", "Класс")

	var prepareCmd = &cobra.Command{
		Use:   "prepare [filename]",
		Short: "Подготовить событие (редактор TUI)",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filename := ""
			if len(args) > 0 {
				filename = args[0]
			}
			if err := prepare.Run(filename); err != nil {
				log.Fatalf("[✕] Ошибка: %v", err)
			}
		},
	}

	rootCmd.AddCommand(installCmd, pilotCmd, csvCmd, genCmd, prepareCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func handlePilotAdd(cmd *cobra.Command, names []string, date string) {
	targetDate := model.Today()
	if date != "" {
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			log.Fatalf("[✕] Ошибка: дата должна быть в формате YYYY-MM-DD (получено: %s)", date)
		}
		targetDate = model.Date(parsedDate)
	}

	for _, name := range names {
		// Передаем targetDate в CalculateNextId, чтобы он искал в нужной папке
		// Согласно твоей логике: год берется из даты, месяц-день — тоже.
		newId, err := db.GenerateNextId(DbPath, "pilot", targetDate)
		if err != nil {
			log.Fatalf("[✕] Ошибка генерации ID: %v", err)
		}

		// Строим путь: data/pilot/YYYY/MM-DD/N.yaml
		parts := strings.Split(string(newId), "/")
		targetPath := filepath.Join(DbPath, "pilot", parts[0], parts[1], parts[2]+".yaml")

		if err := createPilotFile(targetPath, newId, name); err != nil {
			log.Fatalf("[✕] Ошибка записи файла: %v", err)
		}

		data, _ := os.ReadFile(targetPath)
		fmt.Printf("[✓] Пилот добавлен: %s\n", targetPath)
		fmt.Printf("------------\n%s------------\n", string(data))
	}
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
	data, err := model.MarshalPrettyYaml(newPilot)
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

	event, err := db.ReadEventPath(path)
	if err != nil {
		log.Fatalf("[✕] Не удалось прочитать эвент: %s\n", err)
	}

	presumedId, err := db.GenerateNextId(DbPath, "event", event.Date)
	if event.Id != "" && event.Id != presumedId {
		log.Fatalln("[✕] Неверный Id")
	}
	event.Id = presumedId

	targetPath := db.ResolveIdPath(DbPath, "event", event.Id)

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
		pilot, err := db.ReadPilot(DbPath, id)
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
			inputs[i] = elo.Input{
				Position: entry.Position.Int,
				Team:     entry.Team,
				Rating:   oldRatingValue,
			}
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
				Position: model.RelativePosition{
					Position: event.Pilots[i].Position,
					Count:    len(event.Pilots),
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

	eventData, _ := model.MarshalPrettyYaml(event)
	eventFile := db.ResolveIdPath(DbPath, "event", event.Id)

	err := os.MkdirAll(filepath.Dir(eventFile), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(eventFile, eventData, 0644)
	if err != nil {
		return err
	}

	for _, pilot := range pilots {
		pilotData, _ := model.MarshalPrettyYaml(pilot)
		pilotFile := db.ResolveIdPath(DbPath, "pilot", pilot.Id)

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

// validateEvent проверяет файл на соответствие модели Event
func validateEvent(path string) error {

	event, err := db.ReadEventPath(path)
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

	// Карта для проверки уникальности пилотов в рамках ОДНОГО эвента
	eventPilotIds := make(map[string]bool)

	// Группировка пилотов по позиции для проверки ничьих
	positionGroups := make(map[int][]model.PilotEntry)

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

		// В. Проверка позиции
		if p.Position.Int <= 0 {
			return fmt.Errorf("позиция пилота %s должна быть > 0", p.Name)
		}

		// Группируем по позиции
		pos := p.Position.Int
		positionGroups[pos] = append(positionGroups[pos], p)
	}

	// Г. Валидация позиций, ничьих и команд
	if err := validatePositions(positionGroups); err != nil {
		return err
	}

	// --- ЦИКЛ 2: Верификация пилотов по внешней базе данных ---

	for id, p := range allUniquePilots {
		dbPilot, err := db.ReadPilot(DbPath, model.Id(id))
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

// validatePositions проверяет согласованность позиций, ничьих и команд
func validatePositions(positionGroups map[int][]model.PilotEntry) error {
	if len(positionGroups) == 0 {
		return fmt.Errorf("нет пилотов в событии")
	}

	// Находим максимальную позицию
	maxPos := 0
	for pos := range positionGroups {
		if pos > maxPos {
			maxPos = pos
		}
	}

	// Проверяем наличие 1-й позиции
	if _, ok := positionGroups[1]; !ok {
		return fmt.Errorf("отсутствует 1-е место")
	}

	// Проверяем последовательность позиций и ничьи
	expectedPos := 1
	for expectedPos <= maxPos {
		pilots, ok := positionGroups[expectedPos]
		if !ok {
			return fmt.Errorf("пропущено место %d (последовательность прервана)", expectedPos)
		}

		// Проверяем согласованность ничьей
		for _, p := range pilots {
			if p.Position.TieCount == 0 {
				// Нет ничьей — должна быть только одна позиция
				if len(pilots) != 1 {
					return fmt.Errorf("позиция %d: несколько пилотов без указания ничьей (нужен формат %d-%d)",
						expectedPos, expectedPos, expectedPos+len(pilots)-1)
				}
			} else {
				// Есть ничья — проверяем согласованность
				expectedEnd := expectedPos + p.Position.TieCount
				actualEnd := expectedPos + len(pilots) - 1
				if expectedEnd != actualEnd {
					return fmt.Errorf("позиция %s: несогласованная ничья (ожидалось %d-%d, получено %d-%d)",
						p.Position.String(), expectedPos, expectedEnd, expectedPos, actualEnd)
				}
			}
		}

		// Проверяем команды: пилоты с одинаковой позицией должны быть из разных команд
		// или не иметь команды (Team = 0)
		teamSet := make(map[int]bool)
		for _, p := range pilots {
			if p.Team > 0 {
				if teamSet[p.Team] {
					return fmt.Errorf("позиция %d: пилоты из одной команды %d не могут делить место",
						expectedPos, p.Team)
				}
				teamSet[p.Team] = true
			}
		}

		// Переходим к следующей позиции (с учётом ничьей)
		expectedPos += len(pilots)
	}

	return nil
}

func handleExportCsv(cmd *cobra.Command, args []string, class string) {
	// 1. Собираем данные
	type record struct {
		Name   string
		Rating int
	}
	var results []record

	// Рекурсивно обходим папку с пилотами
	pilotDir := db.ResolveTypePath(DbPath, "pilot")
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
