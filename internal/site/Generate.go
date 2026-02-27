package site

import (
	"cmp"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

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
	Events      []*eventRecord
}

type pilotRecord struct {
	Href     string
	Position int
	Name     string
	Rating   int
}

type eventRecord struct {
	id               model.Id
	NumPilots        int
	Name             string
	Date             string
	RatingAssignment string
}

type pilotPage struct {
	Name        string
	Rating      int
	Assignments []*assignmentRecord
}

type assignmentRecord struct {
	num        int
	Position   string
	Name       string
	Date       string
	Assignment string
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
	for _, pilot := range pilots {
		if err = generatePilot(outDir, pilot); err != nil {
			return err
		}
	}
	for _, event := range events {
		if err = generateEvent(event); err != nil {
			return err
		}
	}
	err = copyFile("internal/site/manifest.html", "site/manifest.html")
	if err != nil {
		return err
	}
	return nil
}

func generateIndex(outDir string, events []*model.Event, pilots []*model.Pilot) error {
	var pilotRecords = make([]*pilotRecord, 0, len(events))
	for _, pilot := range pilots {
		rating := pilot.RatingForClass(Class75mm)
		if rating == nil {
			continue
		}
		pilotRecords = append(pilotRecords, &pilotRecord{
			Href:   db.ResolveIdPathExt("", "pilot", pilot.Id, "html"),
			Name:   pilot.Name,
			Rating: rating.Value,
		})
	}
	slices.SortFunc(pilotRecords, func(a, b *pilotRecord) int {
		ord := -cmp.Compare(a.Rating, b.Rating)
		if ord == 0 {
			ord = cmp.Compare(a.Name, b.Name)
		}
		return ord
	})
	for i, pilotRecord := range pilotRecords {
		pilotRecord.Position = i + 1
		if i > 0 && pilotRecord.Rating == pilotRecords[i-1].Rating {
			pilotRecord.Position = pilotRecords[i-1].Position
		}
	}
	var eventRecords = make([]*eventRecord, 0, len(events))
	for _, event := range events {
		if event.Class != Class75mm {
			continue
		}
		eventRecords = append(eventRecords, &eventRecord{
			id:        event.Id,
			NumPilots: len(event.Pilots),
			Name:      strings.ReplaceAll(event.Name, ">", "⟫"),
			Date:      event.Date.String(),
		})
	}
	slices.SortFunc(eventRecords, func(a, b *eventRecord) int {
		ord := -cmp.Compare(a.Date, b.Date)
		if ord == 0 {
			ord = -cmp.Compare(a.id, b.id)
		}
		return ord
	})
	index := indexPage{
		Title:       "FPV Ladder ⟫ Drone Racing ⟫ 75mm",
		GeneratedAt: model.Today(),
		Pilots:      pilotRecords,
		Events:      eventRecords,
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

func generatePilot(outDir string, pilot *model.Pilot) error {
	career := pilot.CareerForClass(Class75mm)
	if career == nil {
		return nil
	}
	var page = &pilotPage{
		Name:   fmt.Sprintf("FPV Ladder ⟫ %s", pilot.Name),
		Rating: career.Ratings[len(career.Ratings)-1].Value,
	}
	for _, rating := range career.Ratings {
		page.Assignments = append(page.Assignments, &assignmentRecord{
			num:        rating.Num,
			Position:   rating.Position.String(),
			Name:       strings.ReplaceAll(rating.Event.Name, ">", "⟫"),
			Date:       rating.Event.Date.String(),
			Assignment: strings.ReplaceAll(fmt.Sprintf("%+d → %d", rating.Delta, rating.Value), "-", "−"),
		})
	}
	slices.SortFunc(page.Assignments, func(a, b *assignmentRecord) int {
		return -cmp.Compare(a.num, b.num)
	})

	path := db.ResolveIdPathExt(outDir, "pilot", pilot.Id, "html")
	tmpl, err := template.New("pilot.tmpl").ParseFiles("internal/site/pilot.tmpl")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Применяем данные к шаблону и пишем в файл
	err = tmpl.ExecuteTemplate(file, "pilot.tmpl", page)
	return err
}

func generateEvent(event *model.Event) error {
	return nil
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
