package config

import (
	"fmt"
	"maps"
	"slices"

	"github.com/zyy17/timbrex/internal/provideropts"
)

func decodeQwen(raw any) (provideropts.Qwen, error) {
	var value provideropts.Qwen
	fields, err := synthesisFields(raw)
	if err != nil {
		return value, err
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		switch key {
		case "seed":
			value.Seed, err = synthesisInteger(fields[key], key)
		case "hot_fix":
			value.HotFix, err = decodeQwenHotFix(fields[key])
		default:
			return value, fmt.Errorf("unknown synthesis option")
		}
		if err != nil {
			return value, err
		}
	}
	return value, value.Validate()
}

func decodeQwenHotFix(raw any) (*provideropts.QwenHotFix, error) {
	fields, err := synthesisFields(raw)
	if err != nil {
		return nil, fmt.Errorf("hot_fix: %w", err)
	}
	value := &provideropts.QwenHotFix{}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		switch key {
		case "pronunciation":
			value.Pronunciation, err = decodeQwenTextPairs(fields[key])
		case "replace":
			value.Replace, err = decodeQwenTextPairs(fields[key])
		default:
			return nil, fmt.Errorf("unknown hot_fix option")
		}
		if err != nil {
			return nil, fmt.Errorf("hot_fix.%s: %w", key, err)
		}
	}
	return value, nil
}

func decodeQwenTextPairs(raw any) (*[]provideropts.QwenTextPair, error) {
	items, err := synthesisList(raw)
	if err != nil {
		return nil, err
	}
	result := make([]provideropts.QwenTextPair, 0, len(items))
	for _, item := range items {
		fields, err := synthesisFields(item)
		if err != nil {
			return nil, err
		}
		for _, key := range slices.Sorted(maps.Keys(fields)) {
			if key != "text" && key != "replacement" {
				return nil, fmt.Errorf("unknown text pair option")
			}
		}
		text, err := synthesisString[string](fields["text"], "text")
		if err != nil {
			return nil, err
		}
		replacement, err := synthesisString[string](fields["replacement"], "replacement")
		if err != nil {
			return nil, err
		}
		result = append(result, provideropts.QwenTextPair{Text: *text, Replacement: *replacement})
	}
	return &result, nil
}
