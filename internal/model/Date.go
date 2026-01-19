package model

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

type Date time.Time

const dateFormat = "2006-01-02"

func Today() Date {
	return Date(time.Now())
}

// --- Универсальный маршалинг ---

func (d Date) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Date) UnmarshalText(text []byte) error {
	t, err := time.Parse(dateFormat, string(text))
	if err != nil {
		return fmt.Errorf("неверный формат даты: %v", err)
	}
	*d = Date(t)
	return nil
}

// --- YAML специфика для красоты ---

func (d Date) MarshalYAML() (interface{}, error) {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: d.String(),
		Style: 0, // Явно указываем "без кавычек"
	}, nil
}

func (d Date) String() string {
	return time.Time(d).Format(dateFormat)
}
