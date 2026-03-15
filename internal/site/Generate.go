package site

import (
	"bytes"
	"cmp"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/eterverda/fpvladder/internal/db"
	"github.com/eterverda/fpvladder/internal/model"
	"github.com/yuin/goldmark"
)

const (
	Class75mm = model.Class("drone-racing > 75mm")
)

type indexPage struct {
	Title        string
	GeneratedAt  model.Date
	Pilots       []*pilotRecord
	Events       []*eventRecord
	FutureEvents []*eventRecord
}

type pilotRecord struct {
	Href     string
	Position int
	Name     string
	Rating   int
}

type eventRecord struct {
	id               model.Id
	Href             string
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
	Href       string
	Position   string
	Name       string
	Date       string
	Assignment string
}

type eventPage struct {
	Name        string
	Date        string
	Description template.HTML
	Rating      int
	Results     []*resultRecord
}

type resultRecord struct {
	Href       string
	Position   int
	Name       string
	Assignment string
}

func Generate(baseDir, outDir string) error {
	events, err := readAllEvents(baseDir)
	if err != nil {
		return err
	}
	futureEvents, err := readAllFutureEvents(baseDir)
	if err != nil {
		return err
	}
	pilots, err := readAllPilots(baseDir)
	if err != nil {
		return err
	}
	err = generateIndex(outDir, events, futureEvents, pilots)
	if err != nil {
		return err
	}
	err = generateICS(outDir, events, futureEvents)
	if err != nil {
		return err
	}
	for _, pilot := range pilots {
		if err = generatePilot(outDir, pilot); err != nil {
			return err
		}
	}
	for _, event := range events {
		if err = generateEvent(outDir, event); err != nil {
			return err
		}
	}
	for _, event := range futureEvents {
		if err = generateFutureEvent(outDir, event); err != nil {
			return err
		}
	}
	err = generateManifest(outDir)
	if err != nil {
		return err
	}
	err = copyFile("internal/site/styles.css", "build/styles.css")
	if err != nil {
		return err
	}
	err = copyFile("internal/site/scripts.js", "build/scripts.js")
	if err != nil {
		return err
	}
	fmt.Printf("[✓] Сайт сгенерирован: file://%s/index.html\n", outDir)
	return nil
}

