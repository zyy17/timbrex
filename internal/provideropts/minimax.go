package provideropts

import "fmt"

// MiniMax overrides the corresponding fields on each synchronous T2A request.
type MiniMax struct {
	TextNormalizationMode *MiniMaxTextNormalization `yaml:"text_normalization_mode,omitempty"`
}

type MiniMaxTextNormalization string

const (
	MiniMaxNormalizationBasic   MiniMaxTextNormalization = "basic"
	MiniMaxNormalizationQuality MiniMaxTextNormalization = "quality"
)

func (MiniMax) providerOptions() {}

// Validate checks scalar constraints; the adapter also checks model eligibility.
func (v MiniMax) Validate() error {
	// https://platform.minimax.io/docs/api-reference/speech-t2a-http.md
	if v.TextNormalizationMode != nil {
		switch *v.TextNormalizationMode {
		case MiniMaxNormalizationBasic, MiniMaxNormalizationQuality:
		default:
			return fmt.Errorf("text_normalization_mode must be basic or quality")
		}
	}
	return nil
}
