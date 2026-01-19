package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestId_YamlCycle(t *testing.T) {
	// Структура-обертка для теста
	type Host struct {
		EventID Id `yaml:"id"`
	}

	// Тестируем формат со слэшами и дефисами
	testID := "2025/12-28/1"
	inputYaml := "id: " + testID + "\n"

	var host Host

	// 1. Проверяем анмаршаллинг (чтение)
	err := yaml.Unmarshal([]byte(inputYaml), &host)
	if err != nil {
		t.Fatalf("Не удалось распарсить YAML: %v", err)
	}

	if string(host.EventID) != testID {
		t.Errorf("Ожидали ID %s, получили %s", testID, host.EventID)
	}

	// 2. Проверяем маршаллинг (запись)
	output, err := yaml.Marshal(&host)
	if err != nil {
		t.Fatalf("Не удалось сериализовать в YAML: %v", err)
	}

	// Проверяем, что ID записался без лишних кавычек (благодаря TextMarshaler)
	if string(output) != inputYaml {
		t.Errorf("Результат маршаллинга отличается.\nОжидали: %q\nПолучили: %q", inputYaml, string(output))
	}
}
