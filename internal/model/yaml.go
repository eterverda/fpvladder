package model

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

// MarshalPrettyYaml сериализует значение в YAML с отступом в 2 пробела
func MarshalPrettyYaml(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
