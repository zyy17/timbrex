package config

import (
	"fmt"
	"maps"
	"slices"

	"github.com/zyy17/timbrex/internal/provideropts"
)

func decodeElevenLabs(raw any) (provideropts.ElevenLabs, error) {
	var value provideropts.ElevenLabs
	fields, err := synthesisFields(raw)
	if err != nil {
		return value, err
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		switch key {
		case "seed":
			value.Seed, err = synthesisInteger(fields[key], key)
		case "apply_text_normalization":
			value.ApplyTextNormalization, err = synthesisString[provideropts.ElevenTextNormalization](fields[key], key)
		case "apply_language_text_normalization":
			value.ApplyLanguageTextNormalization, err = synthesisBool(fields[key], key)
		case "voice_settings":
			value.VoiceSettings, err = decodeElevenVoiceSettings(fields[key])
		case "pronunciation_dictionary_locators":
			value.PronunciationDictionaryLocators, err = decodeElevenLocators(fields[key])
		default:
			return value, fmt.Errorf("unknown synthesis option")
		}
		if err != nil {
			return value, err
		}
	}
	return value, value.Validate()
}

func decodeElevenVoiceSettings(raw any) (*provideropts.ElevenVoiceSettings, error) {
	fields, err := synthesisFields(raw)
	if err != nil {
		return nil, fmt.Errorf("voice_settings: %w", err)
	}
	value := &provideropts.ElevenVoiceSettings{}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		switch key {
		case "stability":
			value.Stability, err = synthesisNumber(fields[key], "voice_settings.stability")
		case "similarity_boost":
			value.SimilarityBoost, err = synthesisNumber(fields[key], "voice_settings.similarity_boost")
		case "style":
			value.Style, err = synthesisNumber(fields[key], "voice_settings.style")
		case "use_speaker_boost":
			value.UseSpeakerBoost, err = synthesisBool(fields[key], "voice_settings.use_speaker_boost")
		default:
			return nil, fmt.Errorf("unknown voice_settings option")
		}
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}

func decodeElevenLocators(raw any) (*[]provideropts.ElevenPronunciationDictionaryLocator, error) {
	items, err := synthesisList(raw)
	if err != nil {
		return nil, fmt.Errorf("pronunciation_dictionary_locators: %w", err)
	}
	result := make([]provideropts.ElevenPronunciationDictionaryLocator, 0, len(items))
	for _, item := range items {
		fields, err := synthesisFields(item)
		if err != nil {
			return nil, fmt.Errorf("pronunciation_dictionary_locators entry: %w", err)
		}
		var value provideropts.ElevenPronunciationDictionaryLocator
		if _, ok := fields["pronunciation_dictionary_id"]; !ok {
			return nil, fmt.Errorf("pronunciation_dictionary_locators entry requires pronunciation_dictionary_id")
		}
		for _, key := range slices.Sorted(maps.Keys(fields)) {
			switch key {
			case "pronunciation_dictionary_id":
				id, err := synthesisString[string](fields[key], key)
				if err != nil {
					return nil, err
				}
				value.PronunciationDictionaryID = *id
			case "version_id":
				value.VersionID, err = synthesisString[string](fields[key], key)
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("unknown pronunciation_dictionary_locators entry option")
			}
		}
		result = append(result, value)
	}
	return &result, nil
}
