package util

import (
	"slices"
	"sort"
	"testing"
)

func TestNaturalCompare_Equal(t *testing.T) {
	tests := []struct {
		a, b string
	}{
		{"", ""},
		{"abc", "abc"},
		{"123", "123"},
		{"abc123", "abc123"},
		// {"00123", "0123"}, // разное количество ведущих нулей - разные значения
	}

	for _, tt := range tests {
		if got := NaturalCompare(tt.a, tt.b); got != 0 {
			t.Errorf("NaturalCompare(%q, %q) = %d, want 0", tt.a, tt.b, got)
		}
	}
}

func TestNaturalCompare_Less(t *testing.T) {
	tests := []struct {
		a, b string
	}{
		{"a", "b"},
		{"a", "aa"},
		{"1", "2"},
		{"1", "10"},
		{"2", "10"},
		{"abc1", "abc2"},
		{"abc1", "abc10"},
		{"abc2", "abc10"},
		{"file1.txt", "file10.txt"},
		{"file1.txt", "file2.txt"},
	}

	for _, tt := range tests {
		if got := NaturalCompare(tt.a, tt.b); got != -1 {
			t.Errorf("NaturalCompare(%q, %q) = %d, want -1", tt.a, tt.b, got)
		}
	}
}

func TestNaturalCompare_Greater(t *testing.T) {
	tests := []struct {
		a, b string
	}{
		{"b", "a"},
		{"aa", "a"},
		{"2", "1"},
		{"10", "1"},
		{"10", "2"},
		{"abc2", "abc1"},
		{"abc10", "abc1"},
		{"abc10", "abc2"},
		{"file10.txt", "file1.txt"},
		{"file2.txt", "file1.txt"},
	}

	for _, tt := range tests {
		if got := NaturalCompare(tt.a, tt.b); got != 1 {
			t.Errorf("NaturalCompare(%q, %q) = %d, want 1", tt.a, tt.b, got)
		}
	}
}

func TestNaturalCompare_LeadingZeros(t *testing.T) {
	// Больше ведущих нулей = меньше число
	tests := []struct {
		a, b     string
		expected int
	}{
		{"001", "01", -1},     // 001 < 01 (больше нулей = меньше)
		{"01", "001", 1},      // 01 > 001
		{"0001", "001", -1},   // 0001 < 001
		{"00123", "0123", -1}, // 00123 < 0123 (больше ведущих нулей = меньше)
	}

	for _, tt := range tests {
		if got := NaturalCompare(tt.a, tt.b); got != tt.expected {
			t.Errorf("NaturalCompare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestNaturalCompare_Mixed(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"abc123def", "abc123def", 0},
		{"abc123def", "abc124def", -1},
		{"abc123def", "abc122def", 1},
		{"abc123def456", "abc123def457", -1},
		{"abc123def456", "abc123def45", 1},
		{"a1b2c3", "a1b2c4", -1},
		{"a1b2c3", "a1b2c2", 1},
	}

	for _, tt := range tests {
		if got := NaturalCompare(tt.a, tt.b); got != tt.expected {
			t.Errorf("NaturalCompare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestNaturalCompare_RealWorld(t *testing.T) {
	// Тестирование реальных сценариев использования
	tests := []struct {
		a, b     string
		expected int
	}{
		// Id формат: YYYY/MM-DD/N
		{"2024/01-15/1", "2024/01-15/2", -1},
		{"2024/01-15/10", "2024/01-15/2", 1},
		{"2024/01-05/1", "2024/01-15/1", -1},
		{"2023/12-31/1", "2024/01-01/1", -1},
		// Class формат
		{"drone-racing > 75mm", "drone-racing > 125mm", -1},
		{"drone-racing > 125mm", "drone-racing > 75mm", 1},
	}

	for _, tt := range tests {
		if got := NaturalCompare(tt.a, tt.b); got != tt.expected {
			t.Errorf("NaturalCompare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestNaturalCompare_Sort(t *testing.T) {
	// Проверяем, что сортировка работает корректно
	input := []string{"file10.txt", "file1.txt", "file2.txt", "file20.txt", "file11.txt"}
	expected := []string{"file1.txt", "file2.txt", "file10.txt", "file11.txt", "file20.txt"}

	slices.SortFunc(input, func(a, b string) int {
		return NaturalCompare(a, b)
	})

	for i, v := range input {
		if v != expected[i] {
			t.Errorf("Sort position %d: got %q, want %q", i, v, expected[i])
		}
	}
}

func TestNaturalCompare_ConsistencyWithStringsCompare(t *testing.T) {
	// Проверяем консистентность со стандартной сортировкой для строк без чисел
	input := []string{"apple", "banana", "cherry", "date"}

	// Копируем и сортируем стандартным способом
	stdSorted := make([]string, len(input))
	copy(stdSorted, input)
	sort.Strings(stdSorted)

	// Копируем и сортируем natural compare
	natSorted := make([]string, len(input))
	copy(natSorted, input)
	slices.SortFunc(natSorted, func(a, b string) int {
		return NaturalCompare(a, b)
	})

	for i := range stdSorted {
		if stdSorted[i] != natSorted[i] {
			t.Errorf("Position %d: std=%q, natural=%q", i, stdSorted[i], natSorted[i])
		}
	}
}

func BenchmarkNaturalCompare(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NaturalCompare("file123.txt", "file124.txt")
	}
}
