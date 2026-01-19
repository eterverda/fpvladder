package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDate_YamlCycle(t *testing.T) {
	// 1. Подготовим структуру с нашей датой
	type Host struct {
		D Date `yaml:"date"`
	}

	inputYaml := "date: 2025-12-28\n"
	var host Host

	// 2. Тестируем Unmarshal (через TextUnmarshaler)
	err := yaml.Unmarshal([]byte(inputYaml), &host)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Проверяем, что дата распарсилась корректно
	expected := "2025-12-28"
	if host.D.String() != expected {
		t.Errorf("Expected date %s, got %s", expected, host.D.String())
	}

	// 3. Тестируем Marshal (через TextMarshaler)
	output, err := yaml.Marshal(&host)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Проверяем, что на выходе получили ту же строку
	if string(output) != inputYaml {
		t.Errorf("Expected yaml output %q, got %q", inputYaml, string(output))
	}
}

func TestDate_InvalidFormat(t *testing.T) {
	type Host struct {
		D Date `yaml:"date"`
	}

	invalidYaml := "date: 28-12-2025\n" // Неверный формат
	var host Host

	err := yaml.Unmarshal([]byte(invalidYaml), &host)
	if err == nil {
		t.Error("Expected error for invalid date format, but got nil")
	}
}
