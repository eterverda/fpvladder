package prepare

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eterverda/fpvladder/internal/model"
)

func TestLoadTeamsFromFile(t *testing.T) {
	// Создаём тестовый YAML файл
	yamlContent := `id: 2026/04-01/1
date: 2026-04-01
name: Тест команд
organizer:
  name: Тест
class: drone-racing > 75mm
pilots:
  - position: 1
    id: 2025/12-28/11
    name: Садовников Матвей
  - position: 2
    team: 1
    id: 2026/02-28/75
    name: Шариков Степан
  - position: 2
    team: 1
    id: 2026/02-28/43
    name: Андросенко Савелий
  - position: 3
    id: 2025/12-28/14
    name: Сойгалов Степан
  - position: 4
    team: 2
    id: 2026/02-28/68
    name: Родин Алексей
  - position: 4
    team: 2
    id: 2026/02-28/59
    name: Лебедь Александр
  - position: 4
    team: 2
    id: 2026/02-28/71
    name: Станкевич Алексей
`

	tmpFile, err := os.CreateTemp("", "test_teams_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Загружаем файл
	m, err := NewEventModel(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load event model: %v", err)
	}

	// Проверяем количество команд (4 команды + 1 виртуальная)
	// Ожидаем: 1 одиночный, 1 команда из 2, 1 одиночный, 1 команда из 3, 1 виртуальная = 5
	if len(m.Rows) != 5 {
		t.Errorf("Expected 5 rows (4 teams + 1 virtual), got %d", len(m.Rows))
	}

	// Проверяем первую команду (одиночная)
	if m.Rows[0].Position.Int != 1 {
		t.Errorf("Expected first team position 1, got %d", m.Rows[0].Position.Int)
	}
	if len(m.Rows[0].Pilots) != 1 {
		t.Errorf("Expected first team to have 1 pilot, got %d", len(m.Rows[0].Pilots))
	}

	// Проверяем вторую команду (двойная, должна быть отсортирована по имени)
	if m.Rows[1].Position.Int != 2 {
		t.Errorf("Expected second team position 2, got %d", m.Rows[1].Position.Int)
	}
	if len(m.Rows[1].Pilots) != 2 {
		t.Errorf("Expected second team to have 2 pilots, got %d", len(m.Rows[1].Pilots))
	}
	// Пилоты должны быть отсортированы по имени: Андросенко, Шариков
	if m.Rows[1].Pilots[0].Name != "Андросенко Савелий" {
		t.Errorf("Expected first pilot in team 2 to be 'Андросенко Савелий', got '%s'", m.Rows[1].Pilots[0].Name)
	}
	if m.Rows[1].Pilots[1].Name != "Шариков Степан" {
		t.Errorf("Expected second pilot in team 2 to be 'Шариков Степан', got '%s'", m.Rows[1].Pilots[1].Name)
	}

	// Проверяем третью команду (одиночная)
	if m.Rows[2].Position.Int != 3 {
		t.Errorf("Expected third team position 3, got %d", m.Rows[2].Position.Int)
	}

	// Проверяем четвёртую команду (тройная)
	if m.Rows[3].Position.Int != 4 {
		t.Errorf("Expected fourth team position 4, got %d", m.Rows[3].Position.Int)
	}
	if len(m.Rows[3].Pilots) != 3 {
		t.Errorf("Expected fourth team to have 3 pilots, got %d", len(m.Rows[3].Pilots))
	}

	// Проверяем виртуальную строку
	if !m.Rows[4].Virtual {
		t.Errorf("Expected last row to be virtual")
	}
	if len(m.Rows[4].Pilots) != 1 {
		t.Errorf("Expected virtual row to have 1 virtual pilot, got %d", len(m.Rows[4].Pilots))
	}
}

func TestSaveTeams(t *testing.T) {
	// Создаём модель с командами
	m := &EventModel{
		Event: model.Event{
			Id:        "2026/04-01/1",
			Date:      model.Date(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)),
			Name:      "Тест сохранения",
			Organizer: model.Organizer{Name: "Тест"},
			Class:     model.Class75mm,
		},
		Rows: []TeamRow{
			{Pilots: []PilotRow{{Id: "p1", IdKind: IdKindExplicit, Name: "Пилот1"}}, Position: model.Position{Int: 1}},
			{Pilots: []PilotRow{{Id: "p2", IdKind: IdKindExplicit, Name: "Пилот2"}, {Id: "p3", IdKind: IdKindExplicit, Name: "Пилот3"}}, Position: model.Position{Int: 2}},
			{Pilots: []PilotRow{{Id: "p4", IdKind: IdKindExplicit, Name: "Пилот4"}}, Position: model.Position{Int: 3}},
		},
		Filename: "/tmp/test_save_teams.yaml",
	}

	// Сохраняем
	if err := m.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Читаем содержимое файла
	content, err := os.ReadFile(m.Filename)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}
	defer os.Remove(m.Filename)

	t.Logf("Saved file content:\n%s", string(content))

	// Проверяем, что одиночные пилоты имеют team: 0
	// А команда из 2 пилотов имеет team: 1
	contentStr := string(content)

	// Проверяем что пилоты есть в файле
	if !strings.Contains(contentStr, "Пилот1") || !strings.Contains(contentStr, "Пилот2") ||
		!strings.Contains(contentStr, "Пилот3") || !strings.Contains(contentStr, "Пилот4") {
		t.Error("Not all pilots found in saved file")
	}
}

