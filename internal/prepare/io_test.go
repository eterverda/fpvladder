package prepare

import (
	"strings"
	"testing"

	"github.com/eterverda/fpvladder/internal/model"
)

func TestValidatePositions_Valid(t *testing.T) {
	tests := []struct {
		name string
		rows []TeamRow
	}{
		{
			name: "последовательные позиции без ничьих",
			rows: []TeamRow{
				{Pilots: []PilotRow{{Name: "П1"}}, Position: model.Position{Int: 1}},
				{Pilots: []PilotRow{{Name: "П2"}}, Position: model.Position{Int: 2}},
				{Pilots: []PilotRow{{Name: "П3"}}, Position: model.Position{Int: 3}},
			},
		},
		{
			name: "ничья из двух",
			rows: []TeamRow{
				{Pilots: []PilotRow{{Name: "П1"}}, Position: model.Position{Int: 1, TieCount: 1}},
				{Pilots: []PilotRow{{Name: "П2"}}, Position: model.Position{Int: 1, TieCount: 1}},
				{Pilots: []PilotRow{{Name: "П3"}}, Position: model.Position{Int: 3}},
			},
		},
		{
			name: "ничья из трех",
			rows: []TeamRow{
				{Pilots: []PilotRow{{Name: "П1"}}, Position: model.Position{Int: 1, TieCount: 2}},
				{Pilots: []PilotRow{{Name: "П2"}}, Position: model.Position{Int: 1, TieCount: 2}},
				{Pilots: []PilotRow{{Name: "П3"}}, Position: model.Position{Int: 1, TieCount: 2}},
				{Pilots: []PilotRow{{Name: "П4"}}, Position: model.Position{Int: 4}},
			},
		},
		{
			name: "две ничьи подряд",
			rows: []TeamRow{
				{Pilots: []PilotRow{{Name: "П1"}}, Position: model.Position{Int: 1, TieCount: 1}},
				{Pilots: []PilotRow{{Name: "П2"}}, Position: model.Position{Int: 1, TieCount: 1}},
				{Pilots: []PilotRow{{Name: "П3"}}, Position: model.Position{Int: 3, TieCount: 1}},
				{Pilots: []PilotRow{{Name: "П4"}}, Position: model.Position{Int: 3, TieCount: 1}},
				{Pilots: []PilotRow{{Name: "П5"}}, Position: model.Position{Int: 5}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EventModel{Rows: tt.rows}
			if err := m.validatePositions(); err != nil {
				t.Errorf("validatePositions() ошибка = %v", err)
			}
		})
	}
}

func TestValidatePositions_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		rows    []TeamRow
		wantErr string
	}{
		{
			name: "пропуск позиции",
			rows: []TeamRow{
				{Pilots: []PilotRow{{Name: "П1"}}, Position: model.Position{Int: 1}},
				{Pilots: []PilotRow{{Name: "П2"}}, Position: model.Position{Int: 3}},
			},
			wantErr: "пропущена позиция 2",
		},
		{
			name: "неправильный TieCount в ничьей из двух",
			rows: []TeamRow{
				{Pilots: []PilotRow{{Name: "П1"}}, Position: model.Position{Int: 1, TieCount: 0}},
				{Pilots: []PilotRow{{Name: "П2"}}, Position: model.Position{Int: 1, TieCount: 0}},
			},
			wantErr: "неправильно оформлена ничья",
		},
		{
			name: "неправильный TieCount в ничьей из трех",
			rows: []TeamRow{
				{Pilots: []PilotRow{{Name: "П1"}}, Position: model.Position{Int: 1, TieCount: 1}},
				{Pilots: []PilotRow{{Name: "П2"}}, Position: model.Position{Int: 1, TieCount: 1}},
				{Pilots: []PilotRow{{Name: "П3"}}, Position: model.Position{Int: 1, TieCount: 1}},
			},
			wantErr: "неправильно оформлена ничья",
		},
		{
			name: "TieCount у одиночной позиции",
			rows: []TeamRow{
				{Pilots: []PilotRow{{Name: "П1"}}, Position: model.Position{Int: 1, TieCount: 1}},
			},
			wantErr: "ожидается 0 для одиночной позиции",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EventModel{Rows: tt.rows}
			err := m.validatePositions()
			if err == nil {
				t.Error("validatePositions() ожидалась ошибка, но её не было")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("validatePositions() ошибка = %v, ожидалось содержание %q", err, tt.wantErr)
			}
		})
	}
}
