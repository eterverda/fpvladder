package model

type Organizer struct {
	Name string `yaml:"name"`
	Link Link   `yaml:"link,omitempty"`
}

type Algorithm string

// RatingAssignment — начисление баллов по конкретному классу
type RatingAssignment struct {
	Class     Class     `yaml:"class"`
	OriginId  Id        `yaml:"origin_id,omitempty"` // Ссылка на предыдущий Journal
	OldValue  int       `yaml:"old_value,omitempty"`
	Algorithm Algorithm `yaml:"algorithm"`
	Delta     int       `yaml:"delta"`
	NewValue  int       `yaml:"new_value"`
}

type PilotEntry struct {
	Position Position           `yaml:"position"`
	Team     int                `yaml:"team,omitempty"` // Номер команды (0 = нет команды)
	Id       Id                 `yaml:"id,omitempty"`
	Name     string             `yaml:"name,omitempty"`
	Ratings  []RatingAssignment `yaml:"ratings,omitempty"`
}

func (p *PilotEntry) RatingForClass(class Class) *RatingAssignment {
	for _, rating := range p.Ratings {
		if rating.Class == class {
			return &rating
		}
	}
	return nil
}

type Event struct {
	Id          Id           `yaml:"id,omitempty"`
	Date        Date         `yaml:"date"`
	Name        string       `yaml:"name"`
	Description string       `yaml:"description,omitempty"`
	Link        Link         `yaml:"link,omitempty"`
	Organizer   Organizer    `yaml:"organizer,omitempty"`
	Class       Class        `yaml:"class,omitempty"` // drone-racing > 75mm
	Pilots      []PilotEntry `yaml:"pilots,omitempty"`
}

type FutureEvent struct {
	Id          Id        `yaml:"id"`
	Date        Date      `yaml:"date"`
	Name        string    `yaml:"name"`
	Description string    `yaml:"description,omitempty"`
	Link        Link      `yaml:"link,omitempty"`
	Organizer   Organizer `yaml:"organizer,omitempty"`
	Classes     []Class   `yaml:"class"`
}
