package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/eterverda/fpvladder/internal/model"
)

// FindLatestId находит самый свежий Id для заданного типа данных (event, rating, pilot).
// Ищет сначала самый большой год, затем самую позднюю дату, затем максимальный номер.
func FindLatestId(baseDir string, dataType string) (model.Id, error) {
	typeDir := ResolveTypePath(baseDir, dataType)

	// Ищем последнюю папку года
	year := lastDirEntry(typeDir)
	if year == "" {
		return "", nil
	}

	// Ищем последнюю папку даты (месяц-день)
	dateDir := filepath.Join(typeDir, year)
	date := lastDirEntry(dateDir)
	if date == "" {
		return "", nil
	}

	// Ищем максимальный номер файла в этой папке
	files, err := os.ReadDir(filepath.Join(dateDir, date))
	if err != nil || len(files) == 0 {
		return "", nil
	}

	maxSeq := -1
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		// Отрезаем расширение: "123.yaml" -> "123"
		name := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
		if seq, err := strconv.Atoi(name); err == nil && seq > maxSeq {
			maxSeq = seq
		}
	}

	if maxSeq == -1 {
		return "", nil
	}
	return model.Id(fmt.Sprintf("%s/%s/%d", year, date, maxSeq)), nil
}

// GenerateNextId создает новый Id с текущей датой и следующим порядковым номером.
func GenerateNextId(baseDir string, dataType string, date model.Date) (model.Id, error) {
	latest, err := FindLatestId(baseDir, dataType)
	if err != nil {
		return "", err
	}

	nextSeq := 1
	if latest != "" {
		// Разбираем Id "2025/12-28/17", чтобы достать номер 17
		parts := strings.Split(string(latest), "/")
		if len(parts) == 3 {
			if seq, err := strconv.Atoi(parts[2]); err == nil {
				nextSeq = seq + 1
			}
		}
	}

	return model.FormatId(date, nextSeq), nil
}

func ResolveTypePath(baseDir string, dataType string) string {
	return filepath.Join(baseDir, dataType)
}

// ResolveIdPath превращает Id в полный путь к файлу с расширением yaml
func ResolveIdPath(baseDir string, dataType string, id model.Id) string {
	return ResolveIdPathExt(baseDir, dataType, id, "yaml")
}

func ResolveIdPathExt(baseDir string, dataType string, id model.Id, extension string) string {
	return filepath.Join(ResolveTypePath(baseDir, dataType), string(id)) + "." + extension
}

// lastDirEntry — вспомогательная функция для поиска последней подпапки
func lastDirEntry(path string) string {
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) == 0 {
		return ""
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}

	if len(dirs) == 0 {
		return ""
	}
	sort.Strings(dirs)
	return dirs[len(dirs)-1]
}

func ListIds(baseDir, dataType string) ([]model.Id, error) {
	var ids []model.Id

	typeDir := ResolveTypePath(baseDir, dataType)

	// Читаем годы (первый уровень: YYYY/)
	yearEntries, err := os.ReadDir(typeDir)
	if err != nil {
		return nil, err
	}

	for _, yearEntry := range yearEntries {
		if !yearEntry.IsDir() {
			continue
		}
		yearDir := filepath.Join(typeDir, yearEntry.Name())

		// Читаем месяцы-дни (второй уровень: YYYY/MM-DD/)
		monthDayEntries, err := os.ReadDir(yearDir)
		if err != nil {
			continue
		}

		for _, monthDayEntry := range monthDayEntries {
			if !monthDayEntry.IsDir() {
				continue
			}
			monthDayDir := filepath.Join(yearDir, monthDayEntry.Name())

			// Читаем файлы (третий уровень: YYYY/MM-DD/N.yaml)
			files, err := os.ReadDir(monthDayDir)
			if err != nil {
				continue
			}

			for _, file := range files {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
					continue
				}

				// Формируем ID: YYYY/MM-DD/N
				num := strings.TrimSuffix(file.Name(), ".yaml")
				id := filepath.ToSlash(filepath.Join(yearEntry.Name(), monthDayEntry.Name(), num))
				ids = append(ids, model.Id(id))
			}
		}
	}

	return ids, nil
}
