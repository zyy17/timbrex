package minimax

import (
	"fmt"

	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

func synthesisOptions(value options.Value, model string) (options.MiniMax, error) {
	switch v := value.(type) {
	case nil:
		return options.MiniMax{}, nil
	case options.MiniMax:
		return v, validateSynthesis(v, model)
	default:
		return options.MiniMax{}, fmt.Errorf("minimax received an unsupported synthesis options type")
	}
}

func validateSynthesis(value options.MiniMax, model string) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.TextNormalizationMode != nil && *value.TextNormalizationMode == options.MiniMaxNormalizationQuality {
		// This adapter sends stream=false. Other models silently fall back to
		// basic, so reject that downgrade locally instead of claiming quality:
		// https://platform.minimax.io/docs/api-reference/speech-t2a-http.md
		switch model {
		case "speech-2.8-hd", "speech-2.8-turbo", "speech-2.6-hd", "speech-2.6-turbo":
		default:
			return fmt.Errorf("text_normalization_mode quality requires speech-2.8-hd, speech-2.8-turbo, speech-2.6-hd, or speech-2.6-turbo")
		}
	}
	return nil
}

// buildWithSynthesis applies settings already checked by synthesisOptions.
func buildWithSynthesis(seg providers.Segment, model string, value options.MiniMax) (request, error) {
	r, err := build(seg, model)
	if err != nil {
		return r, err
	}
	// Omission keeps the existing disabled baseline. Do not send the deprecated
	// voice_setting.text_normalization or invent an explicit default.
	if value.TextNormalizationMode != nil {
		mode := *value.TextNormalizationMode
		r.TextNormalizationMode = &mode
	}
	return r, nil
}
