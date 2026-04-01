package prepare

import "github.com/eterverda/fpvladder/internal/model"

// MoveRowUp перемещает команду вверх с учётом ничьих
func (m *EventModel) MoveRowUp(index int) int {
	if index < 0 || index >= len(m.Rows) || m.Rows[index].Virtual {
		return index
	}

	rowName := m.Rows[index].Name()
	currentPos := m.Rows[index].Position
	currentStart, currentEnd := getTieRange(currentPos)

	// Если мы в ничье — сначала выходим из неё
	if currentPos.TieCount > 0 {
		// Оставшаяся ничья смещается на +1
		tieIndices := m.getRowsAtPosition(currentStart)
		if len(tieIndices) <= 2 {
			// Были вдвоём — оставшийся становится одиночной на +1
			for _, idx := range tieIndices {
				if idx != index {
					m.Rows[idx].Position = model.Position{Int: currentStart + 1}
				}
			}
		} else {
			// Было больше двух — смещаем на +1, уменьшаем TieCount
			newTieCount := currentEnd - currentStart - 1
			newPos := model.Position{Int: currentStart + 1, TieCount: newTieCount}
			for _, idx := range tieIndices {
				if idx != index {
					m.Rows[idx].Position = newPos
				}
			}
		}

		// Теперь мы одиночные — встаём на место старой ничьи
		m.Rows[index].Position = model.Position{Int: currentStart}
		m.sortRows()
		m.Modified = true
		return m.findRowIndex(rowName)
	}

	// Мы одиночные — ищем кого подвинуть вверх
	// Находим строки выше (с end < currentStart)
	var aboveIndices []int
	for i, r := range m.Rows {
		if r.Virtual {
			continue
		}
		_, end := getTieRange(r.Position)
		if end < currentStart {
			aboveIndices = append(aboveIndices, i)
		}
	}

	// Если выше никого — ничего не делаем
	if len(aboveIndices) == 0 {
		return index
	}

	// Находим ближайшую строку сверху (максимальный end)
	closestAbove := aboveIndices[0]
	_, closestEnd := getTieRange(m.Rows[closestAbove].Position)
	for _, idx := range aboveIndices[1:] {
		_, end := getTieRange(m.Rows[idx].Position)
		if end > closestEnd {
			closestEnd = end
			closestAbove = idx
		}
	}

	abovePos := m.Rows[closestAbove].Position
	aboveStart, aboveEnd := getTieRange(abovePos)

	// Мы одиночные
	if abovePos.TieCount == 0 {
		// Оба одиночные — создаём ничью
		newPos := model.Position{Int: aboveStart, TieCount: 1}
		m.Rows[closestAbove].Position = newPos
		m.Rows[index].Position = newPos
	} else {
		// Мы одиночные, выше ничья — расширяем ничью вверх
		newTieCount := aboveEnd - aboveStart + 1
		newPos := model.Position{Int: aboveStart, TieCount: newTieCount}
		// Обновляем все в верхней ничье
		for _, idx := range m.getRowsAtPosition(aboveStart) {
			m.Rows[idx].Position = newPos
		}
		m.Rows[index].Position = newPos
	}

	m.sortRows()
	m.Modified = true
	return m.findRowIndex(rowName)
}

// MoveRowDown перемещает команду вниз с учётом ничьих
func (m *EventModel) MoveRowDown(index int) int {
	if index < 0 || index >= len(m.Rows) || m.Rows[index].Virtual {
		return index
	}

	rowName := m.Rows[index].Name()
	currentPos := m.Rows[index].Position
	currentStart, currentEnd := getTieRange(currentPos)

	// Находим строки ниже (с start > currentEnd)
	var belowIndices []int
	for i, r := range m.Rows {
		if r.Virtual {
			continue
		}
		start, _ := getTieRange(r.Position)
		if start > currentEnd {
			belowIndices = append(belowIndices, i)
		}
	}

	// Если мы в ничье — сначала выходим из неё
	if currentPos.TieCount > 0 {
		// Оставшаяся ничья остаётся на месте (мы уходим вниз)
		tieIndices := m.getRowsAtPosition(currentStart)
		if len(tieIndices) <= 2 {
			// Были вдвоём — оставшийся становится одиночной на текущей позиции
			for _, idx := range tieIndices {
				if idx != index {
					m.Rows[idx].Position = model.Position{Int: currentStart}
				}
			}
		} else {
			// Было больше двух — уменьшаем TieCount, позиция та же
			newTieCount := currentEnd - currentStart - 1
			newPos := model.Position{Int: currentStart, TieCount: newTieCount}
			for _, idx := range tieIndices {
				if idx != index {
					m.Rows[idx].Position = newPos
				}
			}
		}

		// Теперь мы одиночные — встаём на позицию currentEnd (бывший конец ничьи)
		m.Rows[index].Position = model.Position{Int: currentEnd}
		m.sortRows()
		m.Modified = true
		return m.findRowIndex(rowName)
	}

	// Мы одиночные — ищем кого подвинуть вниз
	// Если ниже никого — ничего не делаем
	if len(belowIndices) == 0 {
		return index
	}

	// Находим ближайшую строку снизу (минимальный start)
	closestBelow := belowIndices[0]
	closestStart, _ := getTieRange(m.Rows[closestBelow].Position)
	for _, idx := range belowIndices[1:] {
		start, _ := getTieRange(m.Rows[idx].Position)
		if start < closestStart {
			closestStart = start
			closestBelow = idx
		}
	}

	belowPos := m.Rows[closestBelow].Position
	belowStart, belowEnd := getTieRange(belowPos)

	// Мы одиночные
	if belowPos.TieCount == 0 {
		// Оба одиночные — создаём ничью
		newPos := model.Position{Int: currentStart, TieCount: 1}
		m.Rows[closestBelow].Position = newPos
		m.Rows[index].Position = newPos
	} else {
		// Мы одиночные, ниже ничья — расширяем ничью вниз
		newTieCount := belowEnd - currentStart
		newPos := model.Position{Int: currentStart, TieCount: newTieCount}
		// Обновляем все в нижней ничье
		for _, idx := range m.getRowsAtPosition(belowStart) {
			m.Rows[idx].Position = newPos
		}
		m.Rows[index].Position = newPos
	}

	m.sortRows()
	m.Modified = true
	return m.findRowIndex(rowName)
}
