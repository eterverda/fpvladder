package prepare

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/eterverda/fpvladder/internal/db"
	"github.com/eterverda/fpvladder/internal/model"
)

// Цветовая палитра приложения
const (
	colorAccent  = "#00AAAA" // Бирюзовый (основной акцент)
	colorError   = "#FF0000" // Красный для ошибок
	colorWarning = "#FF8800" // Оранжевый для предупреждений
	colorDim     = "8"       // Серый для placeholder/виртуальных элементов
	colorHelp    = "7"       // Светло-серый для подсказок
	colorBlack   = "#000000" // Чёрный
)

// Общие стили приложения
var (
	// Стиль для placeholder и виртуальных элементов (серый)
	styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))

	// Стиль для подсказок (светло-серый)
	styleHelp = lipgloss.NewStyle().Foreground(lipgloss.Color(colorHelp))

	// Стиль для акцентных элементов (бирюзовый)
	styleAccent = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent))

	// Стиль для выделенных элементов (реверс)
	styleSelected = lipgloss.NewStyle().Reverse(true)

	// Стиль для ошибок (красный)
	styleError = lipgloss.NewStyle().Foreground(lipgloss.Color(colorError))
)

// Screen определяет текущий экран
type Screen int

const (
	ScreenMain     Screen = iota
	ScreenEditName        // Редактирование имени пилота в модалке
	ScreenEditEventDate
	ScreenEditEventName
	ScreenEditEventOrganizer
	ScreenSelectClass      // Выпадающий список для выбора класса
	ScreenFindId           // Поиск ID пилота в базе
	ScreenConfirmExit      // Подтверждение выхода
	ScreenConfirmOverwrite // Подтверждение перезаписи файла
	ScreenSaveAs
)

// SaveAsMode определяет режим сохранения
type SaveAsMode int

const (
	SaveAsModeExit SaveAsMode = iota // Сохранить и выйти (при подтверждении выхода)
	SaveAsModeSave                   // Сохранить и вернуться (при нажатии 's')
)

// homeDir кэширует домашний каталог для преобразования путей
var homeDir, _ = os.UserHomeDir()

// toTildePath преобразует абсолютный путь в путь с ~ для отображения
func toTildePath(absPath string) string {
	if homeDir != "" && strings.HasPrefix(absPath, homeDir) {
		return "~" + absPath[len(homeDir):]
	}
	return absPath
}

// fromTildePath преобразует путь с ~ в абсолютный для использования
func fromTildePath(tildePath string) string {
	if homeDir != "" && strings.HasPrefix(tildePath, "~") {
		return homeDir + tildePath[1:]
	}
	return tildePath
}

// Model TUI
type TUIModel struct {
	EventModel *EventModel
	Screen     Screen
	Cursor     int // Индекс пилота в плоском списке (глобальный индекс)
	Width      int
	Height     int
	Focus      int // 0=date, 1=name, 2=organizer, 3=class, 4+=table (по пилотам)

	// Для редактирования
	TextInput textinput.Model
	EditRow   int // Глобальный индекс редактируемого пилота

	// Для редактирования даты (YYYY-MM-DD)
	DateYear  int
	DateMonth int
	DateDay   int
	DateFocus int // 0=year, 1=month, 2=day

	// Для поиска
	FindResults []FindResult
	FindCursor  int

	// Для выбора класса (выпадающий список)
	ClassCursor int // Индекс выбранного класса в списке

	// Для валидации имени пилота
	ShowValidationError bool // Показывать ли ошибку валидации

	// Кэш максимального номера пилота в базе (для генерации виртуальных ID)
	MaxPilotNum int // -1 означает "не инициализировано"

	// Поддержка Kitty Keyboard Protocol
	supportsBaseCode bool

	// Режим сохранения (для ScreenSaveAs)
	SaveAsMode SaveAsMode

	// Путь для подтверждения перезаписи (для ScreenConfirmOverwrite)
	OverwritePath string

	// Viewport для скроллируемого контента (всё кроме подсказки)
	Viewport viewport.Model
}

// Init инициализирует TUI
func (m TUIModel) Init() tea.Cmd {
	flags := ansi.KittyDisambiguateEscapeCodes | ansi.KittyReportAlternateKeys | ansi.KittyReportAssociatedKeys
	return tea.Sequence(
		tea.Raw(ansi.KittyKeyboard(flags, 1)),
		tea.Raw(ansi.RequestKittyKeyboard),
	)
}

// getFocusLine возвращает номер строки с фокусом (0-based) в контенте
func (m TUIModel) getFocusLine() int {
	// Заголовок: 2 строки ("# Подготовка", "# filename")
	// Поля: date (1), name (1), organizer (2), class (1), pilots: (1) = 8 строк
	// Таблица: заголовок (1) + строки пилотов
	focusLine := m.Focus
	if focusLine >= 4 {
		// В таблице: 9 строк до таблицы + индекс в таблице
		focusLine = 9 + (m.Focus - 4)
	}
	return focusLine
}