func generateIndex(outDir string, events []*model.Event, futureEvents []*model.FutureEvent, pilots []*model.Pilot) error {
	var pilotRecords = make([]*pilotRecord, 0, len(events))
	for _, pilot := range pilots {
		career := pilot.CareerForClass(Class75mm)
		if career == nil {
			continue
		}
		name := pilot.Name
		pilotRecords = append(pilotRecords, &pilotRecord{
			Href:   db.ResolveIdPathExt("", "pilot", pilot.Id, "html"),
			Name:   name,
			Rating: career.Ratings[len(career.Ratings)-1].Value,
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
			Href:      db.ResolveIdPathExt("", "event", event.Id, "html"),
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
	var futureEventRecords = make([]*eventRecord, 0, len(futureEvents))
	for _, event := range futureEvents {
		futureEventRecords = append(futureEventRecords, &eventRecord{
			id:   event.Id,
			Href: db.ResolveIdPathExt("", "future_event", event.Id, "html"),
			Name: strings.ReplaceAll(event.Name, ">", "⟫"),
			Date: event.Date.String(),
		})
	}
	index := indexPage{
		Title:        "Drone Racing ⟫ 75mm",
		GeneratedAt:  model.Today(),
		Pilots:       pilotRecords,
		Events:       eventRecords,
		FutureEvents: futureEventRecords,
	}

	tmpl, err := template.New("index.tmpl").ParseFiles("internal/site/index.tmpl", "internal/site/header.tmpl")
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
		Name:   pilot.Name,
		Rating: career.Ratings[len(career.Ratings)-1].Value,
	}
	for _, rating := range career.Ratings {
		name := strings.ReplaceAll(rating.Event.Name, ">", "⟫")
		page.Assignments = append(page.Assignments, &assignmentRecord{
			num:        rating.Num,
			Href:       db.ResolveIdPathExt("../../../", "event", rating.Event.Id, "html"),
			Position:   rating.Position.String(),
			Name:       name,
			Date:       rating.Event.Date.String(),
			Assignment: strings.ReplaceAll(fmt.Sprintf("%+d → %d", rating.Delta, rating.Value), "-", "−"),
		})
	}
	slices.SortFunc(page.Assignments, func(a, b *assignmentRecord) int {
		return -cmp.Compare(a.num, b.num)
	})

	path := db.ResolveIdPathExt(outDir, "pilot", pilot.Id, "html")
	tmpl, err := template.New("pilot.tmpl").ParseFiles("internal/site/pilot.tmpl", "internal/site/header.tmpl")
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

func generateEvent(outDir string, event *model.Event) error {
	var page = &eventPage{
		Name: strings.ReplaceAll(event.Name, ">", "⟫"),
	}
	for _, pilot := range event.Pilots {
		rating := pilot.RatingForClass(event.Class)
		if rating == nil {
			continue
		}
		name := pilot.Name
		page.Results = append(page.Results, &resultRecord{
			Href:       db.ResolveIdPathExt("../../../", "pilot", pilot.Id, "html"),
			Position:   pilot.Position,
			Name:       name,
			Assignment: strings.ReplaceAll(fmt.Sprintf("%+d → %d", rating.Delta, rating.NewValue), "-", "−"),
		})
	}
	path := db.ResolveIdPathExt(outDir, "event", event.Id, "html")
	tmpl, err := template.New("event.tmpl").ParseFiles("internal/site/event.tmpl", "internal/site/header.tmpl")
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
	err = tmpl.ExecuteTemplate(file, "event.tmpl", page)
	return err
}

func generateFutureEvent(outDir string, event *model.FutureEvent) error {
	var page = &eventPage{
		Name:        strings.ReplaceAll(event.Name, ">", "⟫"),
		Description: template.HTML(md2html(event.Description)),
	}
	path := db.ResolveIdPathExt(outDir, "future_event", event.Id, "html")
	tmpl, err := template.New("future_event.tmpl").ParseFiles("internal/site/future_event.tmpl", "internal/site/header.tmpl")
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
	err = tmpl.ExecuteTemplate(file, "future_event.tmpl", page)
	return err
}

func generateManifest(outDir string) error {
	tmpl, err := template.New("manifest.tmpl").ParseFiles("internal/site/manifest.tmpl", "internal/site/header.tmpl")
	if err != nil {
		return err
	}

	path := filepath.Join(outDir, "manifest.html")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.ExecuteTemplate(file, "manifest.tmpl", nil)
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

func readAllFutureEvents(baseDir string) ([]*model.FutureEvent, error) {
	eventIds, err := db.ListIds(baseDir, "future_event")
	if err != nil {
		return nil, err
	}
	events := make([]*model.FutureEvent, 0, len(eventIds))
	for _, eventId := range eventIds {
		event, err := db.ReadFutureEvent(baseDir, eventId)
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

func generateICS(outDir string, events []*model.Event, futureEvents []*model.FutureEvent) error {
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)

	for _, e := range events {
		event := cal.AddEvent(icsId(e.Id))
		event.SetAllDayStartAt(time.Time(e.Date))
		event.SetSummary(e.Name)
		event.SetDescription(e.Description)

		// 4. HTML описание через твою функцию md2html
		htmlContent := fmt.Sprintf("<!DOCTYPE HTML><html><body>%s</body></html>", md2html(e.Description))

		// Очищаем HTML от переносов строк, так как ics чувствителен к ним в атрибутах
		cleanHtml := strings.ReplaceAll(htmlContent, "\n", "")
		event.AddProperty("X-ALT-DESC;FMTTYPE=text/html", cleanHtml)
	}

	for _, e := range futureEvents {
		event := cal.AddEvent(icsId(e.Id))
		event.SetAllDayStartAt(time.Time(e.Date))
		event.SetSummary(e.Name)
		event.SetDescription(e.Description)

		// 4. HTML описание через твою функцию md2html
		htmlContent := fmt.Sprintf("<!DOCTYPE HTML><html><body>%s</body></html>", md2html(e.Description))

		// Очищаем HTML от переносов строк, так как ics чувствителен к ним в атрибутах
		cleanHtml := strings.ReplaceAll(htmlContent, "\n", "")
		event.AddProperty("X-ALT-DESC;FMTTYPE=text/html", cleanHtml)
	}

	path := filepath.Join(outDir, "calendar.ics")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return cal.SerializeTo(file)
}

func md2html(source string) string {
	var dest bytes.Buffer
	err := goldmark.Convert([]byte(source), &dest)
	if err != nil {
		panic(err)
	}
	return dest.String()
}

func icsId(id model.Id) string {
	return fmt.Sprintf("%s@fpvladder.ru", strings.ReplaceAll(id.String(), "/", "-"))
}
