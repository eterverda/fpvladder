package model

import (
	"gopkg.in/yaml.v3"
)

// Link может содержать одну или несколько строк (URL)
type Link []string

// UnmarshalYAML — магия превращения строки ИЛИ массива в наш слайс
func (l *Link) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode: // Если в YAML просто строка: link: "http://..."
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		*l = Link{s}

	case yaml.SequenceNode: // Если в YAML список: link: ["http...", "http..."]
		var s []string
		if err := value.Decode(&s); err != nil {
			return err
		}
		*l = Link(s)
	}
	return nil
}

// MarshalYAML — магия превращения слайса обратно в строку (если одна) или массив
func (l Link) MarshalYAML() (interface{}, error) {
	if len(l) == 1 {
		return l[0], nil // Кодируем как простую строку
	}
	return []string(l), nil // Кодируем как полноценный список
}
