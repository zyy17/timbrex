package options

import (
	"fmt"
	"maps"
	"math"
	"regexp"
	"slices"
)

// Fish overrides fields on each single-speaker TTS request. Nil fields preserve
// the adapter's existing behavior, including normalize=false and loudness=true.
type Fish struct {
	Normalize                 *bool        `yaml:"normalize,omitempty"`
	NormalizationLanguage     *string      `yaml:"normalization_language,omitempty"`
	Prosody                   *FishProsody `yaml:"prosody,omitempty"`
	Temperature               *float64     `yaml:"temperature,omitempty"`
	TopP                      *float64     `yaml:"top_p,omitempty"`
	ChunkLength               *int64       `yaml:"chunk_length,omitempty"`
	MinChunkLength            *int64       `yaml:"min_chunk_length,omitempty"`
	Latency                   *FishLatency `yaml:"latency,omitempty"`
	ConditionOnPreviousChunks *bool        `yaml:"condition_on_previous_chunks,omitempty"`
	EarlyStopThreshold        *float64     `yaml:"early_stop_threshold,omitempty"`
}

// FishProsody leaves speed and volume under the existing DSL controls.
type FishProsody struct {
	NormalizeLoudness *bool `yaml:"normalize_loudness,omitempty"`
}

type FishLatency string

const (
	FishLatencyLow      FishLatency = "low"
	FishLatencyNormal   FishLatency = "normal"
	FishLatencyBalanced FishLatency = "balanced"
)

func (Fish) providerOptions() {}

var fishNormalizationLanguage = regexp.MustCompile(`^[A-Za-z0-9_-]*$`)

// Validate follows the official TTSRequest schema. Its language field is only
// for text normalization/alignment, not a spoken-language override. Empty and
// auto are valid; do not impose a stronger BCP-47 validator than the API uses.
// https://github.com/fishaudio/docs/blob/main/api-reference/openapi.json
// https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech
func (v Fish) Validate() error {
	if v.NormalizationLanguage != nil && (len(*v.NormalizationLanguage) > 35 || !fishNormalizationLanguage.MatchString(*v.NormalizationLanguage)) {
		return fmt.Errorf("normalization_language must contain at most 35 ASCII letters, digits, underscores, or hyphens")
	}
	for _, field := range []struct {
		name  string
		value *float64
	}{
		{"temperature", v.Temperature},
		{"top_p", v.TopP},
		{"early_stop_threshold", v.EarlyStopThreshold},
	} {
		if field.value != nil && (math.IsNaN(*field.value) || math.IsInf(*field.value, 0) || *field.value < 0 || *field.value > 1) {
			return fmt.Errorf("%s must be a finite number between 0 and 1", field.name)
		}
	}
	if v.ChunkLength != nil && (*v.ChunkLength < 100 || *v.ChunkLength > 300) {
		return fmt.Errorf("chunk_length must be an integer between 100 and 300")
	}
	if v.MinChunkLength != nil && (*v.MinChunkLength < 0 || *v.MinChunkLength > 100) {
		return fmt.Errorf("min_chunk_length must be an integer between 0 and 100")
	}
	if v.Latency != nil {
		switch *v.Latency {
		case FishLatencyLow, FishLatencyNormal, FishLatencyBalanced:
		default:
			return fmt.Errorf("latency must be low, normal, or balanced")
		}
	}
	return nil
}

func decodeFish(raw any) (Fish, error) {
	var value Fish
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
			value.Latency, err = synthesisString[FishLatency](fields[key], key)
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

func decodeFishProsody(raw any) (*FishProsody, error) {
	fields, err := synthesisFields(raw)
	if err != nil {
		return nil, fmt.Errorf("prosody: %w", err)
	}
	value := &FishProsody{}
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
