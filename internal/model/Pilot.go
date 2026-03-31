package model

// RatingSummary — актуальный срез данных по конкретному классу в карточке пилота
type RatingSummary struct {
	Num      int              `yaml:"num"`      // Номер присвоения
	Event    Event            `yaml:"event"`    // Информацмя о событии (сокращенная)
	Position RelativePosition `yaml:"position"` // Относительная позиция
	Delta    int              `yaml:"delta"`    // Изменение рейтинга
	Value    int              `yaml:"value"`    // Текущее значение рейтинга
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
