package qwen

import (
	"fmt"

	"github.com/zyy17/timbrex/internal/provideropts"
	"github.com/zyy17/timbrex/internal/providers"
)

func synthesisOptions(value provideropts.Value) (provideropts.Qwen, error) {
	switch v := value.(type) {
	case nil:
		return provideropts.Qwen{}, nil
	case provideropts.Qwen:
		return v, v.Validate()
	default:
		return provideropts.Qwen{}, fmt.Errorf("qwen received an unsupported synthesis options type")
	}
}

func buildWithSynthesis(seg providers.Segment, model string, value provideropts.Qwen) (request, error) {
	if err := value.Validate(); err != nil {
		return request{}, err
	}
	r, err := build(seg, model)
	if err != nil {
		return r, err
	}
	if value.Seed != nil {
		seed := uint16(*value.Seed)
		r.Input.Seed = &seed
	}
	if fix := value.HotFix; fix != nil && (fix.Pronunciation != nil || fix.Replace != nil) {
		// The API nests both arrays under hot_fix, not input.replace:
		// https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api
		r.Input.HotFix = &hotFix{Pronunciation: textPairs(fix.Pronunciation), Replace: textPairs(fix.Replace)}
	}
	return r, nil
}

func textPairs(pairs *[]provideropts.QwenTextPair) *[]map[string]string {
	if pairs == nil {
		return nil
	}
	// Each prepared request owns its array and maps before concurrent rendering.
	// Preserve duplicate keys in separate objects without assigning precedence.
	result := make([]map[string]string, len(*pairs))
	for i, pair := range *pairs {
		result[i] = map[string]string{pair.Text: pair.Replacement}
	}
	return &result
}
