package model

import (
	"fmt"
	"time"
)

type Id string

// FormatId собирает ID из объекта времени и порядкового номера.
// Используется для обеспечения единого формата YYYY/MM-DD/N по всей системе.
func FormatId(date Date, seq int) Id {
	return Id(fmt.Sprintf("%s/%s/%d",
		time.Time(date).Format("2006"),
		time.Time(date).Format("01-02"),
		seq,
	))
}

// MarshalText позволяет Id вести себя как строка в YAML/JSON
func (i Id) MarshalText() ([]byte, error) {
	return []byte(i), nil
}

// UnmarshalText позволяет загружать Id из строкового поля
func (i *Id) UnmarshalText(text []byte) error {
	// Тут в будущем можно добавить проверку формата регуляркой
	*i = Id(string(text))
	return nil
}

// String для удобного вывода
func (i Id) String() string {
	return string(i)
}
