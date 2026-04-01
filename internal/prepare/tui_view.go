package prepare

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/eterverda/fpvladder/internal/model"
	"github.com/mattn/go-runewidth"
)

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
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// renderMainContent рендерит весь контент кроме подсказки (для viewport)
func (m TUIModel) renderMainContent() string {
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

	// Организатор
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

	// Таблица - отрисовываем всех пилотов (без ограничения по высоте)
	indent := "    "
	colPlaceWidth := 6
	colIDWidth := 15
	colNameWidth := max(m.Width-colIDWidth-colPlaceWidth-4-4, 20)

	b.WriteString(indent)
	b.WriteString(labelStyle.Render(fmt.Sprintf("%-*s %-*s %s",
		colPlaceWidth, "pos:",
		colIDWidth, "id:",
		"name:")))
	b.WriteString("\n")

	// Данные - отрисовываем ВСЕХ пилотов (viewport будет скроллить)
	globalIdx := 0
	for _, team := range m.EventModel.Rows {
		placeStr := team.Position.String()
		isVirtualTeam := team.Virtual
		teamSize := len(team.Pilots)

		for pilotIdx, pilot := range team.Pilots {
			rowFocus := (m.Focus == 4+globalIdx)
			b.WriteString(indent)

			// Для первого пилота в команде показываем позицию, для остальных - скобка
			var pilotPlaceStr string
			if pilotIdx == 0 {
				pilotPlaceStr = placeStr
			} else if pilotIdx == teamSize-1 {
				pilotPlaceStr = "└"
			} else {
				pilotPlaceStr = "│"
			}

			line := m.renderPilotRow(globalIdx, pilot, pilotPlaceStr, isVirtualTeam, colPlaceWidth, colIDWidth, colNameWidth, rowFocus)
			b.WriteString(line)
			b.WriteString("\n")

			globalIdx++
		}
	}

	return b.String()
}

// viewMain отображает viewport с контентом и подсказку внизу
func (m *TUIModel) viewMain() string {
	// Синхронизируем viewport (контент и скролл)
	m.syncViewport()

	// Подсказка внизу
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
	if len(help) < m.Width {
		help = help + strings.Repeat(" ", m.Width-len(help))
	}
	helpLine := styleDim.Render(ellipsize(help, m.Width))

	// Собираем: viewport + подсказка
	return m.Viewport.View() + "\n" + helpLine
}

func (m TUIModel) renderPilotRow(globalIdx int, pilot PilotRow, placeStr string, isVirtualTeam bool, placeWidth, idWidth, nameWidth int, selected bool) string {
	var idStr string
	var suffix string
	var isDim bool // Признак что ID нужно отрисовать бледным

	// Определяем ID и суффикс по IdKind
	switch pilot.IdKind {
	case IdKindExplicit:
		idStr = string(pilot.Id)
	case IdKindSuggested:
		idStr = string(pilot.Id)
		suffix = "*"
		isDim = true
	case IdKindVirtual:
		idStr = m.getVirtualId(globalIdx)
		isDim = true
	}

	nameStr := pilot.Name
	if nameStr == "" {
		nameStr = "[name]"
	}

	idStr = ellipsize(idStr, idWidth-1)
	nameStr = ellipsize(nameStr, nameWidth)

	if selected {
		idWithSuffix := idStr + suffix
		line := fmt.Sprintf("%-*s %-*s %-*s", placeWidth, placeStr, idWidth, idWithSuffix, nameWidth, nameStr)
		return styleSelected.Render(line)
	}

	if isDim {
		idStr = styleDim.Render(idStr)
		suffix = styleDim.Render(suffix)
	}

	if pilot.Name == "" {
		nameStr = styleDim.Render(nameStr)
	}

	placeCell := lipgloss.NewStyle().Width(placeWidth).Render(placeStr)
	if isVirtualTeam {
		placeCell = lipgloss.NewStyle().Width(placeWidth).Render(styleDim.Render(placeStr))
	}

	idWithSuffix := idStr + suffix
	idCell := lipgloss.NewStyle().Width(idWidth).Render(idWithSuffix)
	nameCell := lipgloss.NewStyle().Width(nameWidth).Render(nameStr)

	return placeCell + " " + idCell + " " + nameCell
}

func (m TUIModel) viewEditModal() string {
	if m.Screen == ScreenEditEventDate {
		return m.viewDateEditModal()
	}

	inputWidth := 50
	switch m.Screen {
	case ScreenEditName, ScreenEditEventName, ScreenEditEventOrganizer:
		inputWidth = 36
	}

	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Width(inputWidth).Render(m.TextInput.View()))
	content.WriteString("\n")

	if m.Screen == ScreenEditName && m.ShowValidationError {
		content.WriteString(styleError.Render("* обязательное поле"))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(styleDim.Render("↵ сохранить | ⎋ отмена"))

	modal := lipgloss.NewStyle().
		MaxWidth(60).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAccent)).
		Render(content.String())

	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height, m.getFocusLine())
}

