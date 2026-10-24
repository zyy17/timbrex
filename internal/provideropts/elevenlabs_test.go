package provideropts

import (
	"math"
	"strings"
	"testing"
)

type embeddedValue struct{ Value }

func TestValidateSynthesisOptions(t *testing.T) {
	for _, seed := range []int64{-9223372036854775808, -1, 0, 1, 4294967295, 4294967296, 9223372036854775807} {
		err := Validate("elevenlabs", ElevenLabs{Seed: &seed})
		if (err != nil) != (seed < 0 || seed > 4294967295) {
			t.Errorf("seed %d: %v", seed, err)
		}
	}
	for _, norm := range []ElevenTextNormalization{"auto", "on", "off", "", "AUTO", "false", "unknown"} {
		err := Validate("elevenlabs", ElevenLabs{ApplyTextNormalization: &norm})
		valid := norm == "auto" || norm == "on" || norm == "off"
		if (err == nil) != valid {
			t.Errorf("normalization %q: %v", norm, err)
		}
	}
	for _, id := range []string{"elevenlabs", "fish", "openai", "stub"} {
		if err := Validate(id, nil); err != nil {
			t.Errorf("omitted %s options: %v", id, err)
		}
		for _, value := range []Value{(*ElevenLabs)(nil), &ElevenLabs{}, embeddedValue{}, (*embeddedValue)(nil)} {
			if err := Validate(id, value); err == nil {
				t.Errorf("%s accepted a pointer or embedded options variant", id)
			}
		}
		if id != "elevenlabs" && Validate(id, ElevenLabs{}) == nil {
			t.Errorf("%s accepted another vendor's options", id)
		}
	}
}

func TestElevenVoiceSettingsValidation(t *testing.T) {
	for _, field := range []string{"stability", "similarity_boost", "style"} {
		for _, number := range []float64{-1, -math.SmallestNonzeroFloat64, 0, 0.5, 1, math.Nextafter(1, 2), math.MaxFloat64, math.NaN(), math.Inf(-1), math.Inf(1)} {
			settings := &ElevenVoiceSettings{}
			switch field {
			case "stability":
				settings.Stability = &number
			case "similarity_boost":
				settings.SimilarityBoost = &number
			case "style":
				settings.Style = &number
			}
			value := ElevenLabs{VoiceSettings: settings}
			wantErr := math.IsNaN(number) || math.IsInf(number, 0) || field != "style" && (number < 0 || number > 1)
			// Both the direct variant and shared dispatcher enforce constraints.
			for _, err := range []error{value.Validate(), Validate("elevenlabs", value)} {
				if (err != nil) != wantErr || err != nil && !strings.Contains(err.Error(), "voice_settings."+field) {
					t.Errorf("%s=%g: error=%v, wantErr=%v", field, number, err, wantErr)
				}
			}
		}
	}
	for _, boost := range []bool{false, true} {
		if err := (ElevenLabs{VoiceSettings: &ElevenVoiceSettings{UseSpeakerBoost: &boost}}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestElevenPronunciationDictionaryValidation(t *testing.T) {
	for _, count := range []int{0, 1, 2, 3, 4} {
		locators := make([]ElevenPronunciationDictionaryLocator, count)
		for i := range locators {
			// Duplicate locators count toward the limit, but are not rejected.
			locators[i] = ElevenPronunciationDictionaryLocator{PronunciationDictionaryID: "dictionary"}
		}
		if err := (ElevenLabs{PronunciationDictionaryLocators: &locators}).Validate(); (err != nil) != (count > 3) {
			t.Errorf("count=%d: %v", count, err)
		}
	}
	for _, id := range []string{"", " ", "has space", " leading", "trailing ", "a\tb", "a\nb", "a\x00b", "a\x7fb", "a\u00a0b", "a\xffb"} {
		for _, field := range []string{"pronunciation_dictionary_id", "version_id"} {
			locators := []ElevenPronunciationDictionaryLocator{{PronunciationDictionaryID: "dictionary"}}
			if field == "version_id" {
				locators[0].VersionID = &id
			} else {
				locators[0].PronunciationDictionaryID = id
			}
			value := ElevenLabs{PronunciationDictionaryLocators: &locators}
			for _, err := range []error{value.Validate(), Validate("elevenlabs", value)} {
				if err == nil || !strings.Contains(err.Error(), "[0]."+field) {
					t.Errorf("%s=%q: %v", field, id, err)
				}
			}
		}
	}
	// The service defines opaque string IDs, not an ASCII alphabet or fixed length.
	for _, id := range []string{"a", "ABC-123_xyz", "dictionary.v1", "字典"} {
		locators := []ElevenPronunciationDictionaryLocator{{PronunciationDictionaryID: id, VersionID: &id}}
		if err := (ElevenLabs{PronunciationDictionaryLocators: &locators}).Validate(); err != nil {
			t.Errorf("rejected opaque ID %q: %v", id, err)
		}
	}
}
