package model

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLink_YamlLifecycle(t *testing.T) {
	// Вспомогательная структура, имитирующая поле в Event
	type Host struct {
		Link Link `yaml:"link"`
	}

	t.Run("Single link: from string and back to string", func(t *testing.T) {
		yamlInput := "link: https://youtube.com/live\n"

		var h Host
		err := yaml.Unmarshal([]byte(yamlInput), &h)
		if err != nil {
			t.Fatalf("Ошибка при чтении одиночной ссылки: %v", err)
		}

		// Проверяем, что в коде это слайс из 1 элемента
		if len(h.Link) != 1 || h.Link[0] != "https://youtube.com/live" {
			t.Errorf("Ожидали 1 ссылку, получили: %v", h.Link)
		}

		// Проверяем маршаллинг обратно в строку
		output, err := yaml.Marshal(h)
		if err != nil {
			t.Fatalf("Ошибка при записи одиночной ссылки: %v", err)
		}

		if string(output) != yamlInput {
			t.Errorf("Маршаллинг вернул не строку:\nОжидали: %q\nПолучили: %q", yamlInput, string(output))
		}
	})

	t.Run("Multiple links: from sequence and back to sequence", func(t *testing.T) {
		yamlInput := "link:\n    - https://photo.com\n    - https://video.com\n"

		var h Host
		err := yaml.Unmarshal([]byte(yamlInput), &h)
		if err != nil {
			t.Fatalf("Ошибка при чтении списка ссылок: %v", err)
		}

		// Проверяем, что в коде это слайс из 2 элементов
		if len(h.Link) != 2 || h.Link[1] != "https://video.com" {
			t.Errorf("Ожидали 2 ссылки, получили: %v", h.Link)
		}

		// Проверяем маршаллинг обратно в список
		output, err := yaml.Marshal(h)
		if err != nil {
			t.Fatalf("Ошибка при записи списка ссылок: %v", err)
		}

		// Проверяем наличие дефисов в выводе
		if !strings.Contains(string(output), "- https://photo.com") {
			t.Errorf("Маршаллинг не сохранил формат списка:\n%s", string(output))
		}
	})

	t.Run("Empty link", func(t *testing.T) {
		yamlInput := "link: \"\"\n"
		var h Host
		yaml.Unmarshal([]byte(yamlInput), &h)

		if len(h.Link) != 1 || h.Link[0] != "" {
			t.Errorf("Ожидали пустую строку в слайсе, получили: %v", h.Link)
		}
	})
}
