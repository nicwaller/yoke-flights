package main

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

func ptr[T any](v T) *T { return &v }

func yamlToJSON(data []byte) (json.RawMessage, error) {
	var obj any
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}
	obj = convertMapKeys(obj)
	b, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("json marshal: %w", err)
	}
	return b, nil
}

// yaml.v3 unmarshals maps as map[string]any but nested maps may be map[any]any.
func convertMapKeys(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v2 := range val {
			out[k] = convertMapKeys(v2)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(val))
		for k, v2 := range val {
			out[fmt.Sprint(k)] = convertMapKeys(v2)
		}
		return out
	case []any:
		for i, item := range val {
			val[i] = convertMapKeys(item)
		}
		return val
	default:
		return v
	}
}
