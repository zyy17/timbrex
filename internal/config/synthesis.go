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
		if id != "elevenlabs" {
			return nil, nil, fmt.Errorf("%s is not supported", key)
		}
		value, err := decodeElevenLabs(options)
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

func decodeElevenLabs(raw any) (provideropts.ElevenLabs, error) {
	var value provideropts.ElevenLabs
	fields, ok := raw.(map[string]any)
	if !ok {
		return value, fmt.Errorf("expected a non-null mapping")
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		switch key {
		case "seed":
			var seed int64
			switch n := fields[key].(type) {
			case int:
				seed = int64(n)
			case int64:
				seed = n
			case uint64:
				if n > 4294967295 {
					return value, fmt.Errorf("seed must be an integer between 0 and 4294967295")
				}
				seed = int64(n)
			default:
				return value, fmt.Errorf("seed must be an integer between 0 and 4294967295")
			}
			value.Seed = &seed
		case "apply_text_normalization":
			s, ok := fields[key].(string)
			if !ok {
				return value, fmt.Errorf("apply_text_normalization must be auto, on, or off")
			}
			normalization := provideropts.ElevenTextNormalization(s)
			value.ApplyTextNormalization = &normalization
		case "apply_language_text_normalization":
			b, ok := fields[key].(bool)
			if !ok {
				return value, fmt.Errorf("apply_language_text_normalization must be a boolean")
			}
			value.ApplyLanguageTextNormalization = &b
		default:
			// Do not echo arbitrary keys or values, which may contain credentials.
			return value, fmt.Errorf("unknown synthesis option")
		}
	}
	return value, value.Validate()
}
