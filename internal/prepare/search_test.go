package prepare

import (
	"testing"
)

func TestNormalizeWord(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Иван", "иван"},
		{"ПЕТРОВ", "петров"},
		{"Ёжик", "ежик"},
		{"ёлка", "елка"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeWord(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeWord(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeWords(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Иван Петров", []string{"иван", "петров"}},
		{"  Ёжик  Ёлка  ", []string{"ежик", "елка"}},
		{"Артём Ёжиков", []string{"артем", "ежиков"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeWords(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("normalizeWords(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i, w := range result {
				if w != tt.expected[i] {
					t.Errorf("normalizeWords(%q)[%d] = %q, want %q", tt.input, i, w, tt.expected[i])
				}
			}
		})
	}
}

func TestIsSubset(t *testing.T) {
	// Тесты для isSubset с уже нормализованными словами (ё→е, lower case)
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "exact match",
			a:        []string{"иван", "петров"},
			b:        []string{"иван", "петров"},
			expected: true,
		},
		{
			name:     "subset match",
			a:        []string{"иван", "петров"},
			b:        []string{"иван", "сидорович", "петров"},
			expected: true,
		},
		{
			name:     "different order",
			a:        []string{"петров", "иван"},
			b:        []string{"иван", "сидорович", "петров"},
			expected: true,
		},
		{
			name:     "partial word should not match",
			a:        []string{"иван", "петрович"},
			b:        []string{"иван", "сидоров"},
			expected: false,
		},
		{
			name:     "empty subset",
			a:        []string{},
			b:        []string{"иван", "петров"},
			expected: true,
		},
		{
			name:     "single word match",
			a:        []string{"иван"},
			b:        []string{"иван", "петров"},
			expected: true,
		},
		{
			name:     "single word no match",
			a:        []string{"петр"},
			b:        []string{"иван", "петров"},
			expected: false,
		},
		{
			name:     "already normalized yo to e",
			a:        []string{"артем", "ежик"},
			b:        []string{"артем", "ежик"},
			expected: true,
		},
		{
			name:     "already normalized lower case",
			a:        []string{"иван", "петров"},
			b:        []string{"иван", "петров"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSubset(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("isSubset(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestFindPilotsByNameLogic(t *testing.T) {
	// Тесты для логики FindPilotsByName: isSubset(a,b) || isSubset(b,a)
	// Двусторонняя проверка: либо все слова поиска в имени, либо все слова имени в поиске
	tests := []struct {
		name        string
		searchWords []string
		pilotWords  []string
		shouldFind  bool
	}{
		{
			name:        "Иван Петров находит Иван Сидорович Петров",
			searchWords: []string{"иван", "петров"},
			pilotWords:  []string{"иван", "сидорович", "петров"},
			shouldFind:  true, // isSubset(search, pilot) = true
		},
		{
			name:        "Иван Петрович Сидоров находит Иван Сидоров",
			searchWords: []string{"иван", "петрович", "сидоров"},
			pilotWords:  []string{"иван", "сидоров"},
			shouldFind:  true, // isSubset(pilot, search) = true
		},
		{
			name:        "Иван Сидоров находит Иван Петрович Сидоров",
			searchWords: []string{"иван", "сидоров"},
			pilotWords:  []string{"иван", "петрович", "сидоров"},
			shouldFind:  true, // isSubset(search, pilot) = true
		},
		{
			name:        "полное совпадение",
			searchWords: []string{"иван", "петров"},
			pilotWords:  []string{"иван", "петров"},
			shouldFind:  true,
		},
		{
			name:        "пустой поиск",
			searchWords: []string{},
			pilotWords:  []string{"иван", "петров"},
			shouldFind:  true, // пустое множество является подмножеством любого
		},
		{
			name:        "пересечение но не подмножество",
			searchWords: []string{"иван", "петров"},
			pilotWords:  []string{"иван", "сидоров"},
			shouldFind:  false, // ни одно не является подмножеством другого
		},
		{
			name:        "с ё и е (нормализовано)",
			searchWords: []string{"артем", "ежик"},
			pilotWords:  []string{"артем", "ежик", "иванов"},
			shouldFind:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSubset(tt.searchWords, tt.pilotWords) || isSubset(tt.pilotWords, tt.searchWords)
			if result != tt.shouldFind {
				t.Errorf("findLogic(%v, %v) = %v, want %v",
					tt.searchWords, tt.pilotWords, result, tt.shouldFind)
			}
		})
	}
}
