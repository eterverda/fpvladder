package prepare

import (
	"testing"

	"github.com/eterverda/fpvladder/internal/model"
)

func TestGetTieRange(t *testing.T) {
	tests := []struct {
		pos       model.Position
		wantStart int
		wantEnd   int
	}{
		{model.Position{Int: 1}, 1, 1},
		{model.Position{Int: 5, TieCount: 0}, 5, 5},
		{model.Position{Int: 7, TieCount: 1}, 7, 8},
		{model.Position{Int: 11, TieCount: 2}, 11, 13},
		{model.Position{Int: 3, TieCount: 3}, 3, 6},
	}

	for _, tt := range tests {
		t.Run(tt.pos.String(), func(t *testing.T) {
			start, end := getTieRange(tt.pos)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("getTieRange(%v) = (%d, %d), want (%d, %d)",
					tt.pos, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestMoveRowUp_SingleToTie(t *testing.T) {
	// Создаём тестовую модель с командами на позициях 1, 2, 3
	m := &EventModel{
		Rows: []TeamRow{
			{Pilots: []PilotRow{{Id: "p1", Name: "Pilot1"}}, Position: model.Position{Int: 1}},
			{Pilots: []PilotRow{{Id: "p2", Name: "Pilot2"}}, Position: model.Position{Int: 2}},
			{Pilots: []PilotRow{{Id: "p3", Name: "Pilot3"}}, Position: model.Position{Int: 3}},
			{Virtual: true, Position: model.Position{Int: 4}},
		},
	}

	// Перемещаем команду 2 вверх — должна создаться ничья 1-2
	newIndex := m.MoveRowUp(1)

	// Проверяем, что команда 2 теперь в ничье с командой 1
	if m.Rows[newIndex].Position.String() != "1-2" {
		t.Errorf("Expected position 1-2, got %s", m.Rows[newIndex].Position.String())
	}

	// Проверяем, что команда 1 тоже в ничье 1-2
	p1Index := m.findRowIndex("Pilot1")
	if m.Rows[p1Index].Position.String() != "1-2" {
		t.Errorf("Expected Pilot1 position 1-2, got %s", m.Rows[p1Index].Position.String())
	}
}

func TestMoveRowUp_ExitTie(t *testing.T) {
	// Создаём тестовую модель с ничьёй 1-2
	m := &EventModel{
		Rows: []TeamRow{
			{Pilots: []PilotRow{{Id: "p1", Name: "Pilot1"}}, Position: model.Position{Int: 1, TieCount: 1}},
			{Pilots: []PilotRow{{Id: "p2", Name: "Pilot2"}}, Position: model.Position{Int: 1, TieCount: 1}},
			{Pilots: []PilotRow{{Id: "p3", Name: "Pilot3"}}, Position: model.Position{Int: 3}},
			{Virtual: true, Position: model.Position{Int: 4}},
		},
	}

	// Перемещаем команду 2 (из ничьи) вверх — она выходит из ничьи
	newIndex := m.MoveRowUp(1)

	// Проверяем, что команда 2 теперь одиночная на позиции 1
	if m.Rows[newIndex].Position.String() != "1" {
		t.Errorf("Expected position 1, got %s", m.Rows[newIndex].Position.String())
	}

	// Проверяем, что команда 1 теперь одиночная на позиции 2
	p1Index := m.findRowIndex("Pilot1")
	if m.Rows[p1Index].Position.String() != "2" {
		t.Errorf("Expected Pilot1 position 2, got %s", m.Rows[p1Index].Position.String())
	}
}

func TestMoveRowUp_ExitTieAtFirstPosition(t *testing.T) {
	// Ничья на первых местах — выход должен работать
	m := &EventModel{
		Rows: []TeamRow{
			{Pilots: []PilotRow{{Id: "p1", Name: "Pilot1"}}, Position: model.Position{Int: 1, TieCount: 1}},
			{Pilots: []PilotRow{{Id: "p2", Name: "Pilot2"}}, Position: model.Position{Int: 1, TieCount: 1}},
			{Virtual: true, Position: model.Position{Int: 3}},
		},
	}

	// Перемещаем команду 2 вверх — она выходит из ничьи
	newIndex := m.MoveRowUp(1)

	// Проверяем, что команда 2 теперь одиночная на позиции 1
	if m.Rows[newIndex].Position.String() != "1" {
		t.Errorf("Expected position 1, got %s", m.Rows[newIndex].Position.String())
	}

	// Проверяем, что команда 1 теперь одиночная на позиции 2
	p1Index := m.findRowIndex("Pilot1")
	if m.Rows[p1Index].Position.String() != "2" {
		t.Errorf("Expected Pilot1 position 2, got %s", m.Rows[p1Index].Position.String())
	}
}

func TestMoveRowDown_SingleToTie(t *testing.T) {
	// Создаём тестовую модель с командами на позициях 1, 2, 3
	m := &EventModel{
		Rows: []TeamRow{
			{Pilots: []PilotRow{{Id: "p1", Name: "Pilot1"}}, Position: model.Position{Int: 1}},
			{Pilots: []PilotRow{{Id: "p2", Name: "Pilot2"}}, Position: model.Position{Int: 2}},
			{Pilots: []PilotRow{{Id: "p3", Name: "Pilot3"}}, Position: model.Position{Int: 3}},
			{Virtual: true, Position: model.Position{Int: 4}},
		},
	}

	// Перемещаем команду 2 вниз — должна создаться ничья 2-3
	newIndex := m.MoveRowDown(1)

	// Проверяем, что команда 2 теперь в ничье с командой 3
	if m.Rows[newIndex].Position.String() != "2-3" {
		t.Errorf("Expected position 2-3, got %s", m.Rows[newIndex].Position.String())
	}

	// Проверяем, что команда 3 тоже в ничье 2-3
	p3Index := m.findRowIndex("Pilot3")
	if m.Rows[p3Index].Position.String() != "2-3" {
		t.Errorf("Expected Pilot3 position 2-3, got %s", m.Rows[p3Index].Position.String())
	}
}

func TestMoveRowDown_ExitTie(t *testing.T) {
	// Создаём тестовую модель с ничьёй 2-3
	m := &EventModel{
		Rows: []TeamRow{
			{Pilots: []PilotRow{{Id: "p1", Name: "Pilot1"}}, Position: model.Position{Int: 1}},
			{Pilots: []PilotRow{{Id: "p2", Name: "Pilot2"}}, Position: model.Position{Int: 2, TieCount: 1}},
			{Pilots: []PilotRow{{Id: "p3", Name: "Pilot3"}}, Position: model.Position{Int: 2, TieCount: 1}},
			{Virtual: true, Position: model.Position{Int: 4}},
		},
	}

	// Перемещаем команду 2 (из ничьи) вниз — она выходит из ничьи
	newIndex := m.MoveRowDown(1)

	// Проверяем, что команда 2 теперь одиночная на позиции 3
	if m.Rows[newIndex].Position.String() != "3" {
		t.Errorf("Expected position 3, got %s", m.Rows[newIndex].Position.String())
	}

	// Проверяем, что команда 3 осталась на позиции 2
	p3Index := m.findRowIndex("Pilot3")
	if m.Rows[p3Index].Position.String() != "2" {
		t.Errorf("Expected Pilot3 position 2, got %s", m.Rows[p3Index].Position.String())
	}
}

func TestMoveRowDown_ExitTieAtLastPosition(t *testing.T) {
	// Ничья на последних местах — выход должен работать
	m := &EventModel{
		Rows: []TeamRow{
			{Pilots: []PilotRow{{Id: "p1", Name: "Pilot1"}}, Position: model.Position{Int: 1}},
			{Pilots: []PilotRow{{Id: "p2", Name: "Pilot2"}}, Position: model.Position{Int: 2, TieCount: 1}},
			{Pilots: []PilotRow{{Id: "p3", Name: "Pilot3"}}, Position: model.Position{Int: 2, TieCount: 1}},
			{Virtual: true, Position: model.Position{Int: 4}},
		},
	}

	// Перемещаем команду 2 вниз — она выходит из ничьи
	newIndex := m.MoveRowDown(1)

	// Проверяем, что команда 2 теперь одиночная на позиции 3
	if m.Rows[newIndex].Position.String() != "3" {
		t.Errorf("Expected position 3, got %s", m.Rows[newIndex].Position.String())
	}
}

func TestMoveRowUp_TripleTieExit(t *testing.T) {
	// Ничья из трёх команд — одна выходит
	m := &EventModel{
		Rows: []TeamRow{
			{Pilots: []PilotRow{{Id: "p1", Name: "Pilot1"}}, Position: model.Position{Int: 1, TieCount: 2}},
			{Pilots: []PilotRow{{Id: "p2", Name: "Pilot2"}}, Position: model.Position{Int: 1, TieCount: 2}},
			{Pilots: []PilotRow{{Id: "p3", Name: "Pilot3"}}, Position: model.Position{Int: 1, TieCount: 2}},
			{Virtual: true, Position: model.Position{Int: 4}},
		},
	}

	// Перемещаем команду 2 (из ничьи) вверх
	newIndex := m.MoveRowUp(1)

	// Проверяем, что команда 2 теперь одиночная на позиции 1
	if m.Rows[newIndex].Position.String() != "1" {
		t.Errorf("Expected position 1, got %s", m.Rows[newIndex].Position.String())
	}

	// Проверяем, что оставшиеся в ничье 2-3
	p1Index := m.findRowIndex("Pilot1")
	p3Index := m.findRowIndex("Pilot3")
	if m.Rows[p1Index].Position.String() != "2-3" {
		t.Errorf("Expected Pilot1 position 2-3, got %s", m.Rows[p1Index].Position.String())
	}
	if m.Rows[p3Index].Position.String() != "2-3" {
		t.Errorf("Expected Pilot3 position 2-3, got %s", m.Rows[p3Index].Position.String())
	}
}
