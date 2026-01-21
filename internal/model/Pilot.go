package model

// RatingSummary — актуальный срез данных по конкретному классу в карточке пилота
type RatingSummary struct {
	Class    Class `yaml:"class"`
	Value    int   `yaml:"value"`     // Текущее значение рейтинга
	OriginId Id    `yaml:"origin_id"` // Ссылка на последний расчет
	Date     Date  `yaml:"date"`      // Дата последнего изменения
	Qty      int   `yaml:"qty"`       // Количество присвоений
}

// Pilot — учетная карточка пилота
type Pilot struct {
	Id      Id              `yaml:"id"`
	Name    string          `yaml:"name"`
	Ratings []RatingSummary `yaml:"ratings"`
}
