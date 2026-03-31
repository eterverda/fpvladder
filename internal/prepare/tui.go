package prepare

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/eterverda/fpvladder/internal/db"
	"github.com/eterverda/fpvladder/internal/model"
	"github.com/mattn/go-runewidth"
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
	ScreenMain         Screen = iota
	ScreenEditName            // Редактирование имени пилота в модалке
	ScreenEditPosition        // Редактирование позиции пилота
	ScreenEditTeam            // Редактирование команды пилота
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
// Например: /home/user/FpvLadder/2025-1.yaml -> ~/FpvLadder/2025-1.yaml
func toTildePath(absPath string) string {
	if homeDir != "" && strings.HasPrefix(absPath, homeDir) {
		return "~" + absPath[len(homeDir):]
	}
	return absPath
}

// fromTildePath преобразует путь с ~ в абсолютный для использования
// Например: ~/FpvLadder/2025-1.yaml -> /home/user/FpvLadder/2025-1.yaml
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
	Cursor     int // Индекс строки в таблице
	Width      int
	Height     int
	Focus      int // 0=date, 1=name, 2=organizer, 3=class, 4+=table

	// Для редактирования
	TextInput textinput.Model
	EditRow   int

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
	// BaseCode доступен только когда терминал поддерживает KittyReportAlternateKeys
	supportsBaseCode bool

	// Режим сохранения (для ScreenSaveAs)
	SaveAsMode SaveAsMode

	// Путь для подтверждения перезаписи (для ScreenConfirmOverwrite)
	OverwritePath string
}

// Init инициализирует TUI
func (m TUIModel) Init() tea.Cmd {
	// Включаем Kitty Keyboard Protocol с флагом KittyReportAlternateKeys
	// Это необходимо для получения BaseCode (физическая раскладка клавиш)
	// Bubble Tea v2 по умолчанию не запрашивает этот флаг
	// See: https://github.com/charmbracelet/bubbletea/issues/1591
	flags := ansi.KittyDisambiguateEscapeCodes | ansi.KittyReportAlternateKeys | ansi.KittyReportAssociatedKeys
	return tea.Sequence(
		// Отправляем escape-последовательность для включения Kitty Keyboard Protocol
		tea.Raw(ansi.KittyKeyboard(flags, 1)),
		// Запрашиваем текущие флаги чтобы получить KeyboardEnhancementsMsg
		tea.Raw(ansi.RequestKittyKeyboard),
	)
}

// virtualIdCache кэширует виртуальные ID для всех незарегистрированных пилотов
type virtualIdCache struct {
	ids map[int]string // индекс строки -> виртуальный ID
}

