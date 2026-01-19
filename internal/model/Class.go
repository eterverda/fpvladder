package model

import "strings"

const ClassSeparator = ">"

type Class string

// Parent возвращает родительский класс.
// Например: "a > b > c" -> "a > b"
// Если родителя нет, возвращает ""
// Parent возвращает родительский класс, игнорируя пробелы вокруг ">"
func (c Class) Parent() Class {
	s := string(c)
	lastIdx := strings.LastIndex(s, ClassSeparator)
	if lastIdx == -1 {
		return ""
	}

	// Берем всё до разделителя и убираем пробелы по краям
	parent := strings.TrimSpace(s[:lastIdx])
	return Class(parent)
}

// MarshalText позволяет Class вести себя как строка в YAML/JSON
func (c Class) MarshalText() ([]byte, error) {
	return []byte(c), nil
}

// UnmarshalText позволяет загружать Class из строкового поля
func (c *Class) UnmarshalText(text []byte) error {
	// Тут в будущем можно добавить проверку формата регуляркой
	*c = Class(string(text))
	return nil
}
