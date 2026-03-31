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
	// Создаём тестовую модель с пилотами на позициях 1, 2, 3
	m := &EventModel{
		Rows: []PilotRow{
			{Id: "p1", Position: model.Position{Int: 1}, Name: "Pilot1"},
			{Id: "p2", Position: model.Position{Int: 2}, Name: "Pilot2"},
			{Id: "p3", Position: model.Position{Int: 3}, Name: "Pilot3"},
			{Virtual: true, Position: model.Position{Int: 4}},
		},
	}

	// Перемещаем пилота 2 вверх — должна создаться ничья 1-2
	newIndex := m.MoveRowUp(1)

	// Проверяем, что пилот 2 теперь в ничье с пилотом 1
	if m.Rows[newIndex].Position.String() != "1-2" {
		t.Errorf("Expected position 1-2, got %s", m.Rows[newIndex].Position.String())
	}

	// Проверяем, что пилот 1 тоже в ничье 1-2
	p1Index := m.findRowIndex("Pilot1")
	if m.Rows[p1Index].Position.String() != "1-2" {
		t.Errorf("Expected Pilot1 position 1-2, got %s", m.Rows[p1Index].Position.String())
	}
}

func TestMoveRowUp_ExitTie(t *testing.T) {
	// Создаём тестовую модель с ничьёй 1-2
	m := &EventModel{
		Rows: []PilotRow{
			{Id: "p1", Position: model.Position{Int: 1, TieCount: 1}, Name: "Pilot1"},
			{Id: "p2", Position: model.Position{Int: 1, TieCount: 1}, Name: "Pilot2"},
			{Id: "p3", Position: model.Position{Int: 3}, Name: "Pilot3"},
			{Virtual: true, Position: model.Position{Int: 4}},
		},
	}

	// Перемещаем пилота 2 (из ничьи) вверх — он выходит из ничьи
	newIndex := m.MoveRowUp(1)

	// Проверяем, что пилот 2 теперь одиночный на позиции 1
	if m.Rows[newIndex].Position.String() != "1" {
		t.Errorf("Expected position 1, got %s", m.Rows[newIndex].Position.String())
	}

	// Проверяем, что пилот 1 теперь одиночный на позиции 2
	p1Index := m.findRowIndex("Pilot1")
	if m.Rows[p1Index].Position.String() != "2" {
		t.Errorf("Expected Pilot1 position 2, got %s", m.Rows[p1Index].Position.String())
	}
}

func TestMoveRowUp_ExitTieAtFirstPosition(t *testing.T) {
	// Ничья на первых местах — выход должен работать
	m := &EventModel{
		Rows: []PilotRow{
			{Id: "p1", Position: model.Position{Int: 1, TieCount: 1}, Name: "Pilot1"},
			{Id: "p2", Position: model.Position{Int: 1, TieCount: 1}, Name: "Pilot2"},
			{Virtual: true, Position: model.Position{Int: 3}},
		},
	}

	// Перемещаем пилота 2 вверх — он выходит из ничьи
	newIndex := m.MoveRowUp(1)

	// Проверяем, что пилот 2 теперь одиночный на позиции 1
	if m.Rows[newIndex].Position.String() != "1" {
		t.Errorf("Expected position 1, got %s", m.Rows[newIndex].Position.String())
	}

	// Проверяем, что пилот 1 теперь одиночный на позиции 2
	p1Index := m.findRowIndex("Pilot1")
	if m.Rows[p1Index].Position.String() != "2" {
		t.Errorf("Expected Pilot1 position 2, got %s", m.Rows[p1Index].Position.String())
	}
}