// initMaxPilotNum находит максимальный номер пилота в базе (если ещё не инициализирован)
func (m *TUIModel) initMaxPilotNum() {
	if m.MaxPilotNum >= 0 {
		return // Уже инициализировано
	}

	maxNum := 0
	allPilotIds, _ := db.ListIds("./data", "pilot")
	for _, id := range allPilotIds {
		// Парсим ID формата YYYY/MM-DD/N
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
	m.MaxPilotNum = maxNum
}

// generateVirtualIds генерирует виртуальные ID для всех незарегистрированных пилотов
// Возвращает кэш с ID, где ключ — индекс строки в EventModel.Rows
func (m TUIModel) generateVirtualIds() virtualIdCache {
	cache := virtualIdCache{ids: make(map[int]string)}

	// Получаем дату события
	eventDate := time.Time(m.EventModel.Event.Date)
	year := eventDate.Year()
	month := int(eventDate.Month())
	day := eventDate.Day()

	// Используем кэшированный максимальный номер
	maxNum := m.MaxPilotNum
	if maxNum < 0 {
		maxNum = 0
	}

	// Собираем незарегистрированных пилотов (с пустым Id)
	type pilotInfo struct {
		index int
		name  string
	}
	var unregistered []pilotInfo

	for i, row := range m.EventModel.Rows {
		if row.Id == "" {
			unregistered = append(unregistered, pilotInfo{index: i, name: row.Name})
		}
	}

	// Сортируем по алфавиту (пустое имя — самое последнее)
	sort.Slice(unregistered, func(i, j int) bool {
		// Пустое имя всегда последнее
		if unregistered[i].name == "" && unregistered[j].name != "" {
			return false
		}
		if unregistered[i].name != "" && unregistered[j].name == "" {
			return true
		}
		return strings.ToLower(unregistered[i].name) < strings.ToLower(unregistered[j].name)
	})

	// Присваиваем номера начиная с maxNum + 1
	for i, p := range unregistered {
		newNum := maxNum + 1 + i
		cache.ids[p.index] = fmt.Sprintf("%04d/%02d-%02d/%d", year, month, day, newNum)
	}

	return cache
}

// getVirtualId возвращает виртуальный ID для строки (генерирует при первом вызове)
func (m TUIModel) getVirtualId(rowIndex int) string {
	cache := m.generateVirtualIds()
	if id, ok := cache.ids[rowIndex]; ok {
		return id
	}
	return ""
}

// getNextVirtualId возвращает просто следующий виртуальный ID (max + 1)
// Используется для предложения нового ID в пикере
func (m TUIModel) getNextVirtualId() string {
	// Получаем дату события
	eventDate := time.Time(m.EventModel.Event.Date)
	year := eventDate.Year()
	month := int(eventDate.Month())
	day := eventDate.Day()

	// Используем кэшированный максимальный номер
	maxNum := m.MaxPilotNum
	if maxNum < 0 {
		maxNum = 0
	}

	// Просто max + 1
	newNum := maxNum + 1
	return fmt.Sprintf("%04d/%02d-%02d/%d", year, month, day, newNum)
}

// getSuggestedVirtualId возвращает предложенный виртуальный ID на основе поиска в базе
// Возвращает: id, suffix (*, ? или ""), foundCount
// * = ровно 1 найден, ? = несколько найдено (берём первый), "" = не найден (новый ID)
func (m TUIModel) getSuggestedVirtualId(rowIndex int) (string, string, int) {
	row := m.EventModel.Rows[rowIndex]
	if row.Name == "" {
		return m.getNextVirtualId(), "", 0
	}

	// Ищем в базе
	results := m.EventModel.FindPilotsByName(row.Name)

	if len(results) == 0 {
		// Не найден — просто новый виртуальный ID без суффикса
		return m.getNextVirtualId(), "", 0
	} else if len(results) == 1 {
		// Ровно 1 найден — его ID с *
		return results[0].Id, "*", 1
	} else {
		// Несколько найдено — первый с ?
		return results[0].Id, "?", len(results)
	}
}

// isVirtualId проверяет, является ли ID виртуальным (сгенерированным)
// Виртуальный ID — это ID который не соответствует реальному ID из базы
// NewTUIModel создаёт новую модель TUI
func NewTUIModel(eventModel *EventModel) TUIModel {
	// Настраиваем text input
	ti := textinput.New()
	ti.Focus()
	ti.Prompt = ""

	// Для нового события ставим текущую дату
	if eventModel.IsNew {
		eventModel.Event.Date = model.Today()
	}

	tuiModel := TUIModel{
		EventModel:  eventModel,
		Screen:      ScreenMain,
		Cursor:      0,
		Focus:       4, // По умолчанию фокус на таблице
		TextInput:   ti,
		MaxPilotNum: -1, // Не инициализировано, будет вычислено при первом рендере
	}

	// Инициализируем максимальный номер пилота в базе
	tuiModel.initMaxPilotNum()

	return tuiModel
}

// Update обрабатывает сообщения
func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyboardEnhancementsMsg:
		// Проверяем, поддерживает ли терминал KittyReportAlternateKeys
		// Это необходимо для работы BaseCode (физическая раскладка клавиш)
		m.supportsBaseCode = msg.Flags&ansi.KittyReportAlternateKeys != 0
		return m, nil

	case tea.MouseClickMsg:
		// Обработка кликов мышью
		if m.Screen == ScreenMain {
			return m.handleMouseClick(msg)
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	// Обновляем text input если активен
	if m.Screen == ScreenEditName || m.Screen == ScreenSaveAs ||
		m.Screen == ScreenEditEventDate || m.Screen == ScreenEditEventName ||
		m.Screen == ScreenEditEventOrganizer {
		var cmd tea.Cmd
		m.TextInput, cmd = m.TextInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// keyCode возвращает код клавиши с учётом поддержки BaseCode
// Если терминал поддерживает KittyReportAlternateKeys, используем BaseCode (физическая раскладка)
// Иначе используем Code (может отличаться при не-латинских раскладках)
func (m TUIModel) keyCode(msg tea.KeyPressMsg) rune {
	if m.supportsBaseCode && msg.BaseCode != 0 {
		return msg.BaseCode
	}
	return msg.Code
}

func (m TUIModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// esc - всегда выход
	if msg.String() == "esc" {
		if m.EventModel.Modified {
			m.Screen = ScreenConfirmExit
			return m, nil
		}
		return m, tea.Quit
	}

	switch m.Screen {
	case ScreenMain:
		return m.handleMainKey(msg)
	case ScreenEditName,
		ScreenEditEventDate, ScreenEditEventName, ScreenEditEventOrganizer:
		return m.handleEditKey(msg)
	case ScreenSelectClass:
		return m.handleSelectClassKey(msg)
	case ScreenFindId:
		return m.handleFindIdKey(msg)
	case ScreenConfirmExit:
		return m.handleConfirmKey(msg)
	case ScreenConfirmOverwrite:
		return m.handleConfirmOverwriteKey(msg)
	case ScreenSaveAs:
		return m.handleSaveAsKey(msg)
	}

	return m, nil
}

// handleMouseClick обрабатывает клики мышью на главном экране
func (m TUIModel) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	// Клики работают только на главном экране
	if m.Screen != ScreenMain {
		return m, nil
	}

	// Определяем координаты клика
	x, y := msg.X, msg.Y

	// Проверяем, что клик в пределах таблицы пилотов
	// Структура экрана:
	// 0: # Подготовка мероприятия
	// 1: # filename: ...
	// 2: date: ...
	// 3: name: ...
	// 4: organizer:
	// 5:     name: ...
	// 6: class: ...
	// 7: pilots:
	// 8:     pos:  id:   name: (заголовок таблицы)
	// 9+: данные таблицы
	tableStartY := 9 // Y-координата первой строки данных таблицы
	tableStartX := 4 // Отступ таблицы (indent)

	// Проверяем, что клик в области таблицы (с учётом отступа)
	if x < tableStartX || y < tableStartY {
		return m, nil
	}

	// Вычисляем индекс строки в таблице (минус заголовок таблицы)
	rowIndex := y - tableStartY

	// Проверяем, что индекс в пределах таблицы
	if rowIndex >= 0 && rowIndex < len(m.EventModel.Rows) {
		// Устанавливаем фокус на выбранную строку
		m.Focus = 4 + rowIndex
		m.Cursor = rowIndex
	}

	return m, nil
}

func (m TUIModel) handleMainKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Получаем key code: BaseCode для терминалов с Kitty Protocol,
	// иначе используем Code как fallback
	keyCode := m.keyCode(msg)

	// Используем BaseCode для буквенных шорткатов (независимо от раскладки)
	// и String() для специальных клавиш
	switch {
	// Навигация стрелками (специальные клавиши - через String())
	case msg.String() == "up" || keyCode == 'k':
		if m.Focus > 0 {
			m.Focus--
		}
	case msg.String() == "down" || keyCode == 'j':
		if m.Focus < 4+len(m.EventModel.Rows)-1 {
			m.Focus++
		}
	case msg.String() == "tab":
		if m.Focus < 3 {
			m.Focus++
		} else {
			m.Focus = 4
		}
	case msg.String() == "shift+tab":
		if m.Focus > 4 {
			m.Focus = 3
		} else if m.Focus > 0 {
			m.Focus--
		}
	case msg.String() == "enter":
		switch m.Focus {
		case 0:
			m.startEditEventField(ScreenEditEventDate)
		case 1:
			m.startEditEventField(ScreenEditEventName)
		case 2:
			m.startEditEventField(ScreenEditEventOrganizer)
		case 3:
			m.startSelectClass()
		default:
			// Редактируем имя пилота
			m.Cursor = m.Focus - 4
			m.startEditNameModal()
		}
	// e - редактировать имя пилота
	case keyCode == 'e':
		if m.Focus >= 4 {
			m.Cursor = m.Focus - 4
			m.startEditNameModal()
		} else if m.Focus == 1 {
			// Для названия события - модалка
			m.startEditEventField(ScreenEditEventName)
		} else if m.Focus == 2 {
			// Для организатора - модалка
			m.startEditEventField(ScreenEditEventOrganizer)
		}
	// i - найти ID пилота в базе (не работает на виртуальной строке)
	case keyCode == 'i':
		if m.Focus >= 4 {
			m.Cursor = m.Focus - 4
			// Проверяем, что это не виртуальная строка
			if !m.EventModel.Rows[m.Cursor].Virtual {
				m.startFindId()
			}
		}
	// * - принять всех (однозначно идентифицированных из базы)
	case keyCode == '*':
		m.acceptAllIdentified()
	// delete/backspace - удалить
	case msg.String() == "delete" || msg.String() == "backspace":
		if m.Focus >= 4 {
			m.Cursor = m.Focus - 4
			if !m.EventModel.Rows[m.Cursor].Virtual {
				m.EventModel.DeleteRow(m.Cursor)
				if m.Cursor >= len(m.EventModel.Rows) {
					m.Cursor = len(m.EventModel.Rows) - 1
					m.Focus = 4 + m.Cursor
				}
			}
		}
	// s - сохранить
	case keyCode == 's':
		if m.EventModel.Filename == "" {
			// Файл безымянный — показываем диалог сохранения
			m.Screen = ScreenSaveAs
			m.SaveAsMode = SaveAsModeSave
			cwd, _ := os.Getwd()
			path := cwd + "/" + m.EventModel.GenerateFilename()
			m.TextInput.SetValue(toTildePath(path))
			m.TextInput.Focus()
			m.TextInput.CursorEnd()
		} else {
			if err := m.EventModel.Save(); err != nil {
				// TODO: показать ошибку
			}
		}
	// Ctrl+Up/Down - перемещение строк
	case msg.String() == "ctrl+up":
		if m.Focus >= 4 {
			m.Cursor = m.Focus - 4
			newIndex := m.EventModel.MoveRowUp(m.Cursor)
			m.Cursor = newIndex
			m.Focus = 4 + newIndex
		}
	case msg.String() == "ctrl+down":
		if m.Focus >= 4 {
			m.Cursor = m.Focus - 4
			newIndex := m.EventModel.MoveRowDown(m.Cursor)
			m.Cursor = newIndex
			m.Focus = 4 + newIndex
		}
	}

	return m, nil
}

func (m *TUIModel) startEditEventField(screen Screen) {
	m.Screen = screen
	switch screen {
	case ScreenEditEventDate:
		// Парсим дату в отдельные поля
		t := time.Time(m.EventModel.Event.Date)
		m.DateYear = t.Year()
		m.DateMonth = int(t.Month())
		m.DateDay = t.Day()
		m.DateFocus = 0 // Фокус на годе
	case ScreenEditEventName:
		m.TextInput.SetValue(m.EventModel.Event.Name)
		m.TextInput.Focus()
		m.TextInput.CursorEnd()
	case ScreenEditEventOrganizer:
		m.TextInput.SetValue(m.EventModel.Event.Organizer.Name)
		m.TextInput.Focus()
		m.TextInput.CursorEnd()

	}
}

func (m *TUIModel) startEditNameModal() {
	m.EditRow = m.Cursor
	m.Screen = ScreenEditName
	m.ShowValidationError = false
	row := m.EventModel.Rows[m.Cursor]
	m.TextInput.SetValue(row.Name)
	m.TextInput.Focus()
	m.TextInput.CursorEnd()
}

func (m *TUIModel) startSelectClass() {
	m.Screen = ScreenSelectClass
	// Устанавливаем курсор на текущий класс или на первый элемент
	m.ClassCursor = 0
	currentClass := m.EventModel.Event.Class
	for i, opt := range model.KnownClasses {
		if opt == currentClass {
			m.ClassCursor = i
			break
		}
	}
}

// acceptAllIdentified проходит по всем пилотам без ID, ищет однозначные совпадения в базе
// и проставляет реальный ID для тех, кто найден ровно 1 раз
func (m *TUIModel) acceptAllIdentified() {
	for i, row := range m.EventModel.Rows {
		// Пропускаем виртуальные строки и тех, у кого уже есть ID
		if row.Virtual || row.Id != "" {
			continue
		}

		// Ищем в базе
		results := m.EventModel.FindPilotsByName(row.Name)

		// Если найден ровно 1 — проставляем реальный ID
		if len(results) == 1 {
			m.EventModel.Rows[i].Id = model.Id(results[0].Id)
			m.EventModel.Modified = true
		}
	}
}

func (m TUIModel) handleSelectClassKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Screen = ScreenMain
	case "enter":
		m.EventModel.Event.Class = model.KnownClasses[m.ClassCursor]
		m.EventModel.Modified = true
		m.Screen = ScreenMain
	case "up":
		if m.ClassCursor > 0 {
			m.ClassCursor--
		}
	case "down":
		if m.ClassCursor < len(model.KnownClasses)-1 {
			m.ClassCursor++
		}
	}
	return m, nil
}

