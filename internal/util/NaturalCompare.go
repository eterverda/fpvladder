package util

import (
	"unicode"
)

// NaturalCompare сравнивает строки "естественно": числа сравниваются как числа.
// Возвращает -1 если a < b, 0 если равны, 1 если a > b.
func NaturalCompare(a, b string) int {
	ia, ib := 0, 0
	for ia < len(a) && ib < len(b) {
		ca, cb := rune(a[ia]), rune(b[ib])

		// Если оба символа — цифры, сравниваем числовые группы
		if unicode.IsDigit(ca) && unicode.IsDigit(cb) {
			// Находим начало чисел
			ja, jb := ia, ib
			// Пропускаем ведущие нули и считаем длину
			for ja < len(a) && a[ja] == '0' {
				ja++
			}
			for jb < len(b) && b[jb] == '0' {
				jb++
			}
			// Находим конец чисел
			ka, kb := ja, jb
			for ka < len(a) && unicode.IsDigit(rune(a[ka])) {
				ka++
			}
			for kb < len(b) && unicode.IsDigit(rune(b[kb])) {
				kb++
			}

			// Сравниваем по длине (без ведущих нулей)
			lenA, lenB := ka-ja, kb-jb
			if lenA != lenB {
				if lenA < lenB {
					return -1
				}
				return 1
			}
			// Длины равны — сравниваем посимвольно
			for i := 0; i < lenA; i++ {
				if a[ja+i] != b[jb+i] {
					if a[ja+i] < b[jb+i] {
						return -1
					}
					return 1
				}
			}
			// Числа равны, но ведущие нули могут отличаться
			// Сравниваем количество ведущих нулей (больше нулей = меньше число)
			zerosA, zerosB := ja-ia, jb-ib
			if zerosA != zerosB {
				if zerosA > zerosB {
					return -1
				}
				return 1
			}
			ia, ib = ka, kb
			continue
		}

		// Обычное сравнение символов
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}
		ia++
		ib++
	}

	// Одна строка закончилась
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}
