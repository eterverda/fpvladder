package model

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPosition_YamlSerialization(t *testing.T) {
	tests := []struct {
		name     string
		position Position
		wantYaml string // ожидаемая строка в YAML (без кавычек)
	}{
		{
			name:     "простая позиция без ничьей",
			position: Position{Int: 1, TieCount: 0},
			wantYaml: "1",
		},
		{
			name:     "позиция 5 без ничьей",
			position: Position{Int: 5, TieCount: 0},
			wantYaml: "5",
		},
		{
			name:     "ничья из двух",
			position: Position{Int: 2, TieCount: 1},
			wantYaml: "2-3",
		},
		{
			name:     "ничья из трех",
			position: Position{Int: 2, TieCount: 2},
			wantYaml: "2-4",
		},
		{
			name:     "большая позиция с ничьей",
			position: Position{Int: 11, TieCount: 3},
			wantYaml: "11-14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сериализация
			node, err := tt.position.MarshalYAML()
			if err != nil {
				t.Fatalf("MarshalYAML failed: %v", err)
			}

			// Проверяем что сериализовалось как yaml.Node с правильным значением
			yamlNode, ok := node.(*yaml.Node)
			if !ok {
				t.Fatalf("expected *yaml.Node, got %T", node)
			}

			if yamlNode.Value != tt.wantYaml {
				t.Errorf("yaml value = %q, want %q", yamlNode.Value, tt.wantYaml)
			}

			// Проверяем что нет кавычек (plain style)
			if yamlNode.Style != 0 {
				t.Errorf("yaml style = %d, want 0 (plain style without quotes)", yamlNode.Style)
			}

			// Полная сериализация в YAML и проверка отсутствия кавычек
			data, err := yaml.Marshal(map[string]Position{"position": tt.position})
			if err != nil {
				t.Fatalf("yaml.Marshal failed: %v", err)
			}

			yamlStr := string(data)
			// Проверяем что в выводе нет кавычек вокруг значения
			if strings.Contains(yamlStr, `"`) {
				t.Errorf("yaml output contains quotes: %s", yamlStr)
			}

			// Проверяем что значение присутствует
			expectedLine := "position: " + tt.wantYaml
			if !strings.Contains(yamlStr, expectedLine) {
				t.Errorf("yaml output = %q, want to contain %q", yamlStr, expectedLine)
			}

			// Десериализация обратно
			var result struct {
				Position Position `yaml:"position"`
			}
			if err := yaml.Unmarshal(data, &result); err != nil {
				t.Fatalf("yaml.Unmarshal failed: %v", err)
			}

			if !result.Position.Equal(tt.position) {
				t.Errorf("round-trip failed: got %+v, want %+v", result.Position, tt.position)
			}
		})
	}
}

func TestPosition_YamlDeserialization(t *testing.T) {
	tests := []struct {
		name      string
		yamlInput string
		want      Position
		wantErr   bool
	}{
		{
			name:      "число без кавычек",
			yamlInput: "position: 5",
			want:      Position{Int: 5, TieCount: 0},
		},
		{
			name:      "строка с ничьей без кавычек",
			yamlInput: "position: 2-3",
			want:      Position{Int: 2, TieCount: 1},
		},
		{
			name:      "число в кавычках",
			yamlInput: `position: "5"`,
			want:      Position{Int: 5, TieCount: 0},
		},
		{
			name:      "строка с ничьей в кавычках",
			yamlInput: `position: "2-4"`,
			want:      Position{Int: 2, TieCount: 2},
		},
		{
			name:      "одинарные кавычки",
			yamlInput: `position: '11-12'`,
			want:      Position{Int: 11, TieCount: 1},
		},
		{
			name:      "невалидная строка",
			yamlInput: `position: "invalid"`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Position Position `yaml:"position"`
			}
			err := yaml.Unmarshal([]byte(tt.yamlInput), &result)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("yaml.Unmarshal failed: %v", err)
			}

			if !result.Position.Equal(tt.want) {
				t.Errorf("got %+v, want %+v", result.Position, tt.want)
			}
		})
	}
}

func TestRelativePosition_YamlSerialization(t *testing.T) {
	tests := []struct {
		name     string
		position RelativePosition
		wantYaml string
	}{
		{
			name:     "простая позиция с count",
			position: RelativePosition{Position: Position{Int: 1, TieCount: 0}, Count: 10},
			wantYaml: "1/10",
		},
		{
			name:     "ничья с count",
			position: RelativePosition{Position: Position{Int: 2, TieCount: 1}, Count: 10},
			wantYaml: "2-3/10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сериализация
			node, err := tt.position.MarshalYAML()
			if err != nil {
				t.Fatalf("MarshalYAML failed: %v", err)
			}

			yamlNode, ok := node.(*yaml.Node)
			if !ok {
				t.Fatalf("expected *yaml.Node, got %T", node)
			}

			if yamlNode.Value != tt.wantYaml {
				t.Errorf("yaml value = %q, want %q", yamlNode.Value, tt.wantYaml)
			}

			if yamlNode.Style != 0 {
				t.Errorf("yaml style = %d, want 0 (plain style)", yamlNode.Style)
			}

			// Полная сериализация
			data, err := yaml.Marshal(map[string]RelativePosition{"position": tt.position})
			if err != nil {
				t.Fatalf("yaml.Marshal failed: %v", err)
			}

			yamlStr := string(data)
			if strings.Contains(yamlStr, `"`) {
				t.Errorf("yaml output contains quotes: %s", yamlStr)
			}

			// Десериализация
			var result struct {
				Position RelativePosition `yaml:"position"`
			}
			if err := yaml.Unmarshal(data, &result); err != nil {
				t.Fatalf("yaml.Unmarshal failed: %v", err)
			}

			if result.Position.Position != tt.position.Position || result.Position.Count != tt.position.Count {
				t.Errorf("round-trip failed: got %+v, want %+v", result.Position, tt.position)
			}
		})
	}
}