func (m *TUIModel) startFindId() {
	m.Screen = ScreenFindId
	m.FindCursor = 0

	row := m.EventModel.Rows[m.Cursor]
	searchName := row.Name

	// Ищем в базе
	m.FindResults = m.EventModel.FindPilotsByName(searchName)

	// Добавляем вариант "новый ID" в конец
	// Если у текущей строки уже есть виртуальный ID (пустой Id в модели), показываем его
	// Иначе показываем просто max + 1
	virtualId := m.getVirtualId(m.Cursor)
	if virtualId == "" {
		virtualId = m.getNextVirtualId()
	}
	newIdResult := FindResult{
		Name:   searchName,
		Id:     virtualId,
		Rating: 1200,
	}
	m.FindResults = append(m.FindResults, newIdResult)
}

func (m TUIModel) handleFindIdKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Screen = ScreenMain
	case "enter":
		if len(m.FindResults) > 0 && m.FindCursor < len(m.FindResults) {
			res := m.FindResults[m.FindCursor]
			row := &m.EventModel.Rows[m.Cursor]

			// Проверяем, выбран ли "новый ID" (последний в списке)
			isNewId := (m.FindCursor == len(m.FindResults)-1)

			if isNewId {
				// Новый ID — оставляем пустым в модели (будет виртуальным)
				row.Id = ""
			} else {
				// Реальный ID из базы — сохраняем в модель
				row.Id = model.Id(res.Id)
			}

			if row.Virtual {
				m.EventModel.PromoteVirtualRow(m.Cursor)
			} else {
				m.EventModel.Modified = true
			}
		}
		m.Screen = ScreenMain
	case "up":
		if m.FindCursor > 0 {
			m.FindCursor--
		}
	case "down":
		if m.FindCursor < len(m.FindResults)-1 {
			m.FindCursor++
		}
	}
	return m, nil
}

