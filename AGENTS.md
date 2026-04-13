# FPV Ladder - Project Guide for Agents

## Quick Overview
FPV Ladder — система рейтинга для FPV-дронов. Генерирует статический сайт с турнирными таблицами.

**Ключевая команда:**
```bash
go run main/main.go generate
```
Генерирует сайт из `./data/` → `./build/`

---

## Структура проекта

```
FpvLadder/
├── main/main.go              # CLI точка входа (cobra)
├── internal/
│   ├── site/                 # 🎯 ГЕНЕРАЦИЯ САЙТА (важно!)
│   ├── db/                   # Работа с YAML-базой
│   ├── model/                # Структуры данных
│   ├── elo/                  # Расчёт рейтингов ELO
│   └── prepare/              # TUI редактор событий
├── data/                     # YAML база данных
└── build/                    # Сгенерированный сайт
```

---

## 🎯 Генерация сайта (internal/site/)

### Главный файл: Generate.go

**Основная функция:**
```go
func Generate(baseDir, outDir string) error
// baseDir = "./data" (вход)
// outDir = "./build" (выход)
```

**Что генерирует:**
1. `index.html` — главная с таблицами пилотов и событий
2. `pilot/*.html` — страницы пилотов
3. `event/*.html` — страницы прошедших событий
4. `future_event/*.html` — страницы будущих событий
5. `calendar.ics` — календарь в формате iCal
6. `manifest.html` — манифест
7. Копирует `styles.css` и `scripts.js`

### Шаблоны (tmpl-файлы)

**Базовые:**
- `header.tmpl` — шапка сайта (логотип, настройки)
- `symbols.tmpl` — SVG иконки
- `widget.tmpl` — виджет избранного (сердечки, звёзды)

**Страницы:**
- `index.tmpl` — главная страница
- `pilot.tmpl` — страница пилота
- `event.tmpl` — страница события
- `future_event.tmpl` — будущее событие
- `manifest.tmpl` — манифест

**Переменные для шаблонов:**
```go
// indexPage — для index.tmpl
type indexPage struct {
    Title       string
    GeneratedAt model.Date
    Classes     []*indexClassData  // данные по классам (75/125/200/330mm)
}

// pilotPage — для pilot.tmpl  
type pilotPage struct {
    Id      model.Id
    Name    string
    Classes []*pilotClassData
}

// eventPage — для event.tmpl и future_event.tmpl
type eventPage struct {
    Id          model.Id
    Name        string
    Date        string
    Description template.HTML  // только для future_event
    Results     []*resultRecord
}
```

### Стили и скрипты

- `styles.css` — ВСЕ стили в одном файле (не разбивать на модули!)
  - CSS переменные для тем (light/dark)
  - Цвета классов: --class-75, --class-125, --class-200, --class-330
  - Использует data-class атрибуты для динамических цветов
  
- `scripts.js` — клиентский JavaScript
  - Переключение темы
  - Поиск
  - Избранное
  - Переключение классов

### Классы дронов

```go
var ClassDisplayNames = map[model.Class]string{
    model.Class75mm:  "75мм",
    model.Class125mm: "125мм", 
    model.Class200mm: "200мм",
    model.Class330mm: "330мм",
}

var ClassParamValues = map[model.Class]string{
    model.Class75mm:  "75mm",
    model.Class125mm: "125mm",
    model.Class200mm: "200mm",
    model.Class330mm: "330mm",
}
```

### Функции шаблонов

```go
var templateFuncs = template.FuncMap{
    "now": func() string { return time.Now().Format("20060102150405") },
}
```

---

## 💾 База данных (internal/db/)

Хранится в `./data/` как YAML-файлы:
- `data/pilot/YYYY/MM-DD/N.yaml` — пилоты
- `data/event/YYYY/MM-DD/N.yaml` — события
- `data/future_event/YYYY/MM-DD/N.yaml` — будущие события

**Ключевые функции:**
- `ReadAllPilots()` — читает всех пилотов
- `ReadEvent()` — читает событие по ID
- `GenerateIndex()` — генерирует индекс для поиска

---

## 📊 Модели (internal/model/)

**Основные структуры:**

```go
type Pilot struct {
    Id      Id
    Name    string
    Careers []Career  // карьера по классам
}

type Event struct {
    Id      Id
    Date    Date
    Name    string
    Class   Class
    Pilots  []PilotEntry
}

type FutureEvent struct {
    Id          Id
    Date        Date
    Name        string
    Classes     []Class  // может быть несколько классов!
    Description string   // Markdown
}
```

---

## ⚡ ELO Рейтинги (internal/elo/)

Расчёт изменения рейтинга по системе ELO.

```go
type Input struct {
    Position int    // место в гонке
    Team     int    // номер команды (0 = нет)
    Rating   int    // текущий рейтинг
}

func GroupKCalc(inputs []Input) []int  // возвращает дельты
```

---

## 🖥️ CLI Команды (main/main.go)

```bash
# Добавить событие и пересчитать рейтинги
droon install event.yaml
droon install event.yaml --auto-create-pilots  # автосоздание пилотов

# Создать пилота
droon pilot "Имя Фамилия"
droon pilot "Имя" -d 2024-01-15

# Сгенерировать сайт
droon generate

# Экспорт в CSV
droon csv output.csv -c "drone-racing > 75mm"

# TUI редактор событий
droon prepare
droon prepare draft.yaml
```

---

## 🔧 Частые задачи

### Добавить новое поле в шаблон
1. Добавить поле в структуру (indexPage/pilotPage/eventPage) в Generate.go
2. Заполнить данные в соответствующей generate* функции
3. Использовать в .tmpl файле: `{{ .NewField }}`

### Изменить стили
1. Править только `internal/site/styles.css`
2. НЕ разбивать на файлы (таковы требования)
3. Использовать CSS переменные для тем
4. Для классовых цветов использовать data-class атрибуты

### Добавить новую страницу
1. Создать `newpage.tmpl`
2. Создать структуру newPageData
3. Добавить generateNewPage() функцию
4. Вызвать из Generate()

### Отладка генерации
```bash
# Сгенерировать и посмотреть
rm -rf build/ && go run main/main.go generate
open build/index.html

# Проверить конкретный пилота
open build/pilot/2024/01-15/1.html
```

---

## ⚠️ Важные нюансы

1. **Пути:** Всегда использовать `db.ResolveIdPathExt()` для генерации путей
2. **HTML escaping:** Использовать `template.HTML` для вставки готового HTML (описания событий)
3. **Markdown:** `md2html()` конвертирует Markdown → HTML для future_event
4. **Даты:** Формат `YYYY-MM-DD`, тип `model.Date`
5. **ID:** Формат `YYYY/MM-DD/N`, тип `model.Id`
6. **CSS:** НЕ использовать CSS-модули, держать всё в одном файле

---

## Тестирование сайта

```bash
# Локальный сервер для тестирования
cd build && python3 -m http.server 8000
# или
cd build && npx serve

# Открыть
open http://localhost:8000
```
