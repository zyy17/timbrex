// Package options defines and decodes typed provider settings without I/O.
package options

import "fmt"

// Value is a closed set of provider-specific synthesis settings.
// Only concrete value variants are accepted, not pointers or embedded wrappers.
type Value interface {
	providerOptions()
}

// Decode selects a closed schema by provider key, never by a supplied discriminator.
// Raw values are consumed here; callers receive only typed settings.
func Decode(id string, raw any) (Value, error) {
	switch id {
	case "azure":
		return decodeAzure(raw)
	case "elevenlabs":
		return decodeElevenLabs(raw)
	case "fish":
		return decodeFish(raw)
	case "qwen":
		return decodeQwen(raw)
	case "minimax":
		return decodeMiniMax(raw)
	default:
		return nil, fmt.Errorf("synthesis options are not supported")
	}
}
