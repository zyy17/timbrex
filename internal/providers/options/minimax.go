package options

import (
	"fmt"
	"maps"
	"slices"
)

// MiniMax overrides the corresponding fields on each synchronous T2A request.
type MiniMax struct {
	TextNormalizationMode *MiniMaxTextNormalization `yaml:"text_normalization_mode,omitempty"`
}

type MiniMaxTextNormalization string

const (
	MiniMaxNormalizationBasic   MiniMaxTextNormalization = "basic"
	MiniMaxNormalizationQuality MiniMaxTextNormalization = "quality"
)

func (MiniMax) providerOptions() {}

// Validate checks scalar constraints; the adapter also checks model eligibility.
func (v MiniMax) Validate() error {
	// https://platform.minimax.io/docs/api-reference/speech-t2a-http.md
	if v.TextNormalizationMode != nil {
		switch *v.TextNormalizationMode {
		case MiniMaxNormalizationBasic, MiniMaxNormalizationQuality:
		default:
			return fmt.Errorf("text_normalization_mode must be basic or quality")
		}
	}
	return nil
}

func decodeMiniMax(raw any) (MiniMax, error) {
	var value MiniMax
	fields, err := synthesisFields(raw)
	if err != nil {
		return value, err
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if key != "text_normalization_mode" {
			return value, fmt.Errorf("unknown synthesis option")
		}
		value.TextNormalizationMode, err = synthesisString[MiniMaxTextNormalization](fields[key], key)
		if err != nil {
			return value, err
		}
	}
	return value, value.Validate()
}
