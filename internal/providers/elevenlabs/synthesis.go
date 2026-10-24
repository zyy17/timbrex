package elevenlabs

import (
	"fmt"

	"github.com/zyy17/timbrex/internal/provideropts"
	"github.com/zyy17/timbrex/internal/providers"
)

func synthesisOptions(value provideropts.Value) (provideropts.ElevenLabs, error) {
	switch v := value.(type) {
	case nil:
		return provideropts.ElevenLabs{}, nil
	case provideropts.ElevenLabs:
		return v, v.Validate()
	default:
		return provideropts.ElevenLabs{}, fmt.Errorf("elevenlabs received an unsupported synthesis options type")
	}
}

func buildWithSynthesis(seg providers.Segment, model string, value provideropts.ElevenLabs) (request, error) {
	if err := value.Validate(); err != nil {
		return request{}, err
	}
	r, err := build(seg, model)
	if err != nil {
		return r, err
	}
	// Each prepared request owns its scalars before concurrent rendering begins.
	if value.Seed != nil {
		seed := uint32(*value.Seed)
		r.Seed = &seed
	}
	if value.ApplyTextNormalization != nil {
		normalization := *value.ApplyTextNormalization
		r.ApplyTextNormalization = &normalization
	}
	if value.ApplyLanguageTextNormalization != nil {
		normalization := *value.ApplyLanguageTextNormalization
		r.ApplyLanguageTextNormalization = &normalization
	}
	return r, nil
}