func (m TUIModel) handleEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Особая обработка для редактора даты
	if m.Screen == ScreenEditEventDate {
		return m.handleDateEditKey(msg)
	}

	switch msg.String() {
	case "esc":
		m.ShowValidationError = false
		m.Screen = ScreenMain
	case "enter":
		value := m.TextInput.Value()

		// Обработка полей события
		switch m.Screen {
		case ScreenEditEventName:
			m.EventModel.Event.Name = value
			m.EventModel.Modified = true
			m.Screen = ScreenMain
		case ScreenEditEventOrganizer:
			m.EventModel.Event.Organizer.Name = value
			m.EventModel.Modified = true
			m.Screen = ScreenMain

		case ScreenEditName:
			// Редактирование имени пилота — валидация на пустое значение
			if strings.TrimSpace(value) == "" {
				// Показываем ошибку валидации
				m.ShowValidationError = true
				return m, nil
			}
			m.ShowValidationError = false
			row := &m.EventModel.Rows[m.EditRow]
			row.Name = value
			// Сбрасываем ID в пустую строку чтобы при рендере
			// getSuggestedVirtualId сделал повторный поиск
			row.Id = ""
			// Если редактировали виртуальную строку и что-то ввели
			if row.Virtual && (row.Position.Int != 0 || row.Name != "" || row.Id != "") {
				m.EventModel.PromoteVirtualRow(m.EditRow)
			} else if !row.Virtual {
				m.EventModel.Modified = true
			}
			m.Screen = ScreenMain
		}
	}

	// При любом редактировании текста скрываем ошибку валидации
	if m.ShowValidationError {
		m.ShowValidationError = false
	}

	var cmd tea.Cmd
	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

// handleDateEditKey обрабатывает клавиши для редактора даты
func (m TUIModel) handleDateEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Screen = ScreenMain
		return m, nil
	case "enter":
		// Сохраняем дату
		m.EventModel.Event.Date = model.Date(time.Date(m.DateYear, time.Month(m.DateMonth), m.DateDay, 0, 0, 0, 0, time.UTC))
		m.EventModel.Modified = true
		m.Screen = ScreenMain
		return m, nil
	case "left":
		if m.DateFocus > 0 {
			m.DateFocus--
		}
		return m, nil
	case "right":
		if m.DateFocus < 2 {
			m.DateFocus++
		}
		return m, nil
	case "up":
		t := time.Date(m.DateYear, time.Month(m.DateMonth), m.DateDay, 0, 0, 0, 0, time.UTC)
		switch m.DateFocus {
		case 0: // год
			t = t.AddDate(1, 0, 0)
		case 1: // месяц
			t = addMonthOverflow(t, 1)
		case 2: // день
			t = t.AddDate(0, 0, 1)
		}
		m.DateYear = t.Year()
		m.DateMonth = int(t.Month())
		m.DateDay = t.Day()
		return m, nil
	case "down":
		t := time.Date(m.DateYear, time.Month(m.DateMonth), m.DateDay, 0, 0, 0, 0, time.UTC)
		switch m.DateFocus {
		case 0: // год
			t = t.AddDate(-1, 0, 0)
		case 1: // месяц
			t = addMonthOverflow(t, -1)
		case 2: // день
			t = t.AddDate(0, 0, -1)
		}
		m.DateYear = t.Year()
		m.DateMonth = int(t.Month())
		m.DateDay = t.Day()
		return m, nil
	}
	return m, nil
}

