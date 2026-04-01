package prepare

import (
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/eterverda/fpvladder/internal/model"
)

func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Настраиваем viewport: высота = весь экран минус 1 строка для подсказки
		m.Viewport.SetWidth(msg.Width)
		m.Viewport.SetHeight(msg.Height - 1)
		m.Viewport.SetContent(m.renderMainContent())
		return m, nil

	case tea.KeyboardEnhancementsMsg:
		m.supportsBaseCode = msg.Flags&ansi.KittyReportAlternateKeys != 0
		return m, nil

	case tea.MouseClickMsg:
		if m.Screen == ScreenMain {
			return m.handleMouseClick(msg)
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	// Передаём сообщения в viewport только на главном экране
	if m.Screen == ScreenMain {
		var cmd tea.Cmd
		m.Viewport, cmd = m.Viewport.Update(msg)
		return m, cmd
	}

	if m.Screen == ScreenEditName || m.Screen == ScreenSaveAs ||
		m.Screen == ScreenEditEventDate || m.Screen == ScreenEditEventName ||
		m.Screen == ScreenEditEventOrganizer {
		var cmd tea.Cmd
		m.TextInput, cmd = m.TextInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m TUIModel) keyCode(msg tea.KeyPressMsg) rune {
	if m.supportsBaseCode && msg.BaseCode != 0 {
		return msg.BaseCode
	}
	return msg.Code
}

func (m TUIModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		// Если в модалке редактирования - закрываем её без применения
		if m.Screen == ScreenEditName ||
			m.Screen == ScreenEditEventDate ||
			m.Screen == ScreenEditEventName ||
			m.Screen == ScreenEditEventOrganizer ||
			m.Screen == ScreenSelectClass ||
			m.Screen == ScreenFindId {
			m.Screen = ScreenMain
			return m, nil
		}
		// Если уже в модалке подтверждения - закрываем её
		if m.Screen == ScreenConfirmExit {
			m.Screen = ScreenMain
			return m, nil
		}
		// Если есть несохранённые изменения - показываем подтверждение
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

func (m TUIModel) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if m.Screen != ScreenMain {
		return m, nil
	}

	x, y := msg.X, msg.Y

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
	// 9+: данные таблицы (по строкам пилотов)
	tableStartY := 9
	tableStartX := 4

	if x < tableStartX || y < tableStartY {
		return m, nil
	}

	// Вычисляем глобальный индекс пилота
	rowIndex := y - tableStartY
	globalIdx := 0
	linesPassed := 0

	for _, team := range m.EventModel.Rows {
		teamHeight := len(team.Pilots)
		if teamHeight == 0 {
			teamHeight = 1 // Виртуальная строка
		}

		if linesPassed+teamHeight > rowIndex {
			// Нашли нужную команду
			pilotIdx := rowIndex - linesPassed
			if pilotIdx < len(team.Pilots) {
				globalIdx += pilotIdx
				m.Focus = 4 + globalIdx
				m.Cursor = globalIdx
			}
			break
		}

		linesPassed += teamHeight
		globalIdx += len(team.Pilots)
	}

	return m, nil
}

func (m TUIModel) handleMainKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyCode := m.keyCode(msg)
	totalPilots := m.getTotalPilots()

	switch {
	case msg.String() == "up" || keyCode == 'k':
		if m.Focus > 0 {
			m.Focus--
		}
	case msg.String() == "down" || keyCode == 'j':
		if m.Focus < 4+totalPilots-1 {
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
			m.Cursor = m.Focus - 4
			m.startEditNameModal()
		}
	case keyCode == 'e':
		if m.Focus >= 4 {
			m.Cursor = m.Focus - 4
			m.startEditNameModal()
		} else if m.Focus == 1 {
			m.startEditEventField(ScreenEditEventName)
		} else if m.Focus == 2 {
			m.startEditEventField(ScreenEditEventOrganizer)
		}
	case keyCode == 'i':
		if m.Focus >= 4 {
			m.Cursor = m.Focus - 4
			if !m.isVirtualPilot(m.Cursor) {
				m.startFindId()
			}
		}
	case keyCode == '*':
		m.acceptAllIdentified()
	case msg.String() == "delete" || msg.String() == "backspace":
		if m.Focus >= 4 {
			m.Cursor = m.Focus - 4
			if !m.isVirtualPilot(m.Cursor) {
				m.EventModel.DeletePilot(m.Cursor)
				if m.Cursor >= m.getTotalPilots() {
					m.Cursor = m.getTotalPilots() - 1
					m.Focus = 4 + m.Cursor
				}
			}
		}
	case keyCode == 's':
		if m.EventModel.Filename == "" {
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
	case msg.String() == "ctrl+up":
		if m.Focus >= 4 {
			m.Cursor = m.Focus - 4
			teamIdx, _, ok := m.getTeamForPilot(m.Cursor)
			if ok {
				newTeamIdx := m.EventModel.MoveRowUp(teamIdx)
				// Пересчитываем курсор на первого пилота новой команды
				newGlobalIdx := 0
				for i := 0; i < newTeamIdx && i < len(m.EventModel.Rows); i++ {
					newGlobalIdx += len(m.EventModel.Rows[i].Pilots)
				}
				m.Cursor = newGlobalIdx
				m.Focus = 4 + newGlobalIdx
			}
		}
	case msg.String() == "ctrl+down":
		if m.Focus >= 4 {
			m.Cursor = m.Focus - 4
			teamIdx, _, ok := m.getTeamForPilot(m.Cursor)
			if ok {
				newTeamIdx := m.EventModel.MoveRowDown(teamIdx)
				// Пересчитываем курсор на первого пилота новой команды
				newGlobalIdx := 0
				for i := 0; i < newTeamIdx && i < len(m.EventModel.Rows); i++ {
					newGlobalIdx += len(m.EventModel.Rows[i].Pilots)
				}
				m.Cursor = newGlobalIdx
				m.Focus = 4 + newGlobalIdx
			}
		}
	// t - объединить все команды в ничье с текущей в одну команду
	case keyCode == 't' && msg.Mod&tea.ModShift == 0:
		if m.Focus >= 4 {
			m.Cursor = m.Focus - 4
			teamIdx, _, ok := m.getTeamForPilot(m.Cursor)
			if ok {
				newTeamIdx := m.EventModel.MergeTeam(teamIdx)
				// Пересчитываем курсор на первого пилота новой команды
				newGlobalIdx := 0
				for i := 0; i < newTeamIdx && i < len(m.EventModel.Rows); i++ {
					newGlobalIdx += len(m.EventModel.Rows[i].Pilots)
				}
				m.Cursor = newGlobalIdx
				m.Focus = 4 + newGlobalIdx
			}
		}
	// T (shift+t) - разбить команду на отдельные команды по одному пилоту (в ничье)
	case keyCode == 't' && msg.Mod&tea.ModShift != 0:
		if m.Focus >= 4 {
			m.Cursor = m.Focus - 4
			teamIdx, _, ok := m.getTeamForPilot(m.Cursor)
			if ok {
				newTeamIdx := m.EventModel.SplitTeam(teamIdx)
				// Пересчитываем курсор на первого пилота новой команды
				newGlobalIdx := 0
				for i := 0; i < newTeamIdx && i < len(m.EventModel.Rows); i++ {
					newGlobalIdx += len(m.EventModel.Rows[i].Pilots)
				}
				m.Cursor = newGlobalIdx
				m.Focus = 4 + newGlobalIdx
			}
		}
	}

	return m, nil
}

func (m *TUIModel) startEditEventField(screen Screen) {
	m.Screen = screen
	switch screen {
	case ScreenEditEventDate:
		t := time.Time(m.EventModel.Event.Date)
		m.DateYear = t.Year()
		m.DateMonth = int(t.Month())
		m.DateDay = t.Day()
		m.DateFocus = 0
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
	_, _, pilot, ok := m.getPilot(m.Cursor)
	if ok {
		m.TextInput.SetValue(pilot.Name)
	} else {
		m.TextInput.SetValue("")
	}
	m.TextInput.Focus()
	m.TextInput.CursorEnd()
}

func (m *TUIModel) startSelectClass() {
	m.Screen = ScreenSelectClass
	m.ClassCursor = 0
	currentClass := m.EventModel.Event.Class
	for i, opt := range model.KnownClasses {
		if opt == currentClass {
			m.ClassCursor = i
			break
		}
	}
}

func (m *TUIModel) acceptAllIdentified() {
	for i := range m.EventModel.Rows {
		for j := range m.EventModel.Rows[i].Pilots {
			if !m.EventModel.Rows[i].Virtual && m.EventModel.Rows[i].Pilots[j].IdKind == IdKindSuggested {
				// Принимаем suggested ID как explicit
				m.EventModel.Rows[i].Pilots[j].IdKind = IdKindExplicit
				m.EventModel.Modified = true
			}
		}
	}
	// Пересчитываем виртуальные ID для оставшихся неидентифицированных пилотов
	m.reassignVirtualIds()
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

	_, _, pilot, ok := m.getPilot(m.Cursor)
	if !ok {
		m.Screen = ScreenMain
		return
	}

	searchName := pilot.Name
	m.FindResults = nil

	// Если есть текущий explicit ID - добавляем его первым
	if pilot.IdKind == IdKindExplicit && pilot.Id != "" {
		m.FindResults = append(m.FindResults, FindResult{
			Name:   pilot.Name,
			Id:     string(pilot.Id),
			Rating: 1200,
		})
	}

	// Ищем в базе и добавляем найденные (исключая дубликат текущего explicit)
	results := m.EventModel.FindPilotsByName(searchName)
	for _, r := range results {
		// Пропускаем если совпадает с текущим explicit ID
		if pilot.IdKind == IdKindExplicit && string(pilot.Id) == r.Id {
			continue
		}
		m.FindResults = append(m.FindResults, FindResult{
			Name:   r.Name,
			Id:     r.Id,
			Rating: r.Rating,
		})
	}

	// Добавляем виртуальный ID последним
	virtualId := m.getVirtualId(m.Cursor)
	m.FindResults = append(m.FindResults, FindResult{
		Name:   searchName,
		Id:     virtualId,
		Rating: 1200,
	})
}

func (m TUIModel) handleFindIdKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Screen = ScreenMain
	case "enter":
		if len(m.FindResults) > 0 && m.FindCursor < len(m.FindResults) {
			res := m.FindResults[m.FindCursor]
			teamIdx, pilotIdx, _, ok := m.getPilot(m.Cursor)
			if !ok {
				m.Screen = ScreenMain
				return m, nil
			}

			// Последний элемент - всегда виртуальный ID
			isVirtual := (m.FindCursor == len(m.FindResults)-1)
			m.EventModel.Rows[teamIdx].Pilots[pilotIdx].Id = model.Id(res.Id)
			if isVirtual {
				m.EventModel.Rows[teamIdx].Pilots[pilotIdx].IdKind = IdKindVirtual
			} else {
				m.EventModel.Rows[teamIdx].Pilots[pilotIdx].IdKind = IdKindExplicit
			}
			// Пересчитываем виртуальные ID для оставшихся неидентифицированных пилотов
			m.reassignVirtualIds()

			if m.EventModel.Rows[teamIdx].Virtual {
				m.EventModel.PromoteVirtualRow(teamIdx)
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
	if m.Screen == ScreenEditEventDate {
		return m.handleDateEditKey(msg)
	}

	switch msg.String() {
	case "esc":
		m.ShowValidationError = false
		m.Screen = ScreenMain
	case "enter":
		value := m.TextInput.Value()

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
			if strings.TrimSpace(value) == "" {
				m.ShowValidationError = true
				return m, nil
			}
			m.ShowValidationError = false
			teamIdx, pilotIdx, _, ok := m.getPilot(m.EditRow)
			if ok {
				m.EventModel.Rows[teamIdx].Pilots[pilotIdx].Name = value
				// Если был Explicit ID - оставляем как есть
				// Иначе ищем suggested ID в базе
				if m.EventModel.Rows[teamIdx].Pilots[pilotIdx].IdKind != IdKindExplicit {
					results := m.EventModel.FindPilotsByName(value)
					if len(results) == 1 {
						m.EventModel.Rows[teamIdx].Pilots[pilotIdx].Id = model.Id(results[0].Id)
						m.EventModel.Rows[teamIdx].Pilots[pilotIdx].IdKind = IdKindSuggested
					} else if len(results) > 1 {
						m.EventModel.Rows[teamIdx].Pilots[pilotIdx].Id = model.Id(results[0].Id)
						m.EventModel.Rows[teamIdx].Pilots[pilotIdx].IdKind = IdKindSuggested
					} else {
						m.EventModel.Rows[teamIdx].Pilots[pilotIdx].Id = ""
						m.EventModel.Rows[teamIdx].Pilots[pilotIdx].IdKind = IdKindVirtual
					}
				}
				if m.EventModel.Rows[teamIdx].Virtual {
					m.EventModel.PromoteVirtualRow(teamIdx)
				} else {
					m.EventModel.Modified = true
				}
			}
			m.Screen = ScreenMain
		}
	}

	if m.ShowValidationError {
		m.ShowValidationError = false
	}

	var cmd tea.Cmd
	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

func (m TUIModel) handleDateEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Screen = ScreenMain
		return m, nil
	case "enter":
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
		case 0:
			t = t.AddDate(1, 0, 0)
		case 1:
			t = addMonthOverflow(t, 1)
		case 2:
			t = t.AddDate(0, 0, 1)
		}
		m.DateYear = t.Year()
		m.DateMonth = int(t.Month())
		m.DateDay = t.Day()
		return m, nil
	case "down":
		t := time.Date(m.DateYear, time.Month(m.DateMonth), m.DateDay, 0, 0, 0, 0, time.UTC)
		switch m.DateFocus {
		case 0:
			t = t.AddDate(-1, 0, 0)
		case 1:
			t = addMonthOverflow(t, -1)
		case 2:
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
	keyCode := m.keyCode(msg)

	switch {
	case msg.Code == tea.KeyEscape || msg.String() == "esc" || msg.String() == "":
		m.Screen = ScreenMain
		return m, nil
	case keyCode == 'n':
		return m, tea.Quit
	case keyCode == 'y':
		if m.EventModel.Filename == "" {
			m.Screen = ScreenSaveAs
			m.SaveAsMode = SaveAsModeExit
			cwd, _ := os.Getwd()
			path := cwd + "/" + m.EventModel.GenerateFilename()
			m.TextInput.SetValue(toTildePath(path))
			m.TextInput.Focus()
			m.TextInput.CursorEnd()
			return m, nil
		}
		m.EventModel.Save()
		return m, tea.Quit
	}

	return m, nil
}

func (m TUIModel) handleSaveAsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Screen = ScreenMain
	case "enter":
		absPath := fromTildePath(m.TextInput.Value())

		if m.EventModel.IsNew {
			if _, err := os.Stat(absPath); err == nil {
				m.OverwritePath = absPath
				m.Screen = ScreenConfirmOverwrite
				return m, nil
			}
		}

		m.EventModel.Filename = absPath
		if err := m.EventModel.Save(); err == nil {
			if m.SaveAsMode == SaveAsModeExit {
				return m, tea.Quit
			}
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
		m.Screen = ScreenSaveAs
		m.OverwritePath = ""
		return m, nil
	case keyCode == 'y':
		m.EventModel.Filename = m.OverwritePath
		m.OverwritePath = ""
		if err := m.EventModel.Save(); err == nil {
			if m.SaveAsMode == SaveAsModeExit {
				return m, tea.Quit
			}
			m.Screen = ScreenMain
		}
	}

	return m, nil
}

// View отрисовывает интерфейс
