package provideropts

import (
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ElevenLabs overrides the corresponding fields on each Create Speech request.
// Nil fields preserve the adapter's existing request without an explicit override.
type ElevenLabs struct {
	Seed                            *int64                                  `yaml:"seed,omitempty"`
	ApplyTextNormalization          *ElevenTextNormalization                `yaml:"apply_text_normalization,omitempty"`
	ApplyLanguageTextNormalization  *bool                                   `yaml:"apply_language_text_normalization,omitempty"`
	VoiceSettings                   *ElevenVoiceSettings                    `yaml:"voice_settings,omitempty"`
	PronunciationDictionaryLocators *[]ElevenPronunciationDictionaryLocator `yaml:"pronunciation_dictionary_locators,omitempty"`
}

// ElevenVoiceSettings overrides stored voice settings only for this request.
// Speed remains a DSL control, not a second configuration source.
type ElevenVoiceSettings struct {
	Stability       *float64 `yaml:"stability,omitempty"`
	SimilarityBoost *float64 `yaml:"similarity_boost,omitempty"`
	Style           *float64 `yaml:"style,omitempty"`
	UseSpeakerBoost *bool    `yaml:"use_speaker_boost,omitempty"`
}

// ElevenPronunciationDictionaryLocator references an existing dictionary.
// An omitted version selects the latest version; no dictionary is created.
type ElevenPronunciationDictionaryLocator struct {
	PronunciationDictionaryID string  `yaml:"pronunciation_dictionary_id"`
	VersionID                 *string `yaml:"version_id,omitempty"`
}

type ElevenTextNormalization string

const (
	ElevenNormalizationAuto ElevenTextNormalization = "auto"
	ElevenNormalizationOn   ElevenTextNormalization = "on"
	ElevenNormalizationOff  ElevenTextNormalization = "off"
)

func (ElevenLabs) providerOptions() {}

// Validate checks scalar constraints before any integer narrowing or HTTP calls.
func (v ElevenLabs) Validate() error {
	if v.Seed != nil && (*v.Seed < 0 || *v.Seed > 4294967295) {
		return fmt.Errorf("seed must be an integer between 0 and 4294967295")
	}
	if v.ApplyTextNormalization != nil {
		switch *v.ApplyTextNormalization {
		case ElevenNormalizationAuto, ElevenNormalizationOn, ElevenNormalizationOff:
		default:
			return fmt.Errorf("apply_text_normalization must be auto, on, or off")
		}
	}
	if v.VoiceSettings != nil {
		if err := v.VoiceSettings.validate(); err != nil {
			return err
		}
	}
	if v.PronunciationDictionaryLocators != nil {
		// The endpoint documents ordered application and at most three entries.
		// It does not require uniqueness, so preserve order and duplicates.
		// https://elevenlabs.io/docs/api-reference/text-to-speech/convert
		if len(*v.PronunciationDictionaryLocators) > 3 {
			return fmt.Errorf("pronunciation_dictionary_locators must contain at most 3 entries")
		}
		for i, locator := range *v.PronunciationDictionaryLocators {
			if !validElevenID(locator.PronunciationDictionaryID) {
				return fmt.Errorf("pronunciation_dictionary_locators[%d].pronunciation_dictionary_id must be a nonempty identifier without whitespace or control characters", i)
			}
			if locator.VersionID != nil && !validElevenID(*locator.VersionID) {
				return fmt.Errorf("pronunciation_dictionary_locators[%d].version_id must be a nonempty identifier without whitespace or control characters", i)
			}
		}
	}
	return nil
}

func (v ElevenVoiceSettings) validate() error {
	// Only stability and similarity_boost have documented numeric bounds.
	// Style has no published range here; require JSON-compatible finite numbers.
	// https://elevenlabs.io/docs/api-reference/voices/settings/update
	for _, field := range []struct {
		name    string
		value   *float64
		bounded bool
	}{
		{"stability", v.Stability, true},
		{"similarity_boost", v.SimilarityBoost, true},
		{"style", v.Style, false},
	} {
		if field.value == nil {
			continue
		}
		if math.IsNaN(*field.value) || math.IsInf(*field.value, 0) {
			return fmt.Errorf("voice_settings.%s must be finite", field.name)
		}
		if field.bounded && (*field.value < 0 || *field.value > 1) {
			return fmt.Errorf("voice_settings.%s must be between 0 and 1", field.name)
		}
	}
	return nil
}

func validElevenID(id string) bool {
	// Reject empty, invalid UTF-8 and whitespace/control text as local input
	// hygiene, not a vendor ID syntax. The schema defines opaque strings with
	// no length or alphabet restriction.
	// https://github.com/elevenlabs/elevenlabs-python/blob/main/src/elevenlabs/types/pronunciation_dictionary_version_locator.py
	return id != "" && utf8.ValidString(id) && strings.IndexFunc(id, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) < 0
}