func (m TUIModel) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Получаем key code: BaseCode для терминалов с Kitty Protocol,
	// иначе используем Code как fallback
	keyCode := m.keyCode(msg)

	switch {
	case msg.String() == "esc":
		// Отмена — возвращаемся к редактированию
		m.Screen = ScreenMain
		return m, nil
	case keyCode == 'n':
		// Выход без сохранения
		return m, tea.Quit
	case keyCode == 'y':
		// Сохраняем и выходим
		if m.EventModel.Filename == "" {
			// Файл безымянный — показываем диалог сохранения
			m.Screen = ScreenSaveAs
			m.SaveAsMode = SaveAsModeExit
			cwd, _ := os.Getwd()
			path := cwd + "/" + m.EventModel.GenerateFilename()
			m.TextInput.SetValue(toTildePath(path))
			m.TextInput.Focus()
			m.TextInput.CursorEnd()
			return m, nil
		}
		// Файл уже имеет имя — сохраняем и выходим
		m.EventModel.Save()
		return m, tea.Quit
	}

	return m, nil
}

func (m TUIModel) handleSaveAsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Отмена сохранения
		if m.SaveAsMode == SaveAsModeExit {
			// Пришли из подтверждения выхода — возвращаемся к редактированию
			m.Screen = ScreenMain
		} else {
			// Пришли по 's' — возвращаемся к редактированию
			m.Screen = ScreenMain
		}
	case "enter":
		// Разворачиваем ~ в абсолютный путь
		absPath := fromTildePath(m.TextInput.Value())

		// Проверяем, существует ли файл (только для новых файлов)
		if m.EventModel.IsNew {
			if _, err := os.Stat(absPath); err == nil {
				// Файл существует — показываем подтверждение перезаписи
				m.OverwritePath = absPath
				m.Screen = ScreenConfirmOverwrite
				return m, nil
			}
		}

		// Файл не существует или не новый — сохраняем
		m.EventModel.Filename = absPath
		if err := m.EventModel.Save(); err == nil {
			if m.SaveAsMode == SaveAsModeExit {
				// Сохранили и выходим
				return m, tea.Quit
			}
			// Сохранили и вернулись к редактированию
			m.Screen = ScreenMain
		}
	}

	var cmd tea.Cmd
	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

func (m TUIModel) handleConfirmOverwriteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyCode := m.keyCode(msg)

	switch {
	case keyCode == 'n' || msg.String() == "esc":
		// Отмена — возвращаемся к диалогу сохранения
		m.Screen = ScreenSaveAs
		m.OverwritePath = ""
		return m, nil
	case keyCode == 'y':
		// Перезаписываем файл
		m.EventModel.Filename = m.OverwritePath
		m.OverwritePath = ""
		if err := m.EventModel.Save(); err == nil {
			if m.SaveAsMode == SaveAsModeExit {
				// Сохранили и выходим
				return m, tea.Quit
			}
			// Сохранили и вернулись к редактированию
			m.Screen = ScreenMain
		}
	}

	return m, nil
}

