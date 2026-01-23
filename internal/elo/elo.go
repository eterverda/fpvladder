package elo

import "math"

const (
	K = 30.0
)

type Input struct {
	Position int
	Rating   int
}

func GroupKCalc(inputs []Input) (deltas []int) {
	deltasF := GroupKCalcF(inputs)

	deltas = make([]int, len(inputs))
	for i := range deltas {
		deltas[i] = int(math.Round(deltasF[i]))
	}
	return
}

func GroupKCalcF(inputs []Input) (deltas []float64) {
	deltas = make([]float64, len(inputs))

	for i := range inputs {
		a := inputs[i]
		for j := i + 1; j < len(inputs); j++ {
			b := inputs[j]
			// Пропускаем напарников по команде (одинаковая позиция)
			if a.Position == b.Position {
				continue
			}

			// Считаем микро-дельту дуэли i против j
			deltaA, deltaB := KCalcF(a, b)
			deltas[i] += deltaA
			deltas[j] += deltaB
		}
	}
	return
}

func KCalc(a, b Input) (deltaA, deltaB int) {
	deltaAF, deltaBF := KCalcF(a, b)
	return int(math.Round(deltaAF)), int(math.Round(deltaBF))
}

func KCalcF(a, b Input) (deltaA, deltaB float64) {
	deltaAF, deltaBF := CalcF(a, b)
	return K * deltaAF, K * deltaBF
}

func CalcF(a, b Input) (deltaA, deltaB float64) {
	if a.Position == b.Position {
		return 0.0, 0.0
	}
	// 1. Ожидание: насколько мы сильнее соперника из ДРУГОЙ команды
	exp := 1.0 / (1.0 + math.Pow(10, float64(b.Rating-a.Rating)/400.0))

	// 2. Реальность: обошли мы их или они нас
	var act float64
	if a.Position < b.Position {
		act = 1.0 // Победа
	} else {
		act = 0.0 // Поражение
	}
	return act - exp, exp - act
}
