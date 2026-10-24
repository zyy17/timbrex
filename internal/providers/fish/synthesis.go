package fish

import (
	"fmt"

	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

func synthesisOptions(value options.Value) (options.Fish, error) {
	switch v := value.(type) {
	case nil:
		return options.Fish{}, nil
	case options.Fish:
		return v, v.Validate()
	default:
		return options.Fish{}, fmt.Errorf("fish received an unsupported synthesis options type")
	}
}

// buildWithSynthesis applies settings already checked by synthesisOptions.
func buildWithSynthesis(seg providers.Segment, model string, value options.Fish) (request, error) {
	r, err := build(seg, model)
	if err != nil {
		return r, err
	}
	// Every prepared request owns its values before rendering, including nested
	// prosody. Loudness normalization applies to all three allowed S2 models.
	// https://github.com/fishaudio/docs/blob/main/api-reference/openapi.json
	if value.Normalize != nil {
		r.Normalize = *value.Normalize
	}
	if value.Prosody != nil && value.Prosody.NormalizeLoudness != nil {
		r.Prosody.NormalizeLoudness = *value.Prosody.NormalizeLoudness
	}
	r.Language = snapshot(value.NormalizationLanguage)
	r.Temperature = snapshot(value.Temperature)
	r.TopP = snapshot(value.TopP)
	r.ChunkLength = snapshot(value.ChunkLength)
	r.MinChunkLength = snapshot(value.MinChunkLength)
	r.Latency = snapshot(value.Latency)
	r.ConditionOnPreviousChunks = snapshot(value.ConditionOnPreviousChunks)
	r.EarlyStopThreshold = snapshot(value.EarlyStopThreshold)
	return r, nil
}

func snapshot[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
