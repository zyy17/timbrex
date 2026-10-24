package config

import (
	"fmt"
	"maps"
	"slices"

	"github.com/zyy17/timbrex/internal/provideropts"
)

func decodeFish(raw any) (provideropts.Fish, error) {
	var value provideropts.Fish
	fields, err := synthesisFields(raw)
	if err != nil {
		return value, err
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		switch key {
		case "normalize":
			value.Normalize, err = synthesisBool(fields[key], key)
		case "normalization_language":
			value.NormalizationLanguage, err = synthesisString[string](fields[key], key)
		case "prosody":
			value.Prosody, err = decodeFishProsody(fields[key])
		case "temperature":
			value.Temperature, err = synthesisNumber(fields[key], key)
		case "top_p":
			value.TopP, err = synthesisNumber(fields[key], key)
		case "chunk_length":
			value.ChunkLength, err = synthesisInteger(fields[key], key)
		case "min_chunk_length":
			value.MinChunkLength, err = synthesisInteger(fields[key], key)
		case "latency":
			value.Latency, err = synthesisString[provideropts.FishLatency](fields[key], key)
		case "condition_on_previous_chunks":
			value.ConditionOnPreviousChunks, err = synthesisBool(fields[key], key)
		case "early_stop_threshold":
			value.EarlyStopThreshold, err = synthesisNumber(fields[key], key)
		default:
			return value, fmt.Errorf("unknown synthesis option")
		}
		if err != nil {
			return value, err
		}
	}
	return value, value.Validate()
}

func decodeFishProsody(raw any) (*provideropts.FishProsody, error) {
	fields, err := synthesisFields(raw)
	if err != nil {
		return nil, fmt.Errorf("prosody: %w", err)
	}
	value := &provideropts.FishProsody{}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if key != "normalize_loudness" {
			return nil, fmt.Errorf("unknown prosody option")
		}
		value.NormalizeLoudness, err = synthesisBool(fields[key], "prosody.normalize_loudness")
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}
