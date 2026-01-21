package main

import "math"

const K = 30.0

type pilotInput struct {
	position       int
	oldRatingValue int
}

func recalculateRating(inputs []pilotInput, index int) (delta int) {
	subject := inputs[index]

	actualTotal := 0.0
	expectedTotal := 0.0

	for i, other := range inputs {
		// Пропускаем себя И напарников по команде (тех, у кого такая же позиция)
		if i == index || subject.position == other.position {
			continue
		}

		// 1. Ожидание: насколько мы сильнее соперника из ДРУГОЙ команды
		exp := 1.0 / (1.0 + math.Pow(10, float64(other.oldRatingValue-subject.oldRatingValue)/400.0))
		expectedTotal += exp

		// 2. Реальность: обошли мы их или они нас
		var actual float64
		if subject.position < other.position {
			actual = 1.0 // Победа над чужой командой
		} else {
			actual = 0.0 // Поражение от чужой команды
		}
		actualTotal += actual
	}

	// Изменение рейтинга только на основе матчей с внешними соперниками
	return int(math.Round(K * (actualTotal - expectedTotal)))
}
