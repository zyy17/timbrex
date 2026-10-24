package provideropts

import "fmt"

// ElevenLabs overrides the corresponding fields on each Create Speech request.
// Nil fields preserve the adapter's existing request without an explicit override.
type ElevenLabs struct {
	Seed                           *int64                   `yaml:"seed,omitempty"`
	ApplyTextNormalization         *ElevenTextNormalization `yaml:"apply_text_normalization,omitempty"`
	ApplyLanguageTextNormalization *bool                    `yaml:"apply_language_text_normalization,omitempty"`
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
	return nil
}
