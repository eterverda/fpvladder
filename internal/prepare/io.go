package prepare

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/eterverda/fpvladder/internal/db"
	"github.com/eterverda/fpvladder/internal/model"
	"gopkg.in/yaml.v3"
)

// NewEventModel создаёт новое или загружает существующее событие
// (обёртка для обратной совместимости)
func NewEventModel(filename string) (*EventModel, error) {
	if filename == "" {
		return NewEmptyEventModel(), nil
	}
	return LoadEventModel(filename)
}

// LoadEventModel загружает событие из файла
func LoadEventModel(filename string) (*EventModel, error) {
	m := &EventModel{
		Filename: filename,
		IsNew:    false,
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл: %w", err)
	}
	if err := yaml.Unmarshal(data, &m.Event); err != nil {
		return nil, fmt.Errorf("не удалось распарсить YAML: %w", err)
	}

	// Группируем пилотов по командам
	if err := m.loadFromPilots(); err != nil {
		return nil, err
	}

	m.sortRows()

	// Валидируем позиции команд
	if err := m.validatePositions(); err != nil {
		return nil, err
	}

	m.addVirtualRow()
	return m, nil
}

// NewEmptyEventModel создаёт новое пустое событие
func NewEmptyEventModel() *EventModel {
	m := &EventModel{
		Event: model.Event{},
		IsNew: true,
	}
	m.addVirtualRow()
	return m
}

// loadFromPilots загружает пилотов из Event.Pilots и группирует по командам
// Возвращает ошибку если у какого-либо пилота пустое имя
func (m *EventModel) loadFromPilots() error {
	// Группируем пилотов по командам
	// Team:0 - отдельные команды для каждого пилота
	// Team:N (N>0) - группируем по одинаковому Team

	teamGroups := make(map[int][]model.PilotEntry)

	for _, p := range m.Event.Pilots {
		if p.Name == "" {
			return fmt.Errorf("пилот с ID %s имеет пустое имя", p.Id)
		}
		teamGroups[p.Team] = append(teamGroups[p.Team], p)
	}

	// Создаем TeamRow
	for teamNum, pilots := range teamGroups {
		if teamNum == 0 {
			// Каждый пилот с Team:0 - отдельная команда
			for _, p := range pilots {
				m.Rows = append(m.Rows, TeamRow{
					Position: p.Position,
					Team:     0, // Будет назначен при сериализации
					Pilots: []PilotRow{
						{Id: p.Id, Name: p.Name},
					},
					Virtual: false,
				})
			}
		} else {
			// Группа пилотов с одинаковым Team
			var pilotRows []PilotRow
			position := pilots[0].Position

			for _, p := range pilots {
				pilotRows = append(pilotRows, PilotRow{Id: p.Id, Name: p.Name})
			}

			// Сортируем пилотов по имени
			sort.Slice(pilotRows, func(i, j int) bool {
				return pilotRows[i].Name < pilotRows[j].Name
			})

			m.Rows = append(m.Rows, TeamRow{
				Position: position,
				Team:     teamNum,
				Pilots:   pilotRows,
				Virtual:  false,
			})
		}
	}

	return nil
}

// validatePositions проверяет что позиции команд идут по порядку без пропусков
// и что ничьи оформлены правильно
func (m *EventModel) validatePositions() error {
	if len(m.Rows) == 0 {
		return nil
	}

	// Собираем команды по позициям
	positionGroups := make(map[int][]int) // Position.Int -> индексы команд
	for i, row := range m.Rows {
		if row.Virtual {
			continue
		}
		positionGroups[row.Position.Int] = append(positionGroups[row.Position.Int], i)
	}

	// Проверяем что позиции идут подряд без пропусков
	expectedPos := 1
	for {
		indices, exists := positionGroups[expectedPos]
		if !exists {
			// Проверяем, есть ли позиции больше expectedPos
			hasHigher := false
			for pos := range positionGroups {
				if pos > expectedPos {
					hasHigher = true
					break
				}
			}
			if hasHigher {
				return fmt.Errorf("пропущена позиция %d", expectedPos)
			}
			break
		}

		// Проверяем правильность ничьи
		teamCount := len(indices)
		if teamCount > 1 {
			// Ничья - проверяем что у всех команд одинаковый Position.Int и правильный TieCount
			expectedTieCount := teamCount - 1
			for _, idx := range indices {
				if m.Rows[idx].Position.TieCount != expectedTieCount {
					return fmt.Errorf("неправильно оформлена ничья на позиции %d: ожидается TieCount=%d, получено TieCount=%d",
						expectedPos, expectedTieCount, m.Rows[idx].Position.TieCount)
				}
			}
		} else if teamCount == 1 {
			// Одиночная позиция - проверяем что TieCount = 0
			if m.Rows[indices[0]].Position.TieCount != 0 {
				return fmt.Errorf("команда на позиции %d имеет TieCount=%d, ожидается 0 для одиночной позиции",
					expectedPos, m.Rows[indices[0]].Position.TieCount)
			}
		}

		// Следующая позиция = текущая + количество команд в ничье
		expectedPos += teamCount
	}

	return nil
}

// Save сохраняет событие в файл
// Сохраняются только пилоты с IdKind == IdKindExplicit (явно подтверждённые ID)
func (m *EventModel) Save() error {
	// Конвертируем команды обратно в пилотов
	pilots := []model.PilotEntry{}

	teamCounter := 1

	for _, r := range m.Rows {
		if r.Virtual || len(r.Pilots) == 0 {
			continue
		}

		if r.IsSingle() {
			// Одиночный пилот - Team:0
			// Сохраняем только если IdKind == Explicit
			if r.Pilots[0].IdKind == IdKindExplicit {
				pilots = append(pilots, model.PilotEntry{
					Position: r.Position,
					Team:     0,
					Id:       r.Pilots[0].Id,
					Name:     r.Pilots[0].Name,
				})
			}
		} else {
			// Команда из нескольких пилотов - назначаем Team номер
			// Сохраняем только пилотов с IdKind == Explicit
			teamPilots := []model.PilotEntry{}
			for _, p := range r.Pilots {
				if p.IdKind == IdKindExplicit {
					teamPilots = append(teamPilots, model.PilotEntry{
						Position: r.Position,
						Team:     teamCounter,
						Id:       p.Id,
						Name:     p.Name,
					})
				}
			}
			// Сохраняем команду только если есть хотя бы один пилот с explicit ID
			if len(teamPilots) > 0 {
				pilots = append(pilots, teamPilots...)
				teamCounter++
			}
		}
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

// GenerateFilename генерирует имя файла на основе даты и ID события
func (m *EventModel) GenerateFilename() string {
	presumedId, _ := db.GenerateNextId("./data", "event", m.Event.Date)
	return presumedId.String() + ".yaml"
}
