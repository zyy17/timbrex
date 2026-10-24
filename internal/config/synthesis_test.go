package config

import (
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/providers/options"
)

func TestElevenLabsSynthesisConfiguration(t *testing.T) {
	c, err := loadTest(t, `providers:
  elevenlabs:
    api_key: FAKE-SECRET-DO-NOT-LOG
    synthesis:
      seed: 0
      apply_text_normalization: off
      apply_language_text_normalization: false
`)
	if err != nil {
		t.Fatal(err)
	}
	data, err := c.RedactedYAML()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"seed: 0", "apply_text_normalization: \"off\"", "apply_language_text_normalization: false", "api_key: REDACTED"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("redacted YAML omitted %s: %s", want, data)
		}
	}
	if strings.Contains(string(data), "FAKE-SECRET-DO-NOT-LOG") {
		t.Fatal("credential leaked")
	}
}

func TestRejectInvalidSynthesisConfiguration(t *testing.T) {
	for _, value := range []string{
		"null", "[]", "secret-value", "true",
		"{seed: null}", "{seed: -1}", "{seed: 4294967296}",
		"{seed: 9223372036854775808}", "{seed: 18446744073709551616}",
		"{seed: 1.5}", "{seed: 1.0}", "{seed: 1e2}", "{seed: .nan}", "{seed: .inf}",
		"{seed: '0'}", "{seed: false}", "{seed: []}", "{seed: {secret-value: null}}",
		"{apply_text_normalization: null}", "{apply_text_normalization: secret-value}",
		"{apply_text_normalization: ON}", "{apply_text_normalization: false}",
		"{apply_text_normalization: 0}", "{apply_text_normalization: [secret-value]}",
		"{apply_language_text_normalization: null}", "{apply_language_text_normalization: 'false'}",
		"{apply_language_text_normalization: 0}", "{apply_language_text_normalization: yes}",
		"{apply_language_text_normalization: [secret-value]}", "{secret-value: secret-value}",
		"{normalize: false}", "{Seed: 0}", "{seed: 0, seed: 1}",
	} {
		t.Run(value, func(t *testing.T) {
			_, err := loadTest(t, "providers: {elevenlabs: {synthesis: "+value+"}}", "ELEVENLABS_MODEL=override-model")
			if err == nil || strings.Contains(err.Error(), "secret-value") {
				t.Fatalf("invalid synthesis options were accepted or leaked a value: %v", err)
			}
		})
	}
	for _, id := range []string{"stub", "openai", "gemini"} {
		for _, value := range []string{"{}", "null", "{seed: 0}"} {
			if _, err := loadTest(t, "providers: {"+id+": {synthesis: "+value+"}}"); err == nil {
				t.Errorf("accepted %s synthesis options %s", id, value)
			}
		}
	}
}

func TestSynthesisConfigurationPresenceAndResolution(t *testing.T) {
	for _, body := range []string{"", "providers: {elevenlabs: {synthesis: {}}}", "providers: {elevenlabs: {synthesis: {seed: 4294967295, apply_text_normalization: auto, apply_language_text_normalization: true}}}"} {
		c, err := loadTest(t, body, "ELEVENLABS_API_KEY=test-key", "ELEVENLABS_MODEL=environment-model")
		if err != nil {
			t.Fatal(err)
		}
		p, err := c.Resolve("elevenlabs", "override-model", "override-voice")
		if err != nil || p.Model != "override-model" || p.Voice != "override-voice" {
			t.Fatalf("resolution failed: %+v %v", p, err)
		}
		if body == "" {
			if p.Synthesis != nil {
				t.Fatal("omission created an options override")
			}
			continue
		}
		value, ok := p.Synthesis.(options.ElevenLabs)
		if !ok {
			t.Fatalf("options have wrong concrete type %T", p.Synthesis)
		}
		if strings.Contains(body, "4294967295") {
			if value.Seed == nil || *value.Seed != 4294967295 || value.ApplyTextNormalization == nil || *value.ApplyTextNormalization != "auto" || value.ApplyLanguageTextNormalization == nil || !*value.ApplyLanguageTextNormalization {
				t.Fatalf("options changed: %+v", value)
			}
		} else if value.Seed != nil || value.ApplyTextNormalization != nil || value.ApplyLanguageTextNormalization != nil {
			t.Fatalf("empty options supplied fields: %+v", value)
		}
		redacted, err := c.RedactedYAML()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := loadTest(t, string(redacted)); err != nil {
			t.Fatalf("redacted options cannot round trip: %v", err)
		}
		if c.Providers["elevenlabs"].Model != "environment-model" || c.Providers["elevenlabs"].APIKey != "test-key" {
			t.Fatal("resolve or redaction mutated configuration")
		}
	}
}

func TestElevenLabsSynthesisExample(t *testing.T) {
	c, err := load("../../examples/elevenlabs-synthesis.yaml", func() []string { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if c.Provider != "elevenlabs" {
		t.Fatal("example does not select ElevenLabs")
	}
	if _, ok := c.Providers["elevenlabs"].Synthesis.(options.ElevenLabs); !ok {
		t.Fatal("example did not load typed options")
	}
}

func TestSynthesisRejectsDottedConfigurationKeys(t *testing.T) {
	for _, body := range []string{
		"providers: {elevenlabs.synthesis: {seed: 42}}",
		"providers.elevenlabs.synthesis: {seed: 42}",
		"providers.elevenlabs: {synthesis: {seed: 42}}",
		"providers: {elevenlabs: {synthesis.seed: 42}}",
		"providers: {elevenlabs: {synthesis: {seed: 42}}, elevenlabs.synthesis: {seed: 1}}",
		"providers: {elevenlabs: {synthesis: {seed: 42}}}\nproviders.elevenlabs.synthesis: {seed: 1}",
		"providers: {elevenlabs: {synthesis: {seed: 42}, synthesis.seed: 1}}",
	} {
		t.Run(body, func(t *testing.T) {
			if _, err := loadTest(t, body); err == nil {
				t.Fatal("accepted a dotted key outside the configuration schema")
			}
		})
	}
}
