package provideropts

import "testing"

func TestClosedSynthesisVariants(t *testing.T) {
	variants := map[string]Value{
		"elevenlabs": ElevenLabs{},
		"fish":       Fish{},
		"qwen":       Qwen{},
		"minimax":    MiniMax{},
	}
	for _, id := range []string{"stub", "openai", "azure", "gemini", "elevenlabs", "fish", "qwen", "minimax"} {
		for owner, value := range variants {
			if err := Validate(id, value); (err == nil) != (id == owner) {
				t.Errorf("%s accepts only its own %s variant: %v", id, owner, err)
			}
		}
		for _, value := range []Value{
			&ElevenLabs{}, (*ElevenLabs)(nil), &Fish{}, (*Fish)(nil),
			&Qwen{}, (*Qwen)(nil), &MiniMax{}, (*MiniMax)(nil),
			embeddedValue{}, (*embeddedValue)(nil),
		} {
			if err := Validate(id, value); err == nil {
				t.Errorf("%s accepted pointer/embedded variant %T", id, value)
			}
		}
	}
}
