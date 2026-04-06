package db

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/eterverda/fpvladder/internal/model"
)

// indexEntry простая запись для индекса
type indexEntry struct {
	Class       model.Class `yaml:"class"`
	PilotId     model.Id    `yaml:"pilot_id"`
	PilotName   string      `yaml:"pilot_name"`
	EventDate   model.Date  `yaml:"event_date"`
	RatingNum   int         `yaml:"rating_num"`
	RatingValue int         `yaml:"rating_value"`
}

// GenerateIndex создает index.yaml с простыми записями для каждого пилота-класса
// pilots - список пилотов (должен быть полностью загружен с карьерами)
// outPath - путь для сохранения index.yaml
func GenerateIndex(pilots []*model.Pilot, outPath string) error {
	// Создаем директории при необходимости
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("ошибка создания директории для индекса: %w", err)
	}

	// Собираем все записи
	var entries []indexEntry
	for _, pilot := range pilots {
		for _, career := range pilot.Careers {
			if len(career.Ratings) == 0 {
				continue
			}
			// Берем только последний рейтинг
			lastRating := career.Ratings[len(career.Ratings)-1]

			entries = append(entries, indexEntry{
				Class:       career.Class,
				PilotId:     pilot.Id,
				PilotName:   pilot.Name,
				EventDate:   lastRating.Event.Date,
				RatingNum:   lastRating.Num,
				RatingValue: lastRating.Value,
			})
		}
	}

	// Сортируем: сначала по имени, потом по id, потом по class
	slices.SortFunc(entries, func(a, b indexEntry) int {
		if cmp := cmp.Compare(a.PilotName, b.PilotName); cmp != 0 {
			return cmp
		}
		if cmp := a.PilotId.Compare(b.PilotId); cmp != 0 {
			return cmp
		}
		return a.Class.Compare(b.Class)
	})

	// Пишем файл: каждая запись — отдельный YAML документ
	file, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("ошибка создания файла индекса: %w", err)
	}
	defer file.Close()

	for _, entry := range entries {
		data, err := model.MarshalPrettyYaml(entry)
		if err != nil {
			return fmt.Errorf("ошибка маршалинга записи для %s: %w", entry.PilotId, err)
		}
		// Добавляем document separator перед каждым документом
		if _, err := file.WriteString("---\n"); err != nil {
			return fmt.Errorf("ошибка записи разделителя: %w", err)
		}
		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("ошибка записи записи для %s: %w", entry.PilotId, err)
		}
	}

	return nil
}
