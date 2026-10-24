// Package provideropts defines typed, per-request provider settings without I/O.
package provideropts

import "fmt"

// Value is a closed set of provider-specific synthesis settings.
// Only concrete value variants are accepted, not pointers or embedded wrappers.
type Value interface {
	providerOptions()
}

// Validate rejects unknown variants and settings meant for a different provider.
func Validate(id string, value Value) error {
	switch v := value.(type) {
	case nil:
		return nil
	case ElevenLabs:
		if id != "elevenlabs" {
			return fmt.Errorf("%s does not support ElevenLabs synthesis options", id)
		}
		return v.Validate()
	case Fish:
		if id != "fish" {
			return fmt.Errorf("%s does not support Fish synthesis options", id)
		}
		return v.Validate()
	case Qwen:
		if id != "qwen" {
			return fmt.Errorf("%s does not support Qwen synthesis options", id)
		}
		return v.Validate()
	case MiniMax:
		if id != "minimax" {
			return fmt.Errorf("%s does not support MiniMax synthesis options", id)
		}
		return v.Validate()
	default:
		return fmt.Errorf("%s received an unsupported synthesis options type", id)
	}
}