func TestMoveRowDown_SingleToTie(t *testing.T) {
	// Создаём тестовую модель с пилотами на позициях 1, 2, 3
	m := &EventModel{
		Rows: []PilotRow{
			{Id: "p1", Position: model.Position{Int: 1}, Name: "Pilot1"},
			{Id: "p2", Position: model.Position{Int: 2}, Name: "Pilot2"},
			{Id: "p3", Position: model.Position{Int: 3}, Name: "Pilot3"},
			{Virtual: true, Position: model.Position{Int: 4}},
		},
	}

	// Перемещаем пилота 2 вниз — должна создаться ничья 2-3
	newIndex := m.MoveRowDown(1)

	// Проверяем, что пилот 2 теперь в ничье с пилотом 3
	if m.Rows[newIndex].Position.String() != "2-3" {
		t.Errorf("Expected position 2-3, got %s", m.Rows[newIndex].Position.String())
	}

	// Проверяем, что пилот 3 тоже в ничье 2-3
	p3Index := m.findRowIndex("Pilot3")
	if m.Rows[p3Index].Position.String() != "2-3" {
		t.Errorf("Expected Pilot3 position 2-3, got %s", m.Rows[p3Index].Position.String())
	}
}

func TestMoveRowDown_ExitTie(t *testing.T) {
	// Создаём тестовую модель с ничьёй 2-3
	m := &EventModel{
		Rows: []PilotRow{
			{Id: "p1", Position: model.Position{Int: 1}, Name: "Pilot1"},
			{Id: "p2", Position: model.Position{Int: 2, TieCount: 1}, Name: "Pilot2"},
			{Id: "p3", Position: model.Position{Int: 2, TieCount: 1}, Name: "Pilot3"},
			{Virtual: true, Position: model.Position{Int: 4}},
		},
	}

	// Перемещаем пилота 2 (из ничьи) вниз — он выходит из ничьи
	newIndex := m.MoveRowDown(1)

	// Проверяем, что пилот 2 теперь одиночный на позиции 3
	if m.Rows[newIndex].Position.String() != "3" {
		t.Errorf("Expected position 3, got %s", m.Rows[newIndex].Position.String())
	}

	// Проверяем, что пилот 3 остался на позиции 2
	p3Index := m.findRowIndex("Pilot3")
	if m.Rows[p3Index].Position.String() != "2" {
		t.Errorf("Expected Pilot3 position 2, got %s", m.Rows[p3Index].Position.String())
	}
}

func TestMoveRowDown_ExitTieAtLastPosition(t *testing.T) {
	// Ничья на последних местах — выход должен работать
	m := &EventModel{
		Rows: []PilotRow{
			{Id: "p1", Position: model.Position{Int: 1}, Name: "Pilot1"},
			{Id: "p2", Position: model.Position{Int: 2, TieCount: 1}, Name: "Pilot2"},
			{Id: "p3", Position: model.Position{Int: 2, TieCount: 1}, Name: "Pilot3"},
			{Virtual: true, Position: model.Position{Int: 4}},
		},
	}

	// Перемещаем пилота 2 вниз — он выходит из ничьи
	newIndex := m.MoveRowDown(1)

	// Проверяем, что пилот 2 теперь одиночный на позиции 3
	if m.Rows[newIndex].Position.String() != "3" {
		t.Errorf("Expected position 3, got %s", m.Rows[newIndex].Position.String())
	}
}

func TestMoveRowUp_TripleTieExit(t *testing.T) {
	// Ничья из трёх человек — один выходит
	m := &EventModel{
		Rows: []PilotRow{
			{Id: "p1", Position: model.Position{Int: 1, TieCount: 2}, Name: "Pilot1"},
			{Id: "p2", Position: model.Position{Int: 1, TieCount: 2}, Name: "Pilot2"},
			{Id: "p3", Position: model.Position{Int: 1, TieCount: 2}, Name: "Pilot3"},
			{Virtual: true, Position: model.Position{Int: 4}},
		},
	}

	// Перемещаем пилота 2 (из ничьи) вверх
	newIndex := m.MoveRowUp(1)

	// Проверяем, что пилот 2 теперь одиночный на позиции 1
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
