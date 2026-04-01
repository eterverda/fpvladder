package prepare

import (
	"sort"

	"github.com/eterverda/fpvladder/internal/model"
)

// IdKind вид ID пилота
type IdKind int

const (
	IdKindVirtual   IdKind = iota // Виртуальный ID (сгенерирован автоматически)
	IdKindSuggested               // Предложенный ID из базы (показывается со * или ?)
	IdKindExplicit                // Подтверждённый ID (явно выбран пользователем)
)

// PilotRow представляет пилота внутри команды
type PilotRow struct {
	Id     model.Id
	IdKind IdKind
	Name   string
}

// TeamRow представляет команду (один или несколько пилотов)
type TeamRow struct {
	Position model.Position
	Team     int
	Pilots   []PilotRow
	Virtual  bool // true для виртуальной строки-заглушки
}

// Name возвращает имя команды (имя первого пилота для сортировки)
func (t *TeamRow) Name() string {
	if len(t.Pilots) == 0 {
		return ""
	}
	return t.Pilots[0].Name
}

// IsSingle возвращает true если команда состоит из одного пилота
func (t *TeamRow) IsSingle() bool {
	return len(t.Pilots) == 1
}

// TotalPilots возвращает общее количество пилотов во всех командах
func TotalPilots(rows []TeamRow) int {
	count := 0
	for _, r := range rows {
		count += len(r.Pilots)
	}
	return count
}

// GetPilotByIndex возвращает (teamIndex, pilotIndex) по глобальному индексу пилота
func GetPilotByIndex(rows []TeamRow, globalIndex int) (teamIdx int, pilotIdx int, ok bool) {
	current := 0
	for i, row := range rows {
		for j := range row.Pilots {
			if current == globalIndex {
				return i, j, true
			}
			current++
		}
	}
	return -1, -1, false
}

// EventModel хранит состояние события
type EventModel struct {
	Event    model.Event
	Rows     []TeamRow
	Modified bool
	Filename string
	IsNew    bool
}

// sortRows сортирует команды по Position.Int, затем по имени первого пилота
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
		return m.Rows[i].Name() < m.Rows[j].Name()
	})

	// Сортируем пилотов внутри каждой команды по имени
	for i := range m.Rows {
		if !m.Rows[i].Virtual {
			sort.Slice(m.Rows[i].Pilots, func(a, b int) bool {
				return m.Rows[i].Pilots[a].Name < m.Rows[i].Pilots[b].Name
			})
		}
	}
}

// findRowIndex находит индекс команды по имени первого пилота
func (m *EventModel) findRowIndex(name string) int {
	for i, r := range m.Rows {
		if r.Name() == name {
			return i
		}
	}
	return -1
}

// getRowsAtPosition возвращает индексы команд с заданным Position.Int
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
	m.Rows = append(m.Rows, TeamRow{
		Position: model.Position{Int: maxPos + 1},
		Virtual:  true,
		Pilots:   []PilotRow{{IdKind: IdKindVirtual}}, // Виртуальный пилот
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

// UpdatePilot обновляет данные пилота по глобальному индексу
func (m *EventModel) UpdatePilot(globalIndex int, name string, id model.Id) {
	teamIdx, pilotIdx, ok := GetPilotByIndex(m.Rows, globalIndex)
	if !ok {
		return
	}

	row := &m.Rows[teamIdx]
	if row.Pilots[pilotIdx].Name != name || row.Pilots[pilotIdx].Id != id {
		row.Pilots[pilotIdx].Name = name
		row.Pilots[pilotIdx].Id = id
		m.Modified = true
	}
}

// GetPilot возвращает пилота по глобальному индексу
func (m *EventModel) GetPilot(globalIndex int) (teamIdx int, pilotIdx int, pilot *PilotRow, ok bool) {
	teamIdx, pilotIdx, ok = GetPilotByIndex(m.Rows, globalIndex)
	if !ok {
		return -1, -1, nil, false
	}
	return teamIdx, pilotIdx, &m.Rows[teamIdx].Pilots[pilotIdx], true
}

// DeletePilot удаляет пилота по глобальному индексу
// Если в команде остается 0 пилотов - команда удаляется
func (m *EventModel) DeletePilot(globalIndex int) {
	teamIdx, pilotIdx, _, ok := m.GetPilot(globalIndex)
	if !ok {
		return
	}

	row := &m.Rows[teamIdx]
	row.Pilots = append(row.Pilots[:pilotIdx], row.Pilots[pilotIdx+1:]...)

	// Если в команде не осталось пилотов - удаляем команду
	if len(row.Pilots) == 0 && !row.Virtual {
		m.Rows = append(m.Rows[:teamIdx], m.Rows[teamIdx+1:]...)
	}

	m.Modified = true
}

// getTieRange возвращает диапазон ничьи: (startInt, endInt) для данной позиции
// Для Position{7, 2} вернёт (7, 9) — позиции 7, 8, 9
func getTieRange(pos model.Position) (start, end int) {
	start = pos.Int
	end = pos.Int + pos.TieCount
	return
}

// IsVirtualPilot проверяет, является ли пилот виртуальным (из виртуальной команды)
func (m *EventModel) IsVirtualPilot(globalIndex int) bool {
	teamIdx, _, ok := GetPilotByIndex(m.Rows, globalIndex)
	if !ok {
		return false
	}
	return m.Rows[teamIdx].Virtual
}

// GetTeamForPilot возвращает команду для пилота по глобальному индексу
func (m *EventModel) GetTeamForPilot(globalIndex int) (teamIdx int, team *TeamRow, ok bool) {
	teamIdx, _, ok = GetPilotByIndex(m.Rows, globalIndex)
	if !ok {
		return -1, nil, false
	}
	return teamIdx, &m.Rows[teamIdx], true
}