func (m TUIModel) viewFindId() string {
	var content strings.Builder

	if len(m.FindResults) == 0 {
		content.WriteString("Ничего не найдено")
	} else {
		for i, res := range m.FindResults {
			isNewId := (i == len(m.FindResults)-1)
			idStr := res.Id
			ratingStr := fmt.Sprintf("%d", res.Rating)

			if i == m.FindCursor {
				line := fmt.Sprintf("%-15s %-30s %s", idStr, res.Name, ratingStr)
				content.WriteString(styleSelected.Render(line))
			} else {
				if isNewId {
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
	content.WriteString(styleDim.Render("↑↓ выбор | ↵ подтвердить | ⎋ отмена"))

	modal := lipgloss.NewStyle().
		MaxWidth(70).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAccent)).
		Render(content.String())

	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height, m.getFocusLine())
}

func (m TUIModel) viewDateEditModal() string {
	var content strings.Builder

	activeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(colorAccent)).
		Foreground(lipgloss.Color(colorBlack))

	inactiveStyle := lipgloss.NewStyle()
	weekdayStyle := styleDim

	yearStr := fmt.Sprintf("%04d", m.DateYear)
	monthStr := fmt.Sprintf("%02d", m.DateMonth)
	dayStr := fmt.Sprintf("%02d", m.DateDay)

	var dateLine strings.Builder

	if m.DateFocus == 0 {
		dateLine.WriteString(activeStyle.Render(yearStr))
	} else {
		dateLine.WriteString(inactiveStyle.Render(yearStr))
	}

	dateLine.WriteString(inactiveStyle.Render("-"))

	if m.DateFocus == 1 {
		dateLine.WriteString(activeStyle.Render(monthStr))
	} else {
		dateLine.WriteString(inactiveStyle.Render(monthStr))
	}

	dateLine.WriteString(inactiveStyle.Render("-"))

	if m.DateFocus == 2 {
		dateLine.WriteString(activeStyle.Render(dayStr))
	} else {
		dateLine.WriteString(inactiveStyle.Render(dayStr))
	}

	content.WriteString(dateLine.String())

	weekdayNames := []string{"ВС", "ПН", "ВТ", "СР", "ЧТ", "ПТ", "СБ"}
	t := time.Date(m.DateYear, time.Month(m.DateMonth), m.DateDay, 0, 0, 0, 0, time.UTC)
	weekdayStr := weekdayNames[int(t.Weekday())]
	content.WriteString(" " + weekdayStyle.Render(weekdayStr))
	content.WriteString("\n\n")

	content.WriteString(styleDim.Render("←→ поле | ↑↓ значение | ↵ сохранить | ⎋ отмена"))

	modal := lipgloss.NewStyle().
		MaxWidth(60).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAccent)).
		Render(content.String())

	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height, m.getFocusLine())
}

func (m TUIModel) viewSelectClass() string {
	var content strings.Builder

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
	content.WriteString(styleDim.Render("↑↓ выбор | ↵ подтвердить | ⎋ отмена"))

	modal := lipgloss.NewStyle().
		MaxWidth(60).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAccent)).
		Render(content.String())

	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height, m.getFocusLine())
}

func placeModal(background, modal string, screenWidth, screenHeight, focusLine int) string {
	bgLines := strings.Split(background, "\n")
	modalLines := strings.Split(modal, "\n")

	modalHeight := len(modalLines)
	modalWidth := 0
	for _, line := range modalLines {
		w := lipgloss.Width(line)
		if w > modalWidth {
			modalWidth = w
		}
	}

	// Вычисляем Y позицию модалки чтобы избежать пересечения с focusLine
	centerY := screenHeight / 2
	var startY int
	if focusLine < centerY {
		// Выделение в верхней половине - модалка под ним
		startY = focusLine + 1
		// Если не влезает вниз, сдвигаем вверх
		if startY+modalHeight > screenHeight {
			startY = screenHeight - modalHeight
		}
	} else {
		// Выделение в нижней половине - модалка над ним
		startY = focusLine - modalHeight
		// Если не влезает вверх, сдвигаем вниз
		if startY < 0 {
			startY = 0
		}
	}
	if startY < 0 {
		startY = 0
	}

	startX := (screenWidth - modalWidth) / 2
	if startX < 0 {
		startX = 0
	}

	var result strings.Builder

	for y, bgLine := range bgLines {
		if y >= startY && y < startY+modalHeight {
			modalLine := modalLines[y-startY]
			modalLineWidth := lipgloss.Width(modalLine)
			bgWidth := lipgloss.Width(bgLine)

			if startX > 0 {
				if startX < bgWidth {
					leftPart := truncateWithANSI(bgLine, startX)
					result.WriteString(leftPart)
				} else {
					result.WriteString(bgLine)
					result.WriteString(strings.Repeat(" ", startX-bgWidth))
				}
			}

			result.WriteString(modalLine)

			afterX := startX + modalLineWidth
			if afterX < bgWidth {
				rightPart := truncateFromWithANSI(bgLine, afterX)
				result.WriteString(rightPart)
			}
		} else {
			result.WriteString(bgLine)
		}

		if y < len(bgLines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

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

func (m TUIModel) modalStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		MaxWidth(60).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAccent))
}

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
	content.WriteString(styleDim.Render("y сохранить | n не сохранять | ⎋ отмена"))

	modal := m.modalStyleWarning().Render(content.String())
	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height, m.getFocusLine())
}

func (m TUIModel) viewConfirmOverwrite() string {
	var content strings.Builder

	content.WriteString("Файл уже существует:\n")
	content.WriteString(styleDim.Render(toTildePath(m.OverwritePath)) + "\n\n")
	content.WriteString(styleDim.Render("y перезаписать | n отменить"))

	modal := m.modalStyleWarning().Render(content.String())
	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height, m.getFocusLine())
}

func (m TUIModel) viewSaveAs() string {
	var content strings.Builder

	content.WriteString(m.TextInput.View())
	content.WriteString("\n\n")
	content.WriteString(styleDim.Render("↵ сохранить | ⎋ отмена"))

	modal := m.modalStyle().Render(content.String())
	bg := m.viewMain()
	return placeModal(bg, modal, m.Width, m.Height, m.getFocusLine())
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