// View отрисовывает интерфейс
func (m TUIModel) View() tea.View {
	var content string
	switch m.Screen {
	case ScreenEditName,
		ScreenEditEventDate, ScreenEditEventName, ScreenEditEventOrganizer:
		content = m.viewEditModal()
	case ScreenSelectClass:
		content = m.viewSelectClass()
	case ScreenFindId:
		content = m.viewFindId()
	case ScreenConfirmExit:
		content = m.viewConfirmExit()
	case ScreenConfirmOverwrite:
		content = m.viewConfirmOverwrite()
	case ScreenSaveAs:
		content = m.viewSaveAs()
	default:
		content = m.viewMain()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion // Включаем поддержку мыши
	return v
}

func (m TUIModel) viewMain() string {
	var b strings.Builder

	// Заголовок
	b.WriteString(styleDim.Render("# Подготовка мероприятия"))
	b.WriteString("\n")

	// Информация о файле
	filename := m.EventModel.Filename
	if filename == "" {
		filename = "[новый файл]"
	} else {
		filename = toTildePath(filename)
	}
	b.WriteString(styleDim.Render("# filename: " + filename))
	b.WriteString("\n")

	// Общий стиль для меток
	labelStyle := styleAccent
	focusStyle := styleSelected

	// Дата
	dateStr := m.EventModel.Event.Date.String()
	isDatePlaceholder := dateStr == ""
	if isDatePlaceholder {
		dateStr = "[дата]"
	}
	var dateValue string
	if isDatePlaceholder {
		dateValue = styleDim.Render(dateStr)
	} else {
		dateValue = dateStr
	}
	dateLine := labelStyle.Render("date: ") + dateValue
	if m.Focus == 0 {
		dateLine = labelStyle.Render("date: ") + focusStyle.Render(dateStr)
	}
	b.WriteString(dateLine)
	b.WriteString("\n")

	// Название
	nameStr := m.EventModel.Event.Name
	isNamePlaceholder := nameStr == "" || nameStr == "~"
	if isNamePlaceholder {
		nameStr = "[name]"
	}
	var nameValue string
	if isNamePlaceholder {
		nameValue = styleDim.Render(nameStr)
	} else {
		nameValue = nameStr
	}
	nameLine := labelStyle.Render("name: ") + nameValue
	if m.Focus == 1 {
		nameLine = labelStyle.Render("name: ") + focusStyle.Render(nameStr)
	}
	b.WriteString(nameLine)
	b.WriteString("\n")

	// Организатор (две строки: organizer: и name: с отступом)
	b.WriteString(labelStyle.Render("organizer:"))
	b.WriteString("\n")
	orgStr := m.EventModel.Event.Organizer.Name
	isOrgPlaceholder := orgStr == ""
	if isOrgPlaceholder {
		orgStr = "[name]"
	}
	var orgValue string
	if isOrgPlaceholder {
		orgValue = styleDim.Render(orgStr)
	} else {
		orgValue = orgStr
	}
	orgIndent := "    "
	orgLine := orgIndent + labelStyle.Render("name: ") + orgValue
	if m.Focus == 2 {
		orgLine = orgIndent + labelStyle.Render("name: ") + focusStyle.Render(orgStr)
	}
	b.WriteString(orgLine)
	b.WriteString("\n")

	// Класс
	class := m.EventModel.Event.Class
	if class == "" {
		class = model.Class75mm
	}
	classLine := labelStyle.Render("class: ") + string(class)
	if m.Focus == 3 {
		classLine = labelStyle.Render("class: ") + focusStyle.Render(string(class))
	}
	b.WriteString(classLine)
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("pilots:"))
	b.WriteString("\n")

	// Таблица
	metaLines := 8                          // Заголовок + файл + date + name + organizer: + name: + class + пустая
	tableHeight := m.Height - metaLines - 2 // -2 для подсказки
	if tableHeight < 5 {
		tableHeight = 5
	}

	// Заголовки таблицы: pos | id | name (с отступом 4)
	indent := "    "
	colPlaceWidth := 6
	colIDWidth := 15
	colNameWidth := max(m.Width-colIDWidth-colPlaceWidth-4-4, 20) // -4 для отступа

	b.WriteString(indent)
	b.WriteString(labelStyle.Render(fmt.Sprintf("%-*s %-*s %s",
		colPlaceWidth, "pos:",
		colIDWidth, "id:",
		"name:")))
	b.WriteString("\n")

	// Данные
	for i, row := range m.EventModel.Rows {
		if i >= tableHeight {
			break
		}

		rowFocus := (m.Focus == 4+i)
		b.WriteString(indent)
		line := m.renderRow(i, row, colPlaceWidth, colIDWidth, colNameWidth, rowFocus)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Заполняем пустое пространство до предпоследней строки (подсказка в самом низу)
	for i := len(m.EventModel.Rows); i < tableHeight-1; i++ {
		b.WriteString("\n")
	}

	// Подсказка в самом низу, растянутая на всю ширину
	helpItems := []string{
		"↑↓ навигация",
		"^+↑↓ двигать",
		"↵ редактировать",
		"i идентифицировать",
		"* принять всех",
		"⌫ удалить",
		"s сохранить",
		"⎋ выход",
	}
	help := strings.Join(helpItems, " | ")
	// Дополняем пробелами до ширины экрана
	if len(help) < m.Width {
		help = help + strings.Repeat(" ", m.Width-len(help))
	}
	b.WriteString(styleDim.Render(ellipsize(help, m.Width)))

	return b.String()
}

func (m TUIModel) renderRow(rowIndex int, row PilotRow, placeWidth, idWidth, nameWidth int, selected bool) string {
	placeStr := row.Position.String()

	// Определяем ID для отображения и суффикс
	var idStr string
	var suffix string
	var isVirtual bool
	if row.Id == "" {
		// Получаем предложенный виртуальный ID с суффиксом
		suggestedId, suf, _ := m.getSuggestedVirtualId(rowIndex)
		idStr = suggestedId
		suffix = suf
		isVirtual = true
	} else {
		// Реальный ID из файла или выбранный из базы — белый
		idStr = string(row.Id)
	}

	nameStr := row.Name
	if nameStr == "" {
		nameStr = "[name]"
	}

	// Обрезаем строки до нужной ширины
	// Для ID с суффиксом вычитаем 1 (суффикс без пробела)
	idStr = ellipsize(idStr, idWidth-1)
	nameStr = ellipsize(nameStr, nameWidth)

	// При выделении используем plain text без стилей
	if selected {
		idWithSuffix := idStr + suffix
		line := fmt.Sprintf("%-*s %-*s %-*s", placeWidth, placeStr, idWidth, idWithSuffix, nameWidth, nameStr)
		return styleSelected.Render(line)
	}

	// Виртуальный ID с суффиксом — затемненный (без пробела перед суффиксом)
	if isVirtual {
		idStr = styleDim.Render(idStr)
		suffix = styleDim.Render(suffix)
	} else {
		suffix = ""
	}

	// Формируем nameStr с учётом placeholder'а
	if row.Name == "" {
		nameStr = styleDim.Render(nameStr)
	}

	// Для виртуальных строк весь placeStr тоже серый
	if row.Virtual {
		placeStr = styleDim.Render(placeStr)
	}

	// Собираем строку с правильным выравниванием
	// ID + суффикс без пробела между ними, всё в одной ячейке
	placeCell := lipgloss.NewStyle().Width(placeWidth).Render(placeStr)
	idWithSuffix := idStr + suffix
	idCell := lipgloss.NewStyle().Width(idWidth).Render(idWithSuffix)
	nameCell := lipgloss.NewStyle().Width(nameWidth).Render(nameStr)

	return placeCell + " " + idCell + " " + nameCell
}

func (m TUIModel) viewEditModal() string {
	// Специальная отрисовка для редактора даты
	if m.Screen == ScreenEditEventDate {
		return m.viewDateEditModal()
	}

	// Определяем ширину поля ввода
	inputWidth := 50
	switch m.Screen {
	case ScreenEditName, ScreenEditEventName, ScreenEditEventOrganizer:
		inputWidth = 36
	}

	// Создаём содержимое модалки
	var content strings.Builder

	// Поле ввода с нужной шириной
	content.WriteString(lipgloss.NewStyle().Width(inputWidth).Render(m.TextInput.View()))
	content.WriteString("\n")

	// Ошибка валидации (только для имени пилота)
	if m.Screen == ScreenEditName && m.ShowValidationError {
		content.WriteString(styleError.Render("* обязательное поле"))
		content.WriteString("\n")
	}

	content.WriteString("\n")

	// Подсказка
	content.WriteString(styleHelp.Render("↵ сохранить | ⎋ отмена"))

	// Создаём рамку со скругленными углами (без фона)
	modal := lipgloss.NewStyle().
		MaxWidth(60).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAccent)).
		Render(content.String())

	// Размещаем модалку по центру поверх фона
	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height)
}

