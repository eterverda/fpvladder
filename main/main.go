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
	var pilotDate string

	var draftCmd = &cobra.Command{
		Use:   "draft [file_path]",
		Short: "Интерактивное создание черновика эвента",
		Args:  cobra.ExactArgs(1),
		Run:   handleDraft,
	}

	// Новая команда add
	var addCmd = &cobra.Command{
		Use:   "pilot",
		Short: "Работа с пилотами",
	}

	// Подкоманда add pilot
	var pilotAddCmd = &cobra.Command{
		Use:   "add [name]",
		Short: "Создать карточку нового пилота",
		Args:  cobra.ExactArgs(1),
		Run:   func(cmd *cobra.Command, args []string) { handlePilotAdd(cmd, args, pilotDate) },
	}

	// Добавляем флаг --date (или короткий -d)
	pilotAddCmd.Flags().StringVarP(&pilotDate, "date", "d", model.Today().String(), "Дата регистрации пилота (формат YYYY-MM-DD)")

	addCmd.AddCommand(pilotAddCmd)
	rootCmd.AddCommand(draftCmd, addCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func handlePilotAdd(cmd *cobra.Command, args []string, date string) {
	name := args[0]

	// Очередность выбора даты:
	// 1. Из флага --date
	// 2. Если флаг пуст — текущая дата
	targetDate := model.Today()
	if date != "" {
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			log.Fatalf("Ошибка: дата должна быть в формате YYYY-MM-DD (получено: %s)", date)
		}
		targetDate = model.Date(parsedDate)
	}

	// Передаем targetDate в CalculateNextId, чтобы он искал в нужной папке
	// Согласно твоей логике: год берется из даты, месяц-день — тоже.
	newId, err := db.GenerateNextId(DBPath, "pilot", targetDate)
	if err != nil {
		log.Fatalf("Ошибка генерации ID: %v", err)
	}

	// Строим путь: data/pilot/YYYY/MM-DD/N.yaml
	parts := strings.Split(string(newId), "/")
	targetPath := filepath.Join(DBPath, "pilot", parts[0], parts[1], parts[2]+".yaml")

	if err := createPilotFile(targetPath, newId, name); err != nil {
		log.Fatalf("Ошибка записи файла: %v", err)
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
		return fmt.Errorf("ошибка маршалинга: %w", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return err
}

func handleDraft(cmd *cobra.Command, args []string) {
	targetPath := args[0]

	// 1. Инициализация файла, если его нет
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		// Генерируем новый ID через твой пакет db
		newId, err := db.GenerateNextId(DBPath, "event", model.Today())
		if err != nil {
			log.Fatalf("Ошибка вычисления ID: %v", err)
		}

		if err := initializeEventFile(targetPath, newId); err != nil {
			log.Fatalf("Ошибка создания файла: %v", err)
		}
		fmt.Printf("Создан новый черновик эвента с ID: %s\n", newId)
	}

	// 2. Цикл редактирования
	runEditLoop(targetPath)
}

func initializeEventFile(path string, id model.Id) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Твой шаблон с комментариями
	today := time.Now().Format("2006-01-02")
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
		id, today,
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
			log.Printf("Предупреждение: Редактор закрылся с ошибкой: %v", err)
		}

		// 3. Валидация прямо здесь
		err := validateEvent(path)
		if err == nil {
			fmt.Println("[OK] Файл валиден и готов к работе.")
			break
		}

		fmt.Printf("\n[!] Ошибка в структуре эвента: %v\n", err)
		fmt.Print("Вернуться в редактор для исправления? (Y/n): ")

		var input string
		fmt.Scanln(&input)
		if strings.ToLower(input) == "n" {
			fmt.Println("ВНИМАНИЕ: Файл сохранен с ошибками и может не работать.")
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

	// Простая бизнес-валидация
	if event.Id == "" {
		return fmt.Errorf("поле 'id' не может быть пустым")
	}
	if event.Name == "" || event.Name == "Название события" {
		return fmt.Errorf("поле 'name' должно быть заполнено")
	}

	return nil
}
