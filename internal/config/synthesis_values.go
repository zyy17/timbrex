package config

import (
	"fmt"
	"math"
)

// These readers accept YAML scalar types without coercing strings, booleans, or
// floating-point values into integers. Diagnostics never include input values.
func synthesisFields(raw any) (map[string]any, error) {
	fields, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected a non-null mapping")
	}
	return fields, nil
}

func synthesisList(raw any) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a non-null list")
	}
	return items, nil
}

func synthesisString[T ~string](raw any, field string) (*T, error) {
	value, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("%s must be a string", field)
	}
	result := T(value)
	return &result, nil
}

func synthesisBool(raw any, field string) (*bool, error) {
	value, ok := raw.(bool)
	if !ok {
		return nil, fmt.Errorf("%s must be a boolean", field)
	}
	return &value, nil
}

func synthesisInteger(raw any, field string) (*int64, error) {
	var value int64
	switch n := raw.(type) {
	case int:
		value = int64(n)
	case int64:
		value = n
	case uint64:
		if n > math.MaxInt64 {
			return nil, fmt.Errorf("%s must be a supported integer", field)
		}
		value = int64(n)
	default:
		return nil, fmt.Errorf("%s must be an integer", field)
	}
	return &value, nil
}

func synthesisNumber(raw any, field string) (*float64, error) {
	var value float64
	switch n := raw.(type) {
	case int:
		value = float64(n)
	case int64:
		value = float64(n)
	case uint64:
		value = float64(n)
	case float64:
		value = n
	default:
		return nil, fmt.Errorf("%s must be a finite number", field)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("%s must be a finite number", field)
	}
	return &value, nil
}
