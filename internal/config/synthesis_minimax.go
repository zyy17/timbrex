package config

import (
	"fmt"
	"maps"
	"slices"

	"github.com/zyy17/timbrex/internal/provideropts"
)

func decodeMiniMax(raw any) (provideropts.MiniMax, error) {
	var value provideropts.MiniMax
	fields, err := synthesisFields(raw)
	if err != nil {
		return value, err
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if key != "text_normalization_mode" {
			return value, fmt.Errorf("unknown synthesis option")
		}
		value.TextNormalizationMode, err = synthesisString[provideropts.MiniMaxTextNormalization](fields[key], key)
		if err != nil {
			return value, err
		}
	}
	return value, value.Validate()
}