// syncViewport обновляет контент viewport и скроллит к фокусу с отступом 3 строки
func (m *TUIModel) syncViewport() {
	// Обновляем контент
	m.Viewport.SetContent(m.renderMainContent())

	focusLine := m.getFocusLine()

	// Скроллим так, чтобы фокус был виден с отступом 3 строки от края
	viewportHeight := m.Viewport.Height()
	currentYOffset := m.Viewport.YOffset()

	// Желаемая верхняя граница: фокус - 3 строки отступа
	desiredTop := focusLine - 3
	if desiredTop < 0 {
		desiredTop = 0
	}

	// Желаемая нижняя граница: фокус + 3 строки отступа
	desiredBottom := focusLine + 3 - viewportHeight + 1
	if desiredBottom < 0 {
		desiredBottom = 0
	}

	// Если фокус выше текущей видимой области
	if focusLine < currentYOffset+3 {
		m.Viewport.SetYOffset(desiredTop)
	} else if focusLine > currentYOffset+viewportHeight-1-3 {
		// Если фокус ниже текущей видимой области
		m.Viewport.SetYOffset(desiredBottom)
	}
}

// initMaxPilotNum находит максимальный номер пилота в базе и текущих данных
func (m *TUIModel) initMaxPilotNum() {
	if m.MaxPilotNum >= 0 {
		return
	}
	m.recalcMaxPilotNum()
}

// recalcMaxPilotNum пересчитывает максимальный номер из базы и текущих виртуальных ID
func (m *TUIModel) recalcMaxPilotNum() {
	maxNum := 0

	// Проверяем базу данных
	allPilotIds, _ := db.ListIds("./data", "pilot")
	for _, id := range allPilotIds {
		idStr := string(id)
		parts := strings.Split(idStr, "/")
		if len(parts) == 3 {
			num := 0
			fmt.Sscanf(parts[2], "%d", &num)
			if num > maxNum {
				maxNum = num
			}
		}
	}

	// Проверяем текущие виртуальные ID в event model
	for _, row := range m.EventModel.Rows {
		for _, pilot := range row.Pilots {
			idStr := string(pilot.Id)
			parts := strings.Split(idStr, "/")
			if len(parts) == 3 {
				num := 0
				fmt.Sscanf(parts[2], "%d", &num)
				if num > maxNum {
					maxNum = num
				}
			}
		}
	}

	m.MaxPilotNum = maxNum
}

// getTotalPilots возвращает общее количество пилотов
func (m TUIModel) getTotalPilots() int {
	return TotalPilots(m.EventModel.Rows)
}

// getPilot возвращает пилота по глобальному индексу
func (m TUIModel) getPilot(globalIndex int) (teamIdx int, pilotIdx int, pilot *PilotRow, ok bool) {
	return m.EventModel.GetPilot(globalIndex)
}

// getTeamForPilot возвращает команду для пилота
func (m TUIModel) getTeamForPilot(globalIndex int) (teamIdx int, team *TeamRow, ok bool) {
	return m.EventModel.GetTeamForPilot(globalIndex)
}

// isVirtualPilot проверяет, является ли пилот виртуальным
func (m TUIModel) isVirtualPilot(globalIndex int) bool {
	return m.EventModel.IsVirtualPilot(globalIndex)
}

// virtualIdCache кэширует виртуальные ID для всех незарегистрированных пилотов
type virtualIdCache struct {
	ids map[int]string // глобальный индекс пилота -> виртуальный ID
}

// generateVirtualIds генерирует виртуальные ID для всех незарегистрированных пилотов
// Всегда назначает ID последовательно по алфавиту имен
func (m TUIModel) generateVirtualIds() virtualIdCache {
	cache := virtualIdCache{ids: make(map[int]string)}

	eventDate := time.Time(m.EventModel.Event.Date)
	year := eventDate.Year()
	month := int(eventDate.Month())
	day := eventDate.Day()

	// Находим максимальный номер из базы данных (игнорируем текущие виртуальные)
	maxNum := 0
	allPilotIds, _ := db.ListIds("./data", "pilot")
	for _, id := range allPilotIds {
		idStr := string(id)
		parts := strings.Split(idStr, "/")
		if len(parts) == 3 {
			num := 0
			fmt.Sscanf(parts[2], "%d", &num)
			if num > maxNum {
				maxNum = num
			}
		}
	}

	type pilotInfo struct {
		index int
		name  string
	}
	var unregistered []pilotInfo

	globalIdx := 0
	for _, row := range m.EventModel.Rows {
		for _, pilot := range row.Pilots {
			// Виртуальный ID назначаем только тем у кого IdKind == Virtual
			if pilot.IdKind == IdKindVirtual {
				unregistered = append(unregistered, pilotInfo{index: globalIdx, name: pilot.Name})
			}
			globalIdx++
		}
	}

	// Сортируем по алфавиту (пустое имя — самое последнее)
	sort.Slice(unregistered, func(i, j int) bool {
		if unregistered[i].name == "" && unregistered[j].name != "" {
			return false
		}
		if unregistered[i].name != "" && unregistered[j].name == "" {
			return true
		}
		return strings.ToLower(unregistered[i].name) < strings.ToLower(unregistered[j].name)
	})

	// Назначаем ID последовательно
	for i, p := range unregistered {
		newNum := maxNum + 1 + i
		cache.ids[p.index] = fmt.Sprintf("%04d/%02d-%02d/%d", year, month, day, newNum)
	}

	return cache
}

