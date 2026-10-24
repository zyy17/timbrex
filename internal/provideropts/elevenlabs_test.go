package provideropts

import "testing"

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
