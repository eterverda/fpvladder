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

// ClassDisplayNames маппинг классов на отображаемые имена
var ClassDisplayNames = map[model.Class]string{
	model.Class75mm:  "75мм",
	model.Class125mm: "125мм",
	model.Class200mm: "200мм",
	model.Class330mm: "330мм",
}

// ClassParamValues маппинг классов на значения URL параметра
var ClassParamValues = map[model.Class]string{
	model.Class75mm:  "75mm",
	model.Class125mm: "125mm",
	model.Class200mm: "200mm",
	model.Class330mm: "330mm",
}

type indexPage struct {
	Title       string
	GeneratedAt model.Date
	Classes     []*indexClassData
}

type indexClassData struct {
	Class        model.Class
	ClassName    string
	ParamValue   string
	Pilots       []*pilotRecord
	Events       []*eventRecord
	FutureEvents []*eventRecord
}

type pilotRecord struct {
	Id       model.Id
	Href     string
	Position int
	Name     string
	Rating   int
}

type eventRecord struct {
	Id               model.Id
	Href             string
	NumPilots        int
	Name             string
	Date             string
	RatingAssignment string
}

type pilotPage struct {
	Id      model.Id
	Name    string
	Classes []*pilotClassData
}

type pilotClassData struct {
	Class       model.Class
	ClassName   string
	ParamValue  string
	Rating      int
	Assignments []*assignmentRecord
}

type assignmentRecord struct {
	num        int
	Id         model.Id
	Href       string
	Position   string
	Name       string
	Date       string
	Assignment string
}

type eventPage struct {
	Id          model.Id
	Name        string
	Date        string
	Description template.HTML
	Rating      int
	Results     []*resultRecord
}