// viewFindId отрисовывает список найденных пилотов для выбора ID
func (m TUIModel) viewFindId() string {
	var content strings.Builder

	// Список результатов
	if len(m.FindResults) == 0 {
		content.WriteString("Ничего не найдено")
	} else {
		for i, res := range m.FindResults {
			// Определяем, это последний элемент (новый виртуальный ID)
			isNewId := (i == len(m.FindResults)-1)

			// Формируем строку: ID имя рейтинг
			idStr := res.Id
			ratingStr := fmt.Sprintf("%d", res.Rating)

			// При выделении используем plain text, иначе применяем стили
			if i == m.FindCursor {
				line := fmt.Sprintf("%-15s %-30s %s", idStr, res.Name, ratingStr)
				content.WriteString(styleSelected.Render(line))
			} else {
				// Для виртуального ID применяем серый цвет (только когда не выделено)
				if isNewId {
					// Используем lipgloss.Width для корректного выравнивания с ANSI-кодами
					idCell := lipgloss.NewStyle().Width(15).Render(styleDim.Render(idStr))
					nameCell := lipgloss.NewStyle().Width(30).Render(res.Name)
					ratingCell := lipgloss.NewStyle().Width(4).Render(styleDim.Render(ratingStr))
					content.WriteString(idCell + " " + nameCell + " " + ratingCell)
				} else {
					line := fmt.Sprintf("%-15s %-30s %s", idStr, res.Name, ratingStr)
					content.WriteString(line)
				}
			}
			if i < len(m.FindResults)-1 {
				content.WriteString("\n")
			}
		}
	}

	content.WriteString("\n\n")

	// Подсказка
	content.WriteString(styleHelp.Render("↑↓ выбор | ↵ подтвердить | ⎋ отмена"))

	// Создаём рамку со скругленными углами
	modal := lipgloss.NewStyle().
		MaxWidth(70).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAccent)).
		Render(content.String())

	// Размещаем модалку по центру поверх фона
	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height)
}

// viewDateEditModal отрисовывает редактор даты
func (m TUIModel) viewDateEditModal() string {
	// Создаём содержимое модалки
	var content strings.Builder

	// Стиль для активного поля (с фоном)
	activeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(colorAccent)).
		Foreground(lipgloss.Color(colorBlack))

	// Стиль для неактивного поля
	inactiveStyle := lipgloss.NewStyle()

	// Стиль для дня недели (затененный)
	weekdayStyle := styleDim

	// Форматируем значения с лидирующими нулями
	yearStr := fmt.Sprintf("%04d", m.DateYear)
	monthStr := fmt.Sprintf("%02d", m.DateMonth)
	dayStr := fmt.Sprintf("%02d", m.DateDay)

	// Отрисовываем 3 поля: YYYY - MM - DD
	var dateLine strings.Builder

	// Год
	if m.DateFocus == 0 {
		dateLine.WriteString(activeStyle.Render(yearStr))
	} else {
		dateLine.WriteString(inactiveStyle.Render(yearStr))
	}

	dateLine.WriteString(inactiveStyle.Render("-"))

	// Месяц
	if m.DateFocus == 1 {
		dateLine.WriteString(activeStyle.Render(monthStr))
	} else {
		dateLine.WriteString(inactiveStyle.Render(monthStr))
	}

	dateLine.WriteString(inactiveStyle.Render("-"))

	// День
	if m.DateFocus == 2 {
		dateLine.WriteString(activeStyle.Render(dayStr))
	} else {
		dateLine.WriteString(inactiveStyle.Render(dayStr))
	}

	content.WriteString(dateLine.String())

	// Добавляем день недели (2-буквенное сокращение, капсом)
	weekdayNames := []string{"ВС", "ПН", "ВТ", "СР", "ЧТ", "ПТ", "СБ"}
	t := time.Date(m.DateYear, time.Month(m.DateMonth), m.DateDay, 0, 0, 0, 0, time.UTC)
	weekdayStr := weekdayNames[int(t.Weekday())]
	content.WriteString(" " + weekdayStyle.Render(weekdayStr))
	content.WriteString("\n\n")

	// Подсказка
	content.WriteString(styleHelp.Render("←→ поле | ↑↓ значение | ↵ сохранить | ⎋ отмена"))

	// Создаём рамку со скругленными углами (без фона)
	modal := lipgloss.NewStyle().
		MaxWidth(60).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAccent)).
		Render(content.String())

	// Размещаем модалку по центру поверх фона
	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height)
}

// viewSelectClass отрисовывает выпадающий список для выбора класса
func (m TUIModel) viewSelectClass() string {
	var content strings.Builder

	// Список опций
	for i, opt := range model.KnownClasses {
		if i == m.ClassCursor {
			content.WriteString(styleSelected.Render(string(opt)))
		} else {
			content.WriteString(string(opt))
		}
		if i < len(model.KnownClasses)-1 {
			content.WriteString("\n")
		}
	}

	content.WriteString("\n\n")

	// Подсказка
	content.WriteString(styleHelp.Render("↑↓ выбор | ↵ подтвердить | ⎋ отмена"))

	// Создаём рамку со скругленными углами
	modal := lipgloss.NewStyle().
		MaxWidth(60).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAccent)).
		Render(content.String())

	// Размещаем модалку по центру поверх фона
	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height)
}

