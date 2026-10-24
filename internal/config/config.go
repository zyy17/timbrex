// Package config loads typed settings once, before script execution.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/v2"
	"go.yaml.in/yaml/v3"

	"github.com/zyy17/timbrex/internal/inputfile"
	"github.com/zyy17/timbrex/internal/providers/options"
)

type Config struct {
	Provider   string              `koanf:"provider" yaml:"provider"`
	StrictTags bool                `koanf:"strict_tags" yaml:"strict_tags"`
	Providers  map[string]Provider `koanf:"providers" yaml:"providers"`
}

type Provider struct {
	Synthesis options.Value `koanf:"-" yaml:"synthesis,omitempty"`
	APIKey    string        `koanf:"api_key" yaml:"api_key,omitempty"`
	Model     string        `koanf:"model" yaml:"model,omitempty"`
	Voice     string        `koanf:"voice" yaml:"voice,omitempty"`
	Endpoint  string        `koanf:"endpoint" yaml:"endpoint,omitempty"`
	Region    string        `koanf:"region" yaml:"region,omitempty"`
}

// Load applies defaults, one YAML file, then the documented environment keys.
// An explicit path must exist; an absent default timbrex.yaml is allowed.
func Load(path string) (Config, error) {
	return load(path, os.Environ)
}

func load(path string, environ func() []string) (Config, error) {
	k := koanf.New(".")
	if err := k.Load(confmap.Provider(defaults(), "."), nil); err != nil {
		return Config{}, err
	}
	if err := loadFile(k, path); err != nil {
		return Config{}, err
	}
	// Validate the file before overrides can hide misspelled or mistyped values.
	if _, err := decode(k); err != nil {
		return Config{}, err
	}
	if err := k.Load(env.Provider(".", env.Opt{
		EnvironFunc: environ,
		TransformFunc: func(key, value string) (string, any) {
			name := envKeys[key]
			if name == "" || value == "" {
				return "", nil
			}
			if name == "strict_tags" {
				if b, err := strconv.ParseBool(value); err == nil {
					return name, b
				}
			}
			return name, value
		},
	}), nil); err != nil {
		return Config{}, err
	}
	return decode(k)
}

func loadFile(k *koanf.Koanf, path string) error {
	explicit := path != ""
	if !explicit {
		path = "timbrex.yaml"
	}
	data, err := inputfile.Read(path)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read configuration: %w", err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil && !errors.Is(err, io.EOF) {
		// YAML type errors may quote secrets from the input.
		return fmt.Errorf("configuration %q must be a YAML mapping", path)
	}
	if len(document.Content) > 0 && document.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("configuration %q must be a YAML mapping", path)
	}
	var values map[string]any
	if err := document.Decode(&values); err != nil {
		return fmt.Errorf("configuration %q contains invalid YAML fields", path)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("configuration %q must contain one YAML document", path)
	}
	return k.Load(confmap.Provider(values, ""), nil)
}

func decode(k *koanf.Koanf) (Config, error) {
	common, synthesis, err := splitSynthesis(k)
	if err != nil {
		return Config{}, err
	}
	var c Config
	err = common.UnmarshalWithConf("", &c, koanf.UnmarshalConf{
		DecoderConfig: &mapstructure.DecoderConfig{
			ErrorUnused: true,
			MatchName:   func(key, field string) bool { return key == field },
		},
	})
	if err != nil {
		return Config{}, safeDecodeError(err)
	}
	for _, key := range k.Keys() {
		if k.Get(key) == nil {
			if field := knownConfigField(key); field != "" {
				return Config{}, fmt.Errorf("configuration %s must not be null", field)
			}
			return Config{}, fmt.Errorf("configuration fields must not be null")
		}
	}
	if _, ok := providerDefaults[c.Provider]; !ok {
		return Config{}, fmt.Errorf("unknown default provider")
	}
	for _, id := range slices.Sorted(maps.Keys(c.Providers)) {
		if _, ok := providerDefaults[id]; !ok {
			return Config{}, fmt.Errorf("unknown provider configuration")
		}
		p := c.Providers[id]
		p.Synthesis = synthesis[id]
		c.Providers[id] = p
		if p.Endpoint != "" {
			if err := validateEndpoint(p.Endpoint); err != nil {
				return Config{}, fmt.Errorf("providers.%s.endpoint: %w", id, err)
			}
		}
		if p.Region != "" && (id != "azure" || strings.IndexFunc(p.Region, func(r rune) bool {
			return (r < 'a' || r > 'z') && (r < '0' || r > '9')
		}) >= 0) {
			return Config{}, fmt.Errorf("providers.%s.region must be an Azure region ID", id)
		}
	}
	return c, nil
}

// Resolve applies script/flag overrides and requires the selected provider's credentials.
func (c Config) Resolve(id, model, voice string) (Provider, error) {
	p, ok := c.Providers[id]
	if !ok {
		return Provider{}, fmt.Errorf("unknown provider %q", id)
	}
	p.Model = First(model, p.Model)
	p.Voice = First(voice, p.Voice)
	if id == "stub" {
		return p, nil
	}
	if strings.TrimSpace(p.APIKey) == "" {
		return Provider{}, fmt.Errorf("set providers.%s.api_key or its documented environment variable", id)
	}
	if id == "azure" && p.Endpoint == "" && p.Region != "" {
		p.Endpoint = "https://" + p.Region + ".tts.speech.microsoft.com/cognitiveservices/v1"
	}
	if p.Endpoint == "" {
		if id == "azure" {
			return Provider{}, fmt.Errorf("set providers.azure.endpoint or providers.azure.region")
		}
		return Provider{}, fmt.Errorf("set providers.%s.endpoint", id)
	}
	if id != "azure" && p.Model == "" {
		return Provider{}, fmt.Errorf("set providers.%s.model", id)
	}
	return p, nil
}

func validateEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("expected an HTTP(S) URL without credentials, query, or fragment")
	}
	return nil
}

func (c Config) RedactedYAML() ([]byte, error) {
	c.Providers = maps.Clone(c.Providers)
	for id, p := range c.Providers {
		if p.APIKey != "" {
			p.APIKey = "REDACTED"
			c.Providers[id] = p
		}
	}
	return yaml.Marshal(c)
}

func First(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