type resultRecord struct {
	Id         model.Id
	Href       string
	Position   string
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
	// Build data for all classes
	var classes []*indexClassData
	for _, class := range model.KnownClasses {
		classData := &indexClassData{
			Class:      class,
			ClassName:  ClassDisplayNames[class],
			ParamValue: ClassParamValues[class],
		}

		// Pilots for this class
		var pilotRecords = make([]*pilotRecord, 0)
		for _, pilot := range pilots {
			career := pilot.CareerForClass(class)
			if career == nil {
				continue
			}
			pilotRecords = append(pilotRecords, &pilotRecord{
				Id:     pilot.Id,
				Href:   db.ResolveIdPathExt("", "pilot", pilot.Id, "html"),
				Name:   pilot.Name,
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
		classData.Pilots = pilotRecords

		// Events for this class
		var eventRecords = make([]*eventRecord, 0)
		for _, event := range events {
			if event.Class != class {
				continue
			}
			eventRecords = append(eventRecords, &eventRecord{
				Id:        event.Id,
				Href:      db.ResolveIdPathExt("", "event", event.Id, "html"),
				NumPilots: len(event.Pilots),
				Name:      strings.ReplaceAll(event.Name, ">", "⟫"),
				Date:      event.Date.String(),
			})
		}
		slices.SortFunc(eventRecords, func(a, b *eventRecord) int {
			ord := -cmp.Compare(a.Date, b.Date)
			if ord == 0 {
				ord = -cmp.Compare(a.Id, b.Id)
			}
			return ord
		})
		classData.Events = eventRecords

		// Future events for this class (event may have multiple classes)
		var futureEventRecords = make([]*eventRecord, 0)
		for _, event := range futureEvents {
			// Check if this future event includes current class
			hasClass := false
			for _, c := range event.Classes {
				if c == class {
					hasClass = true
					break
				}
			}
			if !hasClass {
				continue
			}
			futureEventRecords = append(futureEventRecords, &eventRecord{
				Id:   event.Id,
				Href: db.ResolveIdPathExt("", "future_event", event.Id, "html"),
				Name: strings.ReplaceAll(event.Name, ">", "⟫"),
				Date: event.Date.String(),
			})
		}
		classData.FutureEvents = futureEventRecords

		// Only add class if it has any data
		if len(pilotRecords) > 0 || len(eventRecords) > 0 || len(futureEventRecords) > 0 {
			classes = append(classes, classData)
		}
	}

	index := indexPage{
		Title:       "FPV Ladder",
		GeneratedAt: model.Today(),
		Classes:     classes,
	}

	tmpl, err := template.New("index.tmpl").Funcs(template.FuncMap{
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}).ParseFiles("internal/site/index.tmpl", "internal/site/header.tmpl", "internal/site/widget.tmpl")
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

	err = tmpl.Execute(file, index)
	return err
}

func generatePilot(outDir string, pilot *model.Pilot) error {
	// Build data for all classes this pilot has careers in
	var classes []*pilotClassData
	for _, class := range model.KnownClasses {
		career := pilot.CareerForClass(class)
		if career == nil {
			continue
		}
		pc := &pilotClassData{
			Class:      class,
			ClassName:  ClassDisplayNames[class],
			ParamValue: ClassParamValues[class],
			Rating:     career.Ratings[len(career.Ratings)-1].Value,
		}
		for _, rating := range career.Ratings {
			name := strings.ReplaceAll(rating.Event.Name, ">", "⟫")
			pc.Assignments = append(pc.Assignments, &assignmentRecord{
				num:        rating.Num,
				Id:         rating.Event.Id,
				Href:       db.ResolveIdPathExt("../../../", "event", rating.Event.Id, "html"),
				Position:   rating.Position.String(),
				Name:       name,
				Date:       rating.Event.Date.String(),
				Assignment: strings.ReplaceAll(fmt.Sprintf("%+d → %d", rating.Delta, rating.Value), "-", "−"),
			})
		}
		slices.SortFunc(pc.Assignments, func(a, b *assignmentRecord) int {
			return -cmp.Compare(a.num, b.num)
		})
		classes = append(classes, pc)
	}

	if len(classes) == 0 {
		return nil
	}

	page := &pilotPage{
		Id:      pilot.Id,
		Name:    pilot.Name,
		Classes: classes,
	}

	path := db.ResolveIdPathExt(outDir, "pilot", pilot.Id, "html")
	tmpl, err := template.New("pilot.tmpl").Funcs(template.FuncMap{
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}).ParseFiles("internal/site/pilot.tmpl", "internal/site/header.tmpl", "internal/site/widget.tmpl")
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

	err = tmpl.ExecuteTemplate(file, "pilot.tmpl", page)
	return err
}

func generateEvent(outDir string, event *model.Event) error {
	var page = &eventPage{
		Id:   event.Id,
		Name: strings.ReplaceAll(event.Name, ">", "⟫"),
	}
	for _, pilot := range event.Pilots {
		rating := pilot.RatingForClass(event.Class)
		if rating == nil {
			continue
		}
		name := pilot.Name
		page.Results = append(page.Results, &resultRecord{
			Id:         pilot.Id,
			Href:       db.ResolveIdPathExt("../../../", "pilot", pilot.Id, "html"),
			Position:   pilot.Position.String(),
			Name:       name,
			Assignment: strings.ReplaceAll(fmt.Sprintf("%+d → %d", rating.Delta, rating.NewValue), "-", "−"),
		})
	}
	path := db.ResolveIdPathExt(outDir, "event", event.Id, "html")
	tmpl, err := template.New("event.tmpl").Funcs(template.FuncMap{
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}).ParseFiles("internal/site/event.tmpl", "internal/site/header.tmpl", "internal/site/widget.tmpl")
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
	tmpl, err := template.New("future_event.tmpl").ParseFiles("internal/site/future_event.tmpl", "internal/site/header.tmpl", "internal/site/widget.tmpl")
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
	tmpl, err := template.New("manifest.tmpl").ParseFiles("internal/site/manifest.tmpl", "internal/site/header.tmpl", "internal/site/widget.tmpl")
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
