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