// placeModal размещает модалку по центру экрана поверх фона
func placeModal(background, modal string, screenWidth, screenHeight int) string {
	bgLines := strings.Split(background, "\n")
	modalLines := strings.Split(modal, "\n")

	modalHeight := len(modalLines)

	// Находим максимальную ширину модалки
	modalWidth := 0
	for _, line := range modalLines {
		w := lipgloss.Width(line)
		if w > modalWidth {
			modalWidth = w
		}
	}

	// Вычисляем позицию для центрирования
	startY := (screenHeight - modalHeight) / 2
	if startY < 0 {
		startY = 0
	}

	// Вычисляем отступ слева для центрирования модалки
	startX := (screenWidth - modalWidth) / 2
	if startX < 0 {
		startX = 0
	}

	var result strings.Builder

	for y, bgLine := range bgLines {
		if y >= startY && y < startY+modalHeight {
			// Эта строка содержит модалку
			modalLine := modalLines[y-startY]
			modalLineWidth := lipgloss.Width(modalLine)

			// Получаем видимую длину bgLine
			bgWidth := lipgloss.Width(bgLine)

			// Левая часть фона (до модалки)
			if startX > 0 {
				if startX < bgWidth {
					// Обрезаем bgLine до startX, сохраняя ANSI-коды
					leftPart := truncateWithANSI(bgLine, startX)
					result.WriteString(styleDim.Render(leftPart))
				} else {
					// Фон короче, чем startX
					result.WriteString(styleDim.Render(bgLine))
					result.WriteString(styleDim.Render(strings.Repeat(" ", startX-bgWidth)))
				}
			}

			// Сама модалка
			result.WriteString(modalLine)

			// Правая часть фона (после модалки)
			afterX := startX + modalLineWidth
			if afterX < bgWidth {
				rightPart := truncateFromWithANSI(bgLine, afterX)
				result.WriteString(styleDim.Render(rightPart))
			}
			// Если bgLine закончился раньше afterX — ничего не добавляем
		} else {
			// Обычная строка - затемняем
			result.WriteString(styleDim.Render(bgLine))
		}

		if y < len(bgLines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// truncateWithANSI обрезает строку до видимой ширины, сохраняя ANSI-коды
func truncateWithANSI(s string, width int) string {
	if width <= 0 {
		return ""
	}

	var result strings.Builder
	w := 0
	inANSI := false

	for _, r := range s {
		if r == '\x1b' {
			inANSI = true
			result.WriteRune(r)
			continue
		}
		if inANSI {
			result.WriteRune(r)
			if r == 'm' {
				inANSI = false
			}
			continue
		}

		rw := runewidth.RuneWidth(r)
		if w+rw > width {
			break
		}
		result.WriteRune(r)
		w += rw
	}

	return result.String()
}

// truncateFromWithANSI обрезает строку начиная с позиции, сохраняя ANSI-коды
func truncateFromWithANSI(s string, from int) string {
	if from <= 0 {
		return s
	}

	var result strings.Builder
	w := 0
	inANSI := false
	started := false

	for _, r := range s {
		if r == '\x1b' {
			inANSI = true
			if started {
				result.WriteRune(r)
			}
			continue
		}
		if inANSI {
			if started {
				result.WriteRune(r)
			}
			if r == 'm' {
				inANSI = false
			}
			continue
		}

		rw := runewidth.RuneWidth(r)
		if !started && w >= from {
			started = true
		}
		w += rw

		if started {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// modalStyle возвращает базовый стиль для модальных окон
func (m TUIModel) modalStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		MaxWidth(60).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAccent))
}

// modalStyleWarning возвращает стиль для модальных окон с предупреждением (оранжевая рамка)
func (m TUIModel) modalStyleWarning() lipgloss.Style {
	return lipgloss.NewStyle().
		MaxWidth(60).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorWarning))
}

func (m TUIModel) viewConfirmExit() string {
	var content strings.Builder

	content.WriteString("Сохранить изменения перед выходом?\n\n")
	content.WriteString("y сохранить | n не сохранять | ⎋ отмена")

	// Создаём рамку с предупреждением (оранжевая)
	modal := m.modalStyleWarning().Render(content.String())

	// Размещаем модалку по центру поверх фона
	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height)
}

func (m TUIModel) viewConfirmOverwrite() string {
	var content strings.Builder

	content.WriteString("Файл уже существует:\n")
	content.WriteString(styleDim.Render(toTildePath(m.OverwritePath)) + "\n\n")
	content.WriteString("y перезаписать | n отменить")

	// Создаём рамку с предупреждением (красная)
	modal := m.modalStyleWarning().Render(content.String())

	// Размещаем модалку по центру поверх фона
	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height)
}

func (m TUIModel) viewSaveAs() string {
	var content strings.Builder

	content.WriteString(m.TextInput.View())
	content.WriteString("\n\n")
	content.WriteString("↵ сохранить | ⎋ отмена")

	// Создаём рамку с базовым стилем
	modal := m.modalStyle().Render(content.String())

	// Размещаем модалку по центру поверх фона
	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height)
}

func ellipsize(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	width := runewidth.StringWidth(s)
	if width <= maxWidth {
		return s
	}

	if maxWidth <= 3 {
		return runewidth.Truncate(s, maxWidth, "")
	}

	return runewidth.Truncate(s, maxWidth-3, "...")
}

// Run запускает TUI
func Run(filename string) error {
	eventModel, err := NewEventModel(filename)
	if err != nil {
		return err
	}

	p := tea.NewProgram(NewTUIModel(eventModel))
	_, err = p.Run()
	return err
}
