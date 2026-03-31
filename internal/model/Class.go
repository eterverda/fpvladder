package model

import "strings"

const ClassSeparator = ">"

type Class string

const (
	Class75mm  Class = "drone-racing > 75mm"
	Class125mm Class = "drone-racing > 125mm"
	Class200mm Class = "drone-racing > 200mm"
	Class330mm Class = "drone-racing > 330mm"
)

var KnownClasses = []Class{Class75mm, Class125mm, Class200mm, Class330mm}

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
