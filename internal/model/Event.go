package model

type Organizer struct {
	Name string `yaml:"name"`
	Link Link   `yaml:"link,omitempty"`
}

type PilotEntry struct {
	Position int    `yaml:"position"`
	Id       Id     `yaml:"id"`
	Name     string `yaml:"name,omitempty"`
}

type Event struct {
	Id          Id           `yaml:"id"`
	Date        Date         `yaml:"date"`
	Name        string       `yaml:"name"`
	Description string       `yaml:"description,omitempty"`
	Link        Link         `yaml:"link,omitempty"`
	Organizer   Organizer    `yaml:"organizer"`
	Class       Class        `yaml:"class"` // drone-racing :: 75mm :: individual
	Pilots      []PilotEntry `yaml:"pilots"`
}
