package site

import (
	"cmp"
	"html/template"
	"os"
	"path/filepath"
	"slices"

	"github.com/eterverda/fpvladder/internal/db"
	"github.com/eterverda/fpvladder/internal/model"
)

const (
	Class75mm = model.Class("drone-racing > 75mm")
)

type indexPage struct {
	Title       string
	GeneratedAt model.Date
	Pilots      []*pilotRecord
	Events      []*model.Event
}

type pilotRecord struct {
	Position int
	Name     string
	Rating   int
}

func Generate(baseDir, outDir string) error {
	events, err := readAllEvents(baseDir)
	if err != nil {
		return err
	}
	pilots, err := readAllPilots(baseDir)
	if err != nil {
		return err
	}
	err = generateIndex(outDir, events, pilots)
	if err != nil {
		return err
	}
	return nil
}

func generateIndex(outDir string, events []*model.Event, pilots []*model.Pilot) error {
	var pilotRecords []*pilotRecord
	for _, pilot := range pilots {
		rating := pilot.RatingForClass(Class75mm)
		if rating == nil {
			continue
		}
		pilotRecords = append(pilotRecords,
			&pilotRecord{
				Name:   pilot.Name,
				Rating: rating.Value,
			},
		)
	}
	slices.SortFunc(
		pilotRecords,
		func(a, b *pilotRecord) int {
			ord := -cmp.Compare(a.Rating, b.Rating)
			if ord == 0 {
				ord = cmp.Compare(a.Name, b.Name)
			}
			return ord
		},
	)
	for i, pilotRecord := range pilotRecords {
		pilotRecord.Position = i + 1
		if i > 0 && pilotRecord.Rating == pilotRecords[i-1].Rating {
			pilotRecord.Position = pilotRecords[i-1].Position
		}
	}
	index := indexPage{
		Title:       string(Class75mm),
		GeneratedAt: model.Today(),
		Pilots:      pilotRecords,
		Events:      events,
	}

	tmpl, err := template.New("index.tmpl").ParseFiles("internal/site/index.tmpl")
	if err != nil {
		return err
	}

	path := filepath.Join(outDir, "index.html")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Применяем данные к шаблону и пишем в файл
	err = tmpl.Execute(file, index)
	return err
}

func readAllEvents(baseDir string) ([]*model.Event, error) {
	eventIds, err := db.ListIds(baseDir, "event")
	if err != nil {
		return nil, err
	}
	events := make([]*model.Event, 0, len(eventIds))
	for _, eventId := range eventIds {
		event, err := db.ReadEvent(baseDir, eventId)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func readAllPilots(baseDir string) ([]*model.Pilot, error) {
	pilotIds, err := db.ListIds(baseDir, "pilot")
	if err != nil {
		return nil, err
	}
	pilots := make([]*model.Pilot, 0, len(pilotIds))
	for _, pilotId := range pilotIds {
		pilot, err := db.ReadPilot(baseDir, pilotId)
		if err != nil {
			return nil, err
		}
		pilots = append(pilots, pilot)
	}
	return pilots, nil
}
