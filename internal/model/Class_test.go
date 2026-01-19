package model

import "testing"

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
