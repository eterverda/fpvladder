package model

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Position struct {
	Numerator   int
	Denominator int
}

func (p Position) String() string {
	return fmt.Sprintf("%d/%d", p.Numerator, p.Denominator)
}

// MarshalYAML превращает структуру в строку "num/denom"
func (p Position) MarshalYAML() (interface{}, error) {
	return fmt.Sprintf("%d/%d", p.Numerator, p.Denominator), nil
}

// UnmarshalYAML парсит строку "num/denom" обратно в структуру
func (p *Position) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid position format: %s", s)
	}

	_, err := fmt.Sscanf(s, "%d/%d", &p.Numerator, &p.Denominator)
	return err
}

// RatingSummary — актуальный срез данных по конкретному классу в карточке пилота
type RatingSummary struct {
	Num      int      `yaml:"num"`      // Номер присвоения
	Event    Event    `yaml:"event"`    // Информацмя о событии (сокращенная)
	Position Position `yaml:"position"` // Позиция
	Delta    int      `yaml:"delta"`    // Изменение рейтинга
	Value    int      `yaml:"value"`    // Текущее значение рейтинга
}

type Career struct {
	Class   Class           `yaml:"class"`
	Ratings []RatingSummary `yaml:"ratings"`
}

// Pilot — учетная карточка пилота
type Pilot struct {
	Id      Id       `yaml:"id"`
	Name    string   `yaml:"name"`
	Careers []Career `yaml:"careers"`
}

func (p *Pilot) CareerForClass(class Class) *Career {
	for _, career := range p.Careers {
		if career.Class == class {
			return &career
		}
	}
	return nil
}

func (p *Pilot) RatingForClass(class Class) *RatingSummary {
	career := p.CareerForClass(class)
	if career == nil {
		return nil
	}
	return &career.Ratings[len(career.Ratings)-1]
}
