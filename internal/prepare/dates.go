package prepare

import (
	"time"
)

// addMonthOverflow добавляет/вычитает месяцы, корректируя день при переполнении
// При переполнении дня устанавливает последний день целевого месяца
func addMonthOverflow(t time.Time, months int) time.Time {
	result := t.AddDate(0, months, 0)

	// Если день изменился — значит был переполнение (например, 31 марта → 1 мая)
	// Возвращаем последний день целевого месяца
	if result.Day() != t.Day() {
		// Переходим на первое число следующего месяца и откатываем на 1 день
		return time.Date(result.Year(), result.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	}

	return result
}