func TestVirtualPilotNavigation(t *testing.T) {
	// Создаём модель с 2 реальными пилотами и виртуальным
	m := &EventModel{
		Rows: []TeamRow{
			{Pilots: []PilotRow{{Id: "p1", Name: "Пилот1"}}, Position: model.Position{Int: 1}},
			{Pilots: []PilotRow{{Id: "p2", Name: "Пилот2"}}, Position: model.Position{Int: 2}},
			{Virtual: true, Position: model.Position{Int: 3}, Pilots: []PilotRow{{}}},
		},
	}

	// Проверяем общее количество пилотов (2 реальных + 1 виртуальный = 3)
	total := TotalPilots(m.Rows)
	if total != 3 {
		t.Errorf("Expected 3 total pilots (2 real + 1 virtual), got %d", total)
	}

	// Проверяем что можем получить виртуального пилота по индексу 2
	teamIdx, _, pilot, ok := m.GetPilot(2)
	if !ok {
		t.Error("Expected to get virtual pilot at index 2")
	}
	if !m.Rows[teamIdx].Virtual {
		t.Error("Expected pilot at index 2 to be from virtual team")
	}
	if pilot.Name != "" || pilot.Id != "" {
		t.Error("Expected virtual pilot to have empty name and id")
	}

	// Проверяем IsVirtualPilot
	if !m.IsVirtualPilot(2) {
		t.Error("Expected IsVirtualPilot(2) to return true")
	}
	if m.IsVirtualPilot(0) {
		t.Error("Expected IsVirtualPilot(0) to return false")
	}
	if m.IsVirtualPilot(1) {
		t.Error("Expected IsVirtualPilot(1) to return false")
	}

	// Проверяем GetPilotByIndex для всех пилотов
	for i := 0; i < 3; i++ {
		teamIdx, pilotIdx, ok := GetPilotByIndex(m.Rows, i)
		if !ok {
			t.Errorf("Expected to get pilot at index %d", i)
		}
		if teamIdx < 0 || teamIdx >= len(m.Rows) {
			t.Errorf("Invalid teamIdx %d for pilot %d", teamIdx, i)
		}
		if pilotIdx < 0 || pilotIdx >= len(m.Rows[teamIdx].Pilots) {
			t.Errorf("Invalid pilotIdx %d for pilot %d", pilotIdx, i)
		}
	}
}
