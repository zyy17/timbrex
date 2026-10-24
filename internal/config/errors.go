package config

import (
	"errors"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
)

func safeDecodeError(err error) error {
	// Decode errors may include arbitrary input keys and values. Retain only a
	// recognized schema path, never the library's original diagnostic text.
	var detail *mapstructure.DecodeError
	if errors.As(err, &detail) {
		if field := knownConfigField(detail.Name()); field != "" {
			return fmt.Errorf("invalid configuration at %s; check field names and types", field)
		}
	}
	return fmt.Errorf("invalid configuration fields or types; check the YAML schema")
}

func knownConfigField(name string) string {
	switch name {
	case "provider", "strict_tags", "providers":
		return name
	}
	for id := range providerDefaults {
		path, decoded := "providers."+id, "providers["+id+"]"
		if name == path || name == decoded {
			return path
		}
		for _, key := range []string{"api_key", "model", "voice", "endpoint", "region"} {
			if name == path+"."+key || name == decoded+"."+key {
				return path + "." + key
			}
		}
	}
	return ""
}
