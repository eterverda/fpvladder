package model

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Position представляет позицию пилота с учётом ничьих
// Value — собственно позиция (1, 2, 3...)
// TieCount — количество ничьих (0 если нет ничьей)
// При TieCount > 0 позиция отображается как "Value-(Value+TieCount)"
// Например: Position{11, 1} → "11-12" (ничья 11-12 места)
type Position struct {
	Int      int // собственно позиция (1, 2, 3...)
	TieCount int // количество ничьих (0 если нет ничьей)
}

// String возвращает строковое представление позиции
// Без ничьей: "11"
// С ничьей: "11-12"
func (p Position) String() string {
	if p.TieCount == 0 {
		return strconv.Itoa(p.Int)
	}
	return fmt.Sprintf("%d-%d", p.Int, p.Int+p.TieCount)
}

// parsePosition парсит позицию из строки
// Поддерживает форматы: "11", "11-12", "11 - 12"
func parsePosition(s string) (Position, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Position{}, fmt.Errorf("empty position")
	}

	// Проверяем формат с ничьей: "11-12" или "11 - 12"
	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		if len(parts) != 2 {
			return Position{}, fmt.Errorf("invalid tie format: %s", s)
		}
		start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil || start >= end {
			return Position{}, fmt.Errorf("invalid tie format: %s", s)
		}
		return Position{Int: start, TieCount: end - start}, nil
	}

	// Простое число
	val, err := strconv.Atoi(s)
	if err != nil {
		return Position{}, fmt.Errorf("invalid position: %s", s)
	}
	return Position{Int: val, TieCount: 0}, nil
}

// Equal проверяет равенство двух позиций
func (p Position) Equal(other Position) bool {
	return p.Int == other.Int && p.TieCount == other.TieCount
}

// MarshalYAML сериализует Position как строку без кавычек
func (p Position) MarshalYAML() (any, error) {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: p.String(),
		Style: 0, // Plain style - без кавычек
	}, nil
}

// UnmarshalYAML десериализует Position из строки или числа
func (p *Position) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		// Пробуем декодировать как число
		var val int
		if err2 := node.Decode(&val); err2 != nil {
			return err
		}
		*p = Position{Int: val, TieCount: 0}
		return nil
	}

	pos, err := parsePosition(s)
	if err != nil {
		return err
	}
	*p = pos
	return nil
}

// RelativePosition представляет относительную позицию (Position/Count)
// Используется в RatingSummary для хранения относительной позиции
// Position — позиция в событии (с учётом ничьих)
// Count — общее количество участников
type RelativePosition struct {
	Position // позиция в событии (с учётом ничьих)
	Count    int
}

func (rp RelativePosition) String() string {
	return fmt.Sprintf("%s/%d", rp.Position.String(), rp.Count)
}

// MarshalYAML превращает структуру в строку "pos/count" без кавычек
func (rp RelativePosition) MarshalYAML() (interface{}, error) {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: rp.String(),
		Style: 0, // Plain style - без кавычек
	}, nil
}

// UnmarshalYAML парсит строку "pos/count" обратно в структуру
func (rp *RelativePosition) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid relative position format: %s", s)
	}

	pos, err := parsePosition(strings.TrimSpace(parts[0]))
	if err != nil {
		return err
	}

	count, err := fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &rp.Count)
	if err != nil || count != 1 {
		return fmt.Errorf("invalid count in relative position: %s", s)
	}

	rp.Position = pos
	return nil
}
