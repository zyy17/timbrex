package config

import (
	"fmt"
	"maps"
	"slices"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"

	"github.com/zyy17/timbrex/internal/provideropts"
)

// Separate vendor options before decoding the common envelope. Raw values stay
// inside deserialization; neither Config nor provider requests retain a map.
func splitSynthesis(k *koanf.Koanf) (*koanf.Koanf, map[string]provideropts.Value, error) {
	// Literal YAML keys must not be interpreted as Koanf's dotted lookup paths.
	// Leave unknown root/provider keys intact for the strict envelope decoder.
	raw := k.Raw()
	providers, _ := raw["providers"].(map[string]any)
	values := make(map[string]provideropts.Value)
	for _, id := range slices.Sorted(maps.Keys(providerDefaults)) {
		settings, ok := providers[id].(map[string]any)
		if !ok {
			continue
		}
		options, exists := settings["synthesis"]
		if !exists {
			continue
		}
		key := "providers." + id + ".synthesis"
		value, err := decodeSynthesis(id, options)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", key, err)
		}
		values[id] = value
		delete(settings, "synthesis")
	}
	common := koanf.New(".")
	if err := common.Load(confmap.Provider(raw, ""), nil); err != nil {
		return nil, nil, err
	}
	return common, values, nil
}

// Select a closed schema by the provider key, never a user-supplied discriminator.
func decodeSynthesis(id string, raw any) (provideropts.Value, error) {
	switch id {
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
