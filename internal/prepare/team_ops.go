package prepare

import (
	"sort"

	"github.com/eterverda/fpvladder/internal/model"
)

// MergeTeam объединяет все команды в ничье с указанной в одну команду
// Возвращает индекс новой объединённой команды
func (m *EventModel) MergeTeam(teamIdx int) int {
	if teamIdx < 0 || teamIdx >= len(m.Rows) || m.Rows[teamIdx].Virtual {
		return teamIdx
	}

	team := m.Rows[teamIdx]
	if team.Position.TieCount == 0 {
		// Нет ничьи - нечего объединять
		return teamIdx
	}

	// Находим все команды в этой ничье
	tieStart := team.Position.Int
	tieIndices := m.getRowsAtPosition(tieStart)
	if len(tieIndices) <= 1 {
		return teamIdx
	}

	// Собираем всех пилотов из команд в ничье
	var mergedPilots []PilotRow
	for _, idx := range tieIndices {
		mergedPilots = append(mergedPilots, m.Rows[idx].Pilots...)
	}

	// Сортируем пилотов по имени
	sort.Slice(mergedPilots, func(i, j int) bool {
		return mergedPilots[i].Name < mergedPilots[j].Name
	})

	// Создаём новую объединённую команду (выходим из ничьи)
	// Позиция = начало ничьи, TieCount = 0
	newTeam := TeamRow{
		Position: model.Position{Int: tieStart},
		Team:     0,
		Pilots:   mergedPilots,
		Virtual:  false,
	}

	// Удаляем старые команды (в обратном порядке чтобы индексы не съезжали)
	sort.Slice(tieIndices, func(i, j int) bool {
		return tieIndices[i] > tieIndices[j]
	})
	for _, idx := range tieIndices {
		m.Rows = append(m.Rows[:idx], m.Rows[idx+1:]...)
	}

	// Вставляем новую команду на место первой удалённой
	insertIdx := tieIndices[len(tieIndices)-1]
	m.Rows = append(m.Rows[:insertIdx], append([]TeamRow{newTeam}, m.Rows[insertIdx:]...)...)

	// Пересчитываем позиции (подтягиваем места)
	m.recalculatePositions()

	m.Modified = true
	return insertIdx
}

// SplitTeam разбивает команду на отдельные команды по одному пилоту (в ничье)
// Возвращает индекс первой новой команды
func (m *EventModel) SplitTeam(teamIdx int) int {
	if teamIdx < 0 || teamIdx >= len(m.Rows) || m.Rows[teamIdx].Virtual {
		return teamIdx
	}

	team := m.Rows[teamIdx]
	if len(team.Pilots) <= 1 {
		// Нечего разбивать
		return teamIdx
	}

	position := team.Position
	pilots := team.Pilots

	// Сортируем пилотов по имени
	sort.Slice(pilots, func(i, j int) bool {
		return pilots[i].Name < pilots[j].Name
	})

	// Создаём отдельные команды для каждого пилота
	// Все они будут в ничье
	newTeams := make([]TeamRow, len(pilots))
	tieCount := len(pilots) - 1
	for i, pilot := range pilots {
		newTeams[i] = TeamRow{
			Position: model.Position{Int: position.Int, TieCount: tieCount},
			Team:     0,
			Pilots:   []PilotRow{pilot},
			Virtual:  false,
		}
	}

	// Удаляем старую команду
	m.Rows = append(m.Rows[:teamIdx], m.Rows[teamIdx+1:]...)

	// Вставляем новые команды
	m.Rows = append(m.Rows[:teamIdx], append(newTeams, m.Rows[teamIdx:]...)...)

	// Пересчитываем позиции (следующие команды ухудшают позиции)
	m.recalculatePositions()

	m.Modified = true
	return teamIdx
}

// recalculatePositions пересчитывает позиции всех команд после изменений
func (m *EventModel) recalculatePositions() {
	// Группируем команды по текущим позициям
	positionGroups := make(map[int][]int)
	for i, row := range m.Rows {
		if row.Virtual {
			continue
		}
		positionGroups[row.Position.Int] = append(positionGroups[row.Position.Int], i)
	}

	// Собираем уникальные позиции и сортируем
	var positions []int
	for pos := range positionGroups {
		positions = append(positions, pos)
	}
	sort.Ints(positions)

	// Пересчитываем позиции
	currentPos := 1
	for _, oldPos := range positions {
		indices := positionGroups[oldPos]
		teamCount := len(indices)

		if teamCount > 1 {
			// Ничья
			for _, idx := range indices {
				m.Rows[idx].Position.Int = currentPos
				m.Rows[idx].Position.TieCount = teamCount - 1
			}
			currentPos += teamCount
		} else {
			// Одиночная позиция
			m.Rows[indices[0]].Position.Int = currentPos
			m.Rows[indices[0]].Position.TieCount = 0
			currentPos++
		}
	}

	// Обновляем позицию виртуальной строки
	for i := range m.Rows {
		if m.Rows[i].Virtual {
			m.Rows[i].Position = model.Position{Int: currentPos}
			break
		}
	}
}
