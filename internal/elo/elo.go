package elo

import (
	"fmt"
	"math"
)

const (
	K = 14.0
)

var (
	Algorithm = fmt.Sprintf("elo > k-%v", K)
)

type Input struct {
	Position int // Позиция (значение из model.Position.Int)
	Team     int // Номер команды (0 = нет команды)
	Rating   int
}

// GroupKCalc рассчитывает изменения рейтинга для группы пилотов
// Учитывает ничьи между разными командами
func GroupKCalc(inputs []Input) (deltas []int) {
	deltasF := GroupKCalcF(inputs)

	deltas = make([]int, len(inputs))
	for i := range deltas {
		deltas[i] = int(math.Round(deltasF[i]))
	}
	return
}

// GroupKCalcF рассчитывает сырые изменения рейтинга для группы пилотов
func GroupKCalcF(inputs []Input) (deltas []float64) {
	deltas = make([]float64, len(inputs))

	for i := range inputs {
		for j := i + 1; j < len(inputs); j++ {
			// Пропускаем пилотов из одной команды (если команда указана)
			if inputs[i].Team > 0 && inputs[i].Team == inputs[j].Team {
				continue
			}

			// Используем KCalcF для расчёта дуэли
			deltaA, deltaB := KCalcF(inputs[i], inputs[j])
			deltas[i] += deltaA
			deltas[j] += deltaB
		}
	}
	return
}

// KCalc рассчитывает изменения рейтинга для двух пилотов
func KCalc(a, b Input) (deltaA, deltaB int) {
	deltaAF, deltaBF := KCalcF(a, b)
	return int(math.Round(deltaAF)), int(math.Round(deltaBF))
}

// KCalcF рассчитывает сырое изменение рейтинга для двух пилотов (с K)
func KCalcF(a, b Input) (deltaA, deltaB float64) {
	deltaA, deltaB = calcWinLossF(a, b)
	return K * deltaA, K * deltaB
}

// calcWinLossF рассчитывает сырую дельту (без K)
// Обрабатывает победу, поражение и ничью
func calcWinLossF(a, b Input) (deltaA, deltaB float64) {
	// 1. Ожидание: насколько мы сильнее соперника
	exp := 1.0 / (1.0 + math.Pow(10, float64(b.Rating-a.Rating)/400.0))

	// 2. Реальность
	var act float64
	if a.Position == b.Position {
		// Ничья: результат = 0.5
		act = 0.5
	} else if a.Position < b.Position {
		// Победа
		act = 1.0
	} else {
		// Поражение
		act = 0.0
	}

	return act - exp, exp - act
}
