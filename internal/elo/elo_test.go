package elo

import (
	"testing"
)

// TestEloScenarios проверяет расчёт рейтинга для разных сценариев
// Сильный и слабый игроки, разные исходы, и разные начальные рейтинги
func TestEloScenarios(t *testing.T) {
	tests := []struct {
		name           string
		ratingA        int    // рейтинг сильного (выше)
		outcome        string // "win", "loss", "tie"
		ratingB        int    // рейтинг слабого (ниже)
		expectedADelta int    // ожидаемое изменение рейтинга A
		expectedBDelta int    // ожидаемое изменение рейтинга B
	}{
		// Оба выше среднего (1500 vs 1300, разница 200)
		{"оба_высокие_победа_сильного", 1500, "win", 1300, 3, -3},
		{"оба_высокие_поражение_сильного", 1500, "loss", 1300, -11, 11},
		{"оба_высокие_ничья", 1500, "tie", 1300, -4, 4},

		// Большая разница (1600 vs 800, разница 800)
		{"большая_разница_победа_сильного", 1600, "win", 800, 0, 0}, // изменения меньше единицы
		{"большая_разница_поражение_сильного", 1600, "loss", 800, -14, 14},
		{"большая_разница_ничья", 1600, "tie", 800, -7, 7},

		// Средняя разница (1400 vs 1000, разница 400)
		{"средняя_разница_победа_сильного", 1400, "win", 1000, 1, -1},
		{"средняя_разница_поражение_сильного", 1400, "loss", 1000, -13, 13},
		{"средняя_разница_ничья", 1400, "tie", 1000, -6, 6},

		// Оба ниже среднего (1100 vs 900, разница 200)
		{"оба_низкие_победа_сильного", 1100, "win", 900, 3, -3},
		{"оба_низкие_поражение_сильного", 1100, "loss", 900, -11, 11},
		{"оба_низкие_ничья", 1100, "tie", 900, -4, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a, b Input
			a.Rating = tt.ratingA
			b.Rating = tt.ratingB

			switch tt.outcome {
			case "win":
				a.Position = 1
				b.Position = 2
			case "loss":
				a.Position = 2
				b.Position = 1
			case "tie":
				a.Position = 1
				b.Position = 1
			}

			deltaA, deltaB := KCalc(a, b)

			t.Logf("A: %d, outcome: %s, B: %d -> deltaA: %d, deltaB: %d",
				tt.ratingA, tt.outcome, tt.ratingB, deltaA, deltaB)

			// Проверяем, что сумма дельт = 0 (сохранение очков)
			if deltaA+deltaB != 0 {
				t.Errorf("сумма дельт != 0: %d + %d = %d", deltaA, deltaB, deltaA+deltaB)
			}

			// Проверяем направление изменений
			switch tt.outcome {
			case "win":
				if deltaA < 0 {
					t.Errorf("сильный победил, но потерял очки: %d", deltaA)
				}
				if deltaB > 0 {
					t.Errorf("слабый проиграл, но получил очки: %d", deltaB)
				}
			case "loss":
				if deltaA > 0 {
					t.Errorf("сильный проиграл, но получил очки: %d", deltaA)
				}
				if deltaB < 0 {
					t.Errorf("слабый победил, но потерял очки: %d", deltaB)
				}
			case "tie":
				if deltaA >= 0 {
					t.Errorf("сильный сыграл вничью со слабым, но не потерял очки: %d", deltaA)
				}
				if deltaB <= 0 {
					t.Errorf("слабый сыграл вничью с сильным, но не получил очки: %d", deltaB)
				}
			}
		})
	}
}

// TestTeamZeroNotTeammate проверяет, что Team: 0 не считается командой
// Игроки с Team: 0 должны играть друг с другом (не пропускаются как сокомандники)
func TestTeamZeroNotTeammate(t *testing.T) {
	// Три игрока, все с Team: 0 (нет команды)
	// Они должны играть друг с другом
	inputs := []Input{
		{Position: 1, Team: 0, Rating: 1200},
		{Position: 2, Team: 0, Rating: 1200},
		{Position: 3, Team: 0, Rating: 1200},
	}

	deltas := GroupKCalc(inputs)

	// При одинаковом рейтинге 1200 и позициях 1, 2, 3:
	// - Игрок 1: +7 (vs 2) +7 (vs 3) = +14
	// - Игрок 2: -7 (vs 1) +7 (vs 3) = 0
	// - Игрок 3: -7 (vs 1) -7 (vs 2) = -14
	// Игрок 2 имеет 0, потому что проиграл первому, но выиграл у третьего
	// Это правильное поведение ELO, а не признак того, что они не играли

	// Проверяем, что сумма дельт = 0 (сохранение очков)
	sum := 0
	for _, d := range deltas {
		sum += d
	}
	if sum != 0 {
		t.Errorf("Сумма дельт != 0: %d", sum)
	}

	// Проверяем, что первый получил положительную дельту (победитель)
	if deltas[0] <= 0 {
		t.Errorf("Первый место должно иметь положительную дельту: %d", deltas[0])
	}

	// Проверяем, что последний получил отрицательную дельту (проигравший)
	if deltas[2] >= 0 {
		t.Errorf("Последнее место должно иметь отрицательную дельту: %d", deltas[2])
	}

	t.Logf("Дельты для трёх игроков с Team:0: %v", deltas)
}

// TestSameTeamSkipped проверяет, что игроки из одной команды (Team > 0) не играют друг с другом
func TestSameTeamSkipped(t *testing.T) {
	// Четыре игрока: два в команде 1, два в команде 2
	inputs := []Input{
		{Position: 1, Team: 1, Rating: 1200}, // Команда 1
		{Position: 2, Team: 1, Rating: 1200}, // Команда 1 (сокомандник)
		{Position: 3, Team: 2, Rating: 1200}, // Команда 2
		{Position: 4, Team: 2, Rating: 1200}, // Команда 2 (сокомандник)
	}

	deltas := GroupKCalc(inputs)

	// Игроки из одной команды должны иметь одинаковые дельты
	// (они не играли друг с другом, только с соперниками из другой команды)
	if deltas[0] != deltas[1] {
		t.Errorf("Игроки из команды 1 имеют разные дельты: %d vs %d", deltas[0], deltas[1])
	}
	if deltas[2] != deltas[3] {
		t.Errorf("Игроки из команды 2 имеют разные дельты: %d vs %d", deltas[2], deltas[3])
	}

	t.Logf("Дельты для команд (1,1,2,2): %v", deltas)
}

// TestMixedTeamAndNoTeam проверяет смешанный случай: Team:0 и реальные команды
func TestMixedTeamAndNoTeam(t *testing.T) {
	// Четыре игрока: без команды, с командой 1, без команды, с командой 2
	inputs := []Input{
		{Position: 1, Team: 0, Rating: 1200}, // Нет команды
		{Position: 2, Team: 1, Rating: 1200}, // Команда 1
		{Position: 3, Team: 0, Rating: 1200}, // Нет команды
		{Position: 4, Team: 2, Rating: 1200}, // Команда 2
	}

	deltas := GroupKCalc(inputs)

	// Все должны иметь ненулевые дельты, т.к.:
	// - Team:0 играет со всеми (включая других Team:0)
	// - Team:1 играет с Team:0 и Team:2, но не с самим собой
	// - Team:2 играет с Team:0 и Team:1, но не с самим собой
	for i, d := range deltas {
		if d == 0 {
			t.Errorf("Игрок %d (Team:%d) имеет нулевую дельту", i, inputs[i].Team)
		}
	}

	t.Logf("Дельты для (Team:0,1,0,2): %v", deltas)
}
