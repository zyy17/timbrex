package qwen

import (
	"fmt"

	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

// synthesisSnapshot is detached from caller storage and read-only after creation.
// One invocation shares it across prepared requests instead of retaining one
// dictionary copy per segment. Separate invocations never share a snapshot.
type synthesisSnapshot struct {
	seed   *uint16
	hotFix *hotFix
}

func synthesisOptions(value options.Value) (synthesisSnapshot, error) {
	switch v := value.(type) {
	case nil:
		return synthesisSnapshot{}, nil
	case options.Qwen:
		if err := v.Validate(); err != nil {
			return synthesisSnapshot{}, err
		}
		var snapshot synthesisSnapshot
		if v.Seed != nil {
			seed := uint16(*v.Seed)
			snapshot.seed = &seed
		}
		if fix := v.HotFix; fix != nil && (fix.Pronunciation != nil || fix.Replace != nil) {
			// The API nests both arrays under hot_fix, not input.replace:
			// https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api
			snapshot.hotFix = &hotFix{Pronunciation: textPairs(fix.Pronunciation), Replace: textPairs(fix.Replace)}
		}
		return snapshot, nil
	default:
		return synthesisSnapshot{}, fmt.Errorf("qwen received an unsupported synthesis options type")
	}
}

func buildWithSynthesis(seg providers.Segment, model string, value options.Qwen) (request, error) {
	snapshot, err := synthesisOptions(value)
	if err != nil {
		return request{}, err
	}
	return snapshot.build(seg, model)
}

func (snapshot synthesisSnapshot) build(seg providers.Segment, model string) (request, error) {
	r, err := build(seg, model)
	if err != nil {
		return r, err
	}
	r.Input.Seed, r.Input.HotFix = snapshot.seed, snapshot.hotFix
	return r, nil
}

func textPairs(pairs *[]options.QwenTextPair) *[]map[string]string {
	if pairs == nil {
		return nil
	}
	// The invocation owns its array and maps before concurrent rendering.
	// Preserve duplicate keys in separate objects without assigning precedence.
	result := make([]map[string]string, len(*pairs))
	for i, pair := range *pairs {
		result[i] = map[string]string{pair.Text: pair.Replacement}
	}
	return &result
}
