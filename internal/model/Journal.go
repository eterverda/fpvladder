package model

type Algorithm string

// Journal — верхнеуровневый документ (файл в data/rating/...)
type Journal struct {
	EventId     Id            `yaml:"event_id"`
	Description string        `yaml:"description"`
	Date        Date          `yaml:"date"`
	Pilots      []PilotRecord `yaml:"pilots"`
}

// PilotRecord — запись о конкретном пилоте в этом журнале
type PilotRecord struct {
	Id      Id                 `yaml:"id"`
	Name    string             `yaml:"name"`
	Ratings []RatingAssignment `yaml:"ratings"` // Твое поле rating
}

// RatingAssignment — начисление баллов по конкретному классу
type RatingAssignment struct {
	Class     Class     `yaml:"class"`
	OriginId  Id        `yaml:"origin_id,omitempty"` // Ссылка на предыдущий Journal
	OldValue  int       `yaml:"old_value"`
	Algorithm Algorithm `yaml:"algorithm"`
	Delta     int       `yaml:"delta"`
	NewValue  int       `yaml:"new_value"`
}
