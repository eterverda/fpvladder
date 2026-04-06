package model

import (
	"slices"
	"testing"
)

func TestClass_Parent(t *testing.T) {
	tests := []struct {
		name  string
		input Class
		want  Class
	}{
		{
			name:  "Три уровня иерархии",
			input: "drone-racing > 75mm > individual",
			want:  "drone-racing > 75mm",
		},
		{
			name:  "Два уровня",
			input: "drone-racing > 75mm",
			want:  "drone-racing",
		},
		{
			name:  "Корневой класс",
			input: "drone-racing",
			want:  "",
		},
		{
			name:  "Пустая строка",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.Parent(); got != tt.want {
				t.Errorf("Class.Parent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClass_Compare_Sort(t *testing.T) {
	// Проверяем сортировку классов используя KnownClasses
	// Создаём перевернутый слайс
	input := make([]Class, len(KnownClasses))
	for i := 0; i < len(KnownClasses); i++ {
		input[i] = KnownClasses[len(KnownClasses)-1-i]
	}

	// Сортируем используя Compare
	slices.SortFunc(input, Class.Compare)

	// Проверяем, что отсортировалось как KnownClasses
	for i, v := range input {
		if v != KnownClasses[i] {
			t.Errorf("Sort position %d: got %q, want %q", i, v, KnownClasses[i])
		}
	}
}
