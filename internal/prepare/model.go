package prepare

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/eterverda/fpvladder/internal/db"
	"github.com/eterverda/fpvladder/internal/model"
	"gopkg.in/yaml.v3"
)

// PilotRow представляет строку в таблице пилотов
type PilotRow struct {
	Id      model.Id
	Place   int
	Name    string
	Virtual bool // true для виртуальной строки-заглушки
}

// EventModel хранит состояние события
type EventModel struct {
	Event    model.Event
	Rows     []PilotRow
	Modified bool
	Filename string
	IsNew    bool
}

// NewEventModel создаёт новое или загружает существующее событие
func NewEventModel(filename string) (*EventModel, error) {
	m := &EventModel{
		Rows:     []PilotRow{},
		Filename: filename,
	}

	if filename == "" {
		// Создаём новое событие
		m.IsNew = true
		m.Event = model.Event{
			Id:     "~",
			Date:   model.Today(),
			Name:   "~",
			Class:  "drone-racing > 75mm",
			Pilots: []model.PilotEntry{},
		}
		m.addVirtualRow()
		return m, nil
	}

	// Загружаем существующее
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл: %w", err)
	}

	if err := yaml.Unmarshal(data, &m.Event); err != nil {
		return nil, fmt.Errorf("не удалось распарсить YAML: %w", err)
	}

	// Конвертируем пилотов в строки
	for _, p := range m.Event.Pilots {
		name := p.Name
		if name == "" {
			// Загружаем имя из базы если не указано
			if pilot, err := db.ReadPilot("./data", p.Id); err == nil {
				name = pilot.Name
			}
		}
		m.Rows = append(m.Rows, PilotRow{
			Id:      p.Id,
			Place:   p.Position,
			Name:    name,
			Virtual: false,
		})
	}

	m.addVirtualRow()
	return m, nil
}

func (m *EventModel) addVirtualRow() {
	// Находим максимальный place
	maxPlace := 0
	for _, r := range m.Rows {
		if !r.Virtual && r.Place > maxPlace {
			maxPlace = r.Place
		}
	}
	m.Rows = append(m.Rows, PilotRow{
		Place:   maxPlace + 1,
		Virtual: true,
	})
}

// PromoteVirtualRow делает виртуальную строку реальной
func (m *EventModel) PromoteVirtualRow(index int) {
	if index >= 0 && index < len(m.Rows) && m.Rows[index].Virtual {
		m.Rows[index].Virtual = false
		m.Modified = true
		m.addVirtualRow()
	}
}

// UpdateRow обновляет данные строки
func (m *EventModel) UpdateRow(index int, place int, name string, id model.Id) {
	if index < 0 || index >= len(m.Rows) {
		return
	}
	row := &m.Rows[index]
	if row.Place != place || row.Name != name || row.Id != id {
		row.Place = place
		row.Name = name
		row.Id = id
		m.Modified = true
	}
}

// DeleteRow удаляет строку
func (m *EventModel) DeleteRow(index int) {
	if index < 0 || index >= len(m.Rows) || m.Rows[index].Virtual {
		return
	}
	m.Rows = append(m.Rows[:index], m.Rows[index+1:]...)
	m.Modified = true
}

// MoveRowUp перемещает строку вверх (уменьшает позицию)
func (m *EventModel) MoveRowUp(index int) int {
	if index <= 0 || index >= len(m.Rows) || m.Rows[index].Virtual {
		return index
	}
	// Меняем местами с предыдущей строкой
	m.Rows[index-1], m.Rows[index] = m.Rows[index], m.Rows[index-1]
	// Пересчитываем позиции
	m.recalculatePlaces()
	m.Modified = true
	return index - 1
}

// MoveRowDown перемещает строку вниз (увеличивает позицию)
func (m *EventModel) MoveRowDown(index int) int {
	if index < 0 || index >= len(m.Rows)-1 || m.Rows[index].Virtual {
		return index
	}
	// Нельзя двигать ниже виртуальной строки
	if m.Rows[index+1].Virtual {
		return index
	}
	// Меняем местами со следующей строкой
	m.Rows[index+1], m.Rows[index] = m.Rows[index], m.Rows[index+1]
	// Пересчитываем позиции
	m.recalculatePlaces()
	m.Modified = true
	return index + 1
}

// recalculatePlaces пересчитывает позиции всех строк
func (m *EventModel) recalculatePlaces() {
	for i := range m.Rows {
		m.Rows[i].Place = i + 1
	}
}

// Save сохраняет событие в файл
func (m *EventModel) Save() error {
	// Конвертируем строки обратно в пилотов
	pilots := []model.PilotEntry{}
	for _, r := range m.Rows {
		if r.Virtual {
			continue
		}
		pilots = append(pilots, model.PilotEntry{
			Position: r.Place,
			Id:       r.Id,
			Name:     r.Name,
		})
	}
	m.Event.Pilots = pilots

	// Сериализуем
	data, err := model.MarshalPrettyYaml(m.Event)
	if err != nil {
		return fmt.Errorf("не удалось сериализовать: %w", err)
	}

	// Создаём директории если нужно
	dir := filepath.Dir(m.Filename)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("не удалось создать директорию: %w", err)
		}
	}

	if err := os.WriteFile(m.Filename, data, 0644); err != nil {
		return fmt.Errorf("не удалось записать файл: %w", err)
	}

	m.Modified = false
	m.IsNew = false
	return nil
}

// FindPilotsByName ищет пилотов по имени с учётом текущего класса события
func (m *EventModel) FindPilotsByName(name string) []FindResult {
	var results []FindResult
	if name == "" {
		return results
	}

	searchWords := strings.Fields(strings.ToLower(name))
	pilots, _ := db.ListIds("./data", "pilot")
	currentClass := string(m.Event.Class)
	if currentClass == "" {
		currentClass = string(model.Class75mm)
	}

	for _, id := range pilots {
		pilot, err := db.ReadPilot("./data", id)
		if err != nil {
			continue
		}
		pilotWords := strings.Fields(strings.ToLower(pilot.Name))

		// Проверяем: подмножество слов поиска в имени пилота ИЛИ подмножество слов пилота в поиске
		if isSubset(searchWords, pilotWords) || isSubset(pilotWords, searchWords) {
			rating := 1200 // По умолчанию для новых пилотов
			career := pilot.CareerForClass(model.Class(currentClass))
			if len(career.Ratings) > 0 {
				rating = career.Ratings[len(career.Ratings)-1].Value
			}

			results = append(results, FindResult{
				Name:   pilot.Name,
				Id:     string(pilot.Id),
				Rating: rating,
			})
		}
	}

	return results
}

func isSubset(a, b []string) bool {
	if len(a) > len(b) {
		return false
	}
	for _, aw := range a {
		found := false
		for _, bw := range b {
			if strings.Contains(bw, aw) || strings.Contains(aw, bw) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// FindResult результат поиска пилота
type FindResult struct {
	Name   string
	Id     string
	Rating int
}

// GenerateFilename генерирует имя файла для нового события
func (m *EventModel) GenerateFilename() string {
	date := m.Event.Date
	presumedId, _ := db.GenerateNextId("./data", "event", date)
	return presumedId.String() + ".yaml"
}

// ParsePlace парсит строку места в число
func ParsePlace(s string) int {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0
	}
	// Убираем скобки если есть
	s = strings.Trim(s, "()")
	n, _ := strconv.Atoi(s)
	return n
}
