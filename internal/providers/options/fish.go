package options

import (
	"fmt"
	"maps"
	"math"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// Fish overrides fields on each single-speaker TTS request. Nil fields preserve
// the adapter's existing behavior, including normalize=false and loudness=true.
type Fish struct {
	PronunciationDictionary   *FishDictionaries `yaml:"pronunciation_dictionary,omitempty"`
	Features                  *[]FishFeature    `yaml:"features,omitempty"`
	Normalize                 *bool             `yaml:"normalize,omitempty"`
	NormalizationLanguage     *string           `yaml:"normalization_language,omitempty"`
	Prosody                   *FishProsody      `yaml:"prosody,omitempty"`
	Temperature               *float64          `yaml:"temperature,omitempty"`
	TopP                      *float64          `yaml:"top_p,omitempty"`
	ChunkLength               *int64            `yaml:"chunk_length,omitempty"`
	MinChunkLength            *int64            `yaml:"min_chunk_length,omitempty"`
	Latency                   *FishLatency      `yaml:"latency,omitempty"`
	ConditionOnPreviousChunks *bool             `yaml:"condition_on_previous_chunks,omitempty"`
	EarlyStopThreshold        *float64          `yaml:"early_stop_threshold,omitempty"`
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
	if v.Features != nil {
		for _, feature := range *v.Features {
			if feature != FishQualityGuard {
				return fmt.Errorf("features only supports quality-guard; availability depends on the inference backend")
			}
		}
	}
	if v.PronunciationDictionary != nil {
		return v.PronunciationDictionary.Validate()
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
		case "pronunciation_dictionary":
			value.PronunciationDictionary, err = decodeFishDictionaries(fields[key])
		case "features":
			value.Features, err = decodeFishFeatures(fields[key])
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

// FishDictionaries is the homogeneous array union accepted by Fish TTS.
// Exactly one branch must be present; a pointer to an empty slice sends [].
type FishDictionaries struct {
	References *[]FishDictionaryReference
	Inline     *[]FishDictionaryInline
}

type FishDictionaryReference struct {
	ID      string `yaml:"id" json:"id"`
	Version string `yaml:"version" json:"version"`
}

type FishDictionaryInline struct {
	Items []FishPronunciationItem `yaml:"items" json:"items"`
}

type FishPronunciationItem struct {
	Key           string `yaml:"key" json:"key"`
	Value         string `yaml:"value" json:"value"`
	CaseSensitive *bool  `yaml:"case_sensitive,omitempty" json:"case_sensitive,omitempty"`
}

type FishFeature string

const FishQualityGuard FishFeature = "quality-guard"

var fishDictionaryID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// MarshalYAML retains the API's array shape rather than exposing the Go union.
func (v FishDictionaries) MarshalYAML() (any, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	if v.References != nil {
		return *v.References, nil
	}
	return *v.Inline, nil
}

// Validate follows PronunciationDictionaryRef, PronunciationDictionaryInline,
// and PronunciationItem in the official TTS OpenAPI schema. These are phonetic
// substitutions across the request, not occurrence-local DSL annotations.
// https://github.com/fishaudio/docs/blob/main/api-reference/openapi.json
func (v FishDictionaries) Validate() error {
	if (v.References == nil) == (v.Inline == nil) {
		return fmt.Errorf("pronunciation_dictionary requires exactly one homogeneous reference or inline list")
	}
	if v.References != nil {
		if len(*v.References) > 3 {
			return fmt.Errorf("pronunciation_dictionary supports at most 3 dictionaries")
		}
		for _, ref := range *v.References {
			if !fishDictionaryID.MatchString(ref.ID) || !fishDictionaryID.MatchString(ref.Version) {
				return fmt.Errorf("pronunciation_dictionary id and version must each contain 1 to 128 ASCII letters, digits, underscores, or hyphens")
			}
		}
		return nil
	}
	if len(*v.Inline) > 3 {
		return fmt.Errorf("pronunciation_dictionary supports at most 3 dictionaries")
	}
	for _, dictionary := range *v.Inline {
		if dictionary.Items == nil || len(dictionary.Items) > 5000 {
			return fmt.Errorf("pronunciation_dictionary items must be a non-null list with at most 5000 entries")
		}
		for _, item := range dictionary.Items {
			if !fishPronunciationText(item.Key, 256) || !fishPronunciationText(item.Value, 1024) {
				return fmt.Errorf("pronunciation_dictionary key/value must be valid UTF-8 with 1 to 256/1024 characters and no phoneme boundary markers")
			}
		}
	}
	return nil
}

func fishPronunciationText(text string, maximum int) bool {
	return utf8.ValidString(text) && utf8.RuneCountInString(text) >= 1 && utf8.RuneCountInString(text) <= maximum &&
		!strings.Contains(text, "<|phoneme_start|>") && !strings.Contains(text, "<|phoneme_end|>")
}

func decodeFishFeatures(raw any) (*[]FishFeature, error) {
	items, err := synthesisList(raw)
	if err != nil {
		return nil, fmt.Errorf("features: %w", err)
	}
	result := make([]FishFeature, len(items))
	for i, item := range items {
		feature, err := synthesisString[FishFeature](item, "features entry")
		if err != nil {
			return nil, err
		}
		result[i] = *feature
	}
	return &result, nil
}

func decodeFishDictionaries(raw any) (*FishDictionaries, error) {
	items, err := synthesisList(raw)
	if err != nil {
		return nil, fmt.Errorf("pronunciation_dictionary: %w", err)
	}
	if len(items) > 3 {
		return nil, fmt.Errorf("pronunciation_dictionary supports at most 3 dictionaries")
	}
	refs := make([]FishDictionaryReference, 0, len(items))
	inline := make([]FishDictionaryInline, 0, len(items))
	for _, item := range items {
		fields, err := synthesisFields(item)
		if err != nil {
			return nil, fmt.Errorf("pronunciation_dictionary entry: %w", err)
		}
		if entries, ok := fields["items"]; ok {
			if len(fields) != 1 {
				return nil, fmt.Errorf("inline pronunciation_dictionary entry only supports items")
			}
			value, err := decodeFishPronunciationItems(entries)
			if err != nil {
				return nil, err
			}
			inline = append(inline, FishDictionaryInline{Items: value})
		} else {
			ref, err := decodeFishDictionaryReference(fields)
			if err != nil {
				return nil, err
			}
			refs = append(refs, ref)
		}
	}
	if len(refs) != 0 && len(inline) != 0 {
		return nil, fmt.Errorf("pronunciation_dictionary cannot mix references and inline dictionaries")
	}
	if len(inline) != 0 {
		return &FishDictionaries{Inline: &inline}, nil
	}
	return &FishDictionaries{References: &refs}, nil
}

func decodeFishDictionaryReference(fields map[string]any) (FishDictionaryReference, error) {
	var value FishDictionaryReference
	if len(fields) != 2 {
		return value, fmt.Errorf("pronunciation_dictionary reference requires only id and version")
	}
	id, err := synthesisString[string](fields["id"], "pronunciation_dictionary id")
	if err != nil {
		return value, err
	}
	version, err := synthesisString[string](fields["version"], "pronunciation_dictionary version")
	if err != nil {
		return value, err
	}
	return FishDictionaryReference{ID: *id, Version: *version}, nil
}

func decodeFishPronunciationItems(raw any) ([]FishPronunciationItem, error) {
	items, err := synthesisList(raw)
	if err != nil {
		return nil, fmt.Errorf("pronunciation_dictionary items: %w", err)
	}
	if len(items) > 5000 {
		return nil, fmt.Errorf("pronunciation_dictionary supports at most 5000 items per dictionary")
	}
	result := make([]FishPronunciationItem, len(items))
	for i, item := range items {
		fields, err := synthesisFields(item)
		if err != nil {
			return nil, fmt.Errorf("pronunciation_dictionary item: %w", err)
		}
		key, err := synthesisString[string](fields["key"], "pronunciation_dictionary key")
		if err != nil {
			return nil, err
		}
		value, err := synthesisString[string](fields["value"], "pronunciation_dictionary value")
		if err != nil {
			return nil, err
		}
		result[i] = FishPronunciationItem{Key: *key, Value: *value}
		for _, field := range slices.Sorted(maps.Keys(fields)) {
			switch field {
			case "key", "value":
			case "case_sensitive":
				result[i].CaseSensitive, err = synthesisBool(fields[field], "pronunciation_dictionary case_sensitive")
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("unknown pronunciation_dictionary item option")
			}
		}
	}
	return result, nil
}
