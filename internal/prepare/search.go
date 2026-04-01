package prepare

import (
	"slices"
	"strings"

	"github.com/eterverda/fpvladder/internal/db"
)

// FindResult представляет результат поиска пилота
type FindResult struct {
	Name   string
	Id     string
	Rating int
}

// FindPilotsByName ищет пилотов по имени в базе
func (m *EventModel) FindPilotsByName(name string) []FindResult {
	var results []FindResult
	if name == "" {
		return results
	}

	searchWords := normalizeWords(name)
	pilots, _ := db.ListIds("./data", "pilot")

	for _, id := range pilots {
		pilot, err := db.ReadPilot("./data", id)
		if err != nil {
			continue
		}
		pilotWords := normalizeWords(pilot.Name)

		// Поиск: все слова запроса должны быть в имени пилота или наоборот
		if isSubset(searchWords, pilotWords) || isSubset(pilotWords, searchWords) {
			rating := 1200
			career := pilot.CareerForClass(m.Event.Class)
			if career != nil && len(career.Ratings) > 0 {
				rating = career.Ratings[len(career.Ratings)-1].Value
			}

			results = append(results, FindResult{
				Name:   pilot.Name,
				Id:     string(pilot.Id),
				Rating: rating,
			})
		}
	}
	return results
}

// normalizeWords разбивает строку на слова и нормализует каждое
func normalizeWords(s string) []string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		words[i] = normalizeWord(w)
	}
	return words
}

// normalizeWord нормализует слово: нижний регистр и ё → е
func normalizeWord(s string) string {
	s = strings.ToLower(s)
	return strings.ReplaceAll(s, "ё", "е")
}

// isSubset проверяет, является ли a подмножеством b (сравнение целых слов, порядок не важен)
// Учитывает ё/е как одинаковые символы
func isSubset(a, b []string) bool {
	for _, aw := range a {
		if !slices.Contains(b, aw) {
			return false
		}
	}
	return true
}
