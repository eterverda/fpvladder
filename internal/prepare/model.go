package prepare

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/eterverda/fpvladder/internal/db"
	"github.com/eterverda/fpvladder/internal/model"
	"gopkg.in/yaml.v3"
)

// PilotRow представляет строку в таблице пилотов
type PilotRow struct {
	Id       model.Id
	Position model.Position
	Team     int
	Name     string
	Virtual  bool // true для виртуальной строки-заглушки
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
		Filename: filename,
		IsNew:    filename == "",
	}

	if filename != "" {
		// Загружаем существующий файл
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
				Id:       p.Id,
				Position: p.Position,
				Team:     p.Team,
				Name:     name,
				Virtual:  false,
			})
		}

		m.sortRows()
		m.addVirtualRow()
		return m, nil
	}

	// Новое событие
	m.Event = model.Event{}
	m.addVirtualRow()
	return m, nil
}

// sortRows сортирует строки по Position.Int, затем по имени
func (m *EventModel) sortRows() {
	sort.SliceStable(m.Rows, func(i, j int) bool {
		if m.Rows[i].Virtual != m.Rows[j].Virtual {
			return !m.Rows[i].Virtual
		}
		if m.Rows[i].Virtual {
			return false
		}
		if m.Rows[i].Position.Int != m.Rows[j].Position.Int {
			return m.Rows[i].Position.Int < m.Rows[j].Position.Int
		}
		return m.Rows[i].Name < m.Rows[j].Name
	})
}

// findRowIndex находит индекс строки по имени
func (m *EventModel) findRowIndex(name string) int {
	for i, r := range m.Rows {
		if r.Name == name {
			return i
		}
	}
	return -1
}

// getRowsAtPosition возвращает строки с заданным Position.Int
func (m *EventModel) getRowsAtPosition(posInt int) []int {
	var indices []int
	for i, r := range m.Rows {
		if !r.Virtual && r.Position.Int == posInt {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m *EventModel) addVirtualRow() {
	maxPos := 0
	for _, r := range m.Rows {
		if !r.Virtual {
			_, end := getTieRange(r.Position)
			if end > maxPos {
				maxPos = end
			}
		}
	}
	m.Rows = append(m.Rows, PilotRow{
		Position: model.Position{Int: maxPos + 1},
		Virtual:  true,
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
func (m *EventModel) UpdateRow(index int, pos model.Position, name string, id model.Id) {
	if index < 0 || index >= len(m.Rows) {
		return
	}
	row := &m.Rows[index]
	if !row.Position.Equal(pos) || row.Name != name || row.Id != id {
		row.Position = pos
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

// GenerateFilename генерирует имя файла на основе даты и ID события
func (m *EventModel) GenerateFilename() string {
	presumedId, _ := db.GenerateNextId("./data", "event", m.Event.Date)
	return presumedId.String() + ".yaml"
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
			Position: r.Position,
			Team:     r.Team,
			Id:       r.Id,
			Name:     r.Name,
		})
	}

	// Сортируем пилотов по позиции и имени перед сохранением
	sort.Slice(pilots, func(i, j int) bool {
		if pilots[i].Position.Int != pilots[j].Position.Int {
			return pilots[i].Position.Int < pilots[j].Position.Int
		}
		return pilots[i].Name < pilots[j].Name
	})

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
