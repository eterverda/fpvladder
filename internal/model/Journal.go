package model

type Algorithm struct {
	name string `yaml:"name"`
}

// Journal — верхнеуровневый документ (файл в data/rating/...)
type Journal struct {
	EventId     Id            `yaml:"event_id"`
	Description string        `yaml:"description"`
	Date        string        `yaml:"date"`
	Pilots      []PilotRecord `yaml:"pilots"` // Твое поле pilots
}

// PilotRecord — запись о конкретном пилоте в этом журнале
type PilotRecord struct {
	PilotId Id                 `yaml:"id"`
	Name    string             `yaml:"name"`
	Ratings []RatingAssignment `yaml:"ratings"` // Твое поле rating
}

// RatingAssignment — начисление баллов по конкретному классу
type RatingAssignment struct {
	Class     string    `yaml:"class"`
	StageName string    `yaml:"stage_name"`
	OriginId  Id        `yaml:"origin_id"` // Ссылка на предыдущий Journal
	OldValue  int       `yaml:"old_value"`
	Algorithm Algorithm `yaml:"algorithm"`
	Delta     int       `yaml:"delta"`
	NewValue  int       `yaml:"new_value"`
}
