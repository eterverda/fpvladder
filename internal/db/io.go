package db

import (
	"fmt"
	"os"

	"github.com/eterverda/fpvladder/internal/model"
	"gopkg.in/yaml.v3"
)

func ReadPilot(baseDir string, id model.Id) (*model.Pilot, error) {
	pilotPath := ResolveIdPath(baseDir, "pilot", id)

	pData, err := os.ReadFile(pilotPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("пилот [%s] не найден в БД (путь: %s)", id, pilotPath)
		}
		return nil, fmt.Errorf("ошибка доступа к БД пилотов: %w", err)
	}

	var dbPilot model.Pilot
	if err := yaml.Unmarshal(pData, &dbPilot); err != nil {
		return nil, fmt.Errorf("ошибка структуры файла пилота %s: %w", id, err)
	}
	return &dbPilot, nil
}

func ReadEvent(baseDir string, id model.Id) (*model.Event, error) {
	eventPath := ResolveIdPath(baseDir, "event", id)
	dbEvent, err := ReadEventPath(eventPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("событие [%s] не найдено в БД (путь: %s)", id, eventPath)
	}
	return dbEvent, err
}

func ReadEventPath(eventPath string) (*model.Event, error) {
	pData, err := os.ReadFile(eventPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения события: %w", err)
	}

	var dbEvent model.Event
	if err := yaml.Unmarshal(pData, &dbEvent); err != nil {
		return nil, fmt.Errorf("ошибка структуры файла события %s: %w", eventPath, err)
	}
	return &dbEvent, nil
}