// getVirtualId возвращает виртуальный ID для пилота
// Всегда генерирует ID для указанного индекса, даже если пилот имеет другой ID
func (m TUIModel) getVirtualId(globalIndex int) string {
	// Получаем базовый maxNum из базы
	maxNum := 0
	allPilotIds, _ := db.ListIds("./data", "pilot")
	for _, id := range allPilotIds {
		idStr := string(id)
		parts := strings.Split(idStr, "/")
		if len(parts) == 3 {
			num := 0
			fmt.Sscanf(parts[2], "%d", &num)
			if num > maxNum {
				maxNum = num
			}
		}
	}

	// Считаем сколько виртуальных ID перед этим индексом
	virtualCount := 0
	currentIdx := 0
	for _, row := range m.EventModel.Rows {
		for _, pilot := range row.Pilots {
			if currentIdx == globalIndex {
				// Нашли нужный индекс
				eventDate := time.Time(m.EventModel.Event.Date)
				year := eventDate.Year()
				month := int(eventDate.Month())
				day := eventDate.Day()
				newNum := maxNum + 1 + virtualCount
				return fmt.Sprintf("%04d/%02d-%02d/%d", year, month, day, newNum)
			}
			if pilot.IdKind == IdKindVirtual {
				virtualCount++
			}
			currentIdx++
		}
	}
	return ""
}

// getSuggestedVirtualId возвращает виртуальный ID для пилота
func (m TUIModel) getSuggestedVirtualId(globalIndex int) string {
	return m.getVirtualId(globalIndex)
}

// reassignVirtualIds переназначает виртуальные ID всем неидентифицированным пилотам
// Вызывать после идентификации пилота чтобы пересчитать номера по алфавиту
func (m *TUIModel) reassignVirtualIds() {
	cache := m.generateVirtualIds()

	globalIdx := 0
	for i := range m.EventModel.Rows {
		for j := range m.EventModel.Rows[i].Pilots {
			if newId, ok := cache.ids[globalIdx]; ok {
				// Присваиваем виртуальный ID только тем у кого IdKind == Virtual
				if m.EventModel.Rows[i].Pilots[j].IdKind == IdKindVirtual {
					m.EventModel.Rows[i].Pilots[j].Id = model.Id(newId)
				}
			}
			globalIdx++
		}
	}
}

// NewTUIModel создаёт новую модель TUI
func NewTUIModel(eventModel *EventModel) TUIModel {
	ti := textinput.New()
	ti.Focus()
	ti.Prompt = ""

	if eventModel.IsNew {
		eventModel.Event.Date = model.Today()
	}

	// Создаём viewport (размер будет установлен при получении WindowSizeMsg)
	vp := viewport.New()

	tuiModel := TUIModel{
		EventModel:  eventModel,
		Screen:      ScreenMain,
		Cursor:      0,
		Focus:       4, // По умолчанию фокус на таблице
		TextInput:   ti,
		MaxPilotNum: -1,
		Viewport:    vp,
	}

	tuiModel.initMaxPilotNum()

	// Инициализируем IdKind для пилотов без ID
	// (при загрузке из файла у всех пилотов IdKind == 0, что соответствует IdKindVirtual)
	// Но для пилотов с непустым Id нужно проверить есть ли suggested из базы
	for i := range eventModel.Rows {
		for j := range eventModel.Rows[i].Pilots {
			pilot := &eventModel.Rows[i].Pilots[j]
			if pilot.Id == "" && pilot.Name != "" {
				// Ищем в базе
				results := eventModel.FindPilotsByName(pilot.Name)
				if len(results) > 0 {
					pilot.Id = model.Id(results[0].Id)
					pilot.IdKind = IdKindSuggested
				} else {
					pilot.IdKind = IdKindVirtual
				}
			} else if pilot.Id != "" {
				pilot.IdKind = IdKindExplicit
			}
		}
	}

	// Начальное назначение виртуальных ID
	tuiModel.reassignVirtualIds()

	// Начальная синхронизация viewport (будет выполнена после получения размеров окна)
	// tuiModel.syncViewport()

	return tuiModel
}
