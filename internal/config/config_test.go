package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadTest(t *testing.T, body string, variables ...string) (Config, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return load(path, func() []string { return variables })
}

func TestDefaultsFileEnvironmentAndOverrides(t *testing.T) {
	c, err := loadTest(t, `provider: openai
strict_tags: true
providers:
  openai:
    api_key: file-key
    model: file-model
    voice: file-voice
`, "OPENAI_API_KEY=env-key", "OPENAI_MODEL=env-model", "OPENAI_VOICE=", "TIMBREX_STRICT_TAGS=false", "UNRELATED=ignored")
	if err != nil {
		t.Fatal(err)
	}
	p := c.Providers["openai"]
	if c.Provider != "openai" || c.StrictTags || p.APIKey != "env-key" || p.Model != "env-model" || p.Voice != "file-voice" || p.Endpoint != "https://api.openai.com/v1" {
		t.Fatal("configuration precedence is incorrect")
	}
	resolved, err := c.Resolve("openai", "script-model", "flag-voice")
	if err != nil || resolved.Model != "script-model" || resolved.Voice != "flag-voice" || c.Providers["openai"].Voice != "file-voice" {
		t.Fatalf("overrides mutated configuration or failed: %v", err)
	}
}

func TestConfigFileDiscovery(t *testing.T) {
	t.Chdir(t.TempDir())
	environ := func() []string { return nil }
	c, err := load("", environ)
	if err != nil || c.Provider != "stub" || c.StrictTags {
		t.Fatalf("missing optional file: %v", err)
	}
	if _, err := load("missing.yaml", environ); err == nil {
		t.Fatal("accepted missing explicit file")
	}
	if err := os.WriteFile("timbrex.yaml", []byte("provider: qwen\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err = load("", environ)
	if err != nil || c.Provider != "qwen" {
		t.Fatalf("default file not loaded: %v", err)
	}
	c, err = loadTest(t, "provider: openai\n")
	if err != nil || c.Provider != "openai" {
		t.Fatalf("explicit file not selected: %v", err)
	}
}

func TestConfigRegularFileInputs(t *testing.T) {
	for _, body := range []string{"", "strict_tags: true\n"} {
		t.Run(body, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if err := os.WriteFile("settings.yaml", []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("settings.yaml", "timbrex.yaml"); err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{"settings.yaml", "timbrex.yaml", ""} {
				cfg, err := load(path, func() []string { return nil })
				if err != nil || cfg.Provider != "stub" || cfg.StrictTags != (body != "") {
					t.Fatalf("load(%q) = %+v, %v", path, cfg, err)
				}
			}
		})
	}
}

func TestRejectInvalidConfigurationWithoutLeakingValues(t *testing.T) {
	for _, body := range []string{
		"strict_tgas: true",
		"strict_tags: secret-value",
		"Strict_Tags: true",
		"provider: missing",
		"providers: {missing: {voice: test}}",
		"providers: {openai: {voiec: test}}",
		"providers: {openai: {api_key: [secret-value]}}",
		"providers: {openai: {model: 42}}",
		"providers: {openai: {model: null}}",
		"providers: null",
		"null",
		"providers: {openai: {region: eastus}}",
		"providers: {azure: {region: 'east/us'}}",
		"providers: {qwen: {endpoint: 'https://user:secret-value@example.com'}}",
		"providers: {qwen: {endpoint: 'https://example.com?key=secret-value'}}",
		"providers: {qwen: {endpoint: 'file:///secret-value'}}",
		"provider: stub\nprovider: openai",
		"provider: stub\n---\nprovider: openai",
		"secret-value",
		"[secret-value",
	} {
		_, err := loadTest(t, body, "OPENAI_MODEL=env-model", "TIMBREX_STRICT_TAGS=false")
		if err == nil || strings.Contains(err.Error(), "secret-value") {
			t.Errorf("invalid config was accepted or leaked a value: %v", err)
		}
	}
	if _, err := loadTest(t, "", "TIMBREX_STRICT_TAGS=secret-value"); err == nil || strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("invalid environment: %v", err)
	}
}

func TestMalformedProviderSettingsDoNotLeakCredentials(t *testing.T) {
	const secret = "FAKE-SECRET-DO-NOT-LOG"
	for _, id := range []string{"stub", "openai", "elevenlabs", "minimax", "azure", "qwen", "gemini", "fish"} {
		for _, tt := range []struct {
			name, settings string
		}{
			{"credential mapping", "{api_key: {" + secret + ": null}}"},
			{"nested credential mapping", "{api_key: {nested: {" + secret + ": null}}}"},
			{"credential mapping value", "{api_key: {value: " + secret + "}}"},
			{"credential list", "{api_key: [" + secret + ", null]}"},
			{"null credential", "{api_key: null, voice: " + secret + "}"},
			{"settings list", "[{api_key: " + secret + "}]"},
			{"unknown field", "{api_key: " + secret + ", unknown: " + secret + "}"},
			{"nested unknown field", "{unknown: {api_key: " + secret + "}}"},
		} {
			t.Run(id+"/"+tt.name, func(t *testing.T) {
				body := "providers:\n  " + id + ": " + tt.settings + "\n"
				_, err := loadTest(t, body)
				if err == nil || strings.Contains(err.Error(), secret) {
					t.Fatalf("invalid config was accepted or leaked a credential: %v", err)
				}
			})
		}
	}
}

func TestSelectedProviderRequirementsAndAzureEndpoint(t *testing.T) {
	c, err := loadTest(t, "providers:\n  azure: {api_key: test-key, region: eastus}\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Resolve("stub", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Resolve("openai", "", ""); err == nil {
		t.Fatal("accepted missing selected credentials")
	}
	p, err := c.Resolve("azure", "", "")
	if err != nil || p.Endpoint != "https://eastus.tts.speech.microsoft.com/cognitiveservices/v1" {
		t.Fatalf("Azure region endpoint: %v", err)
	}
	c, err = loadTest(t, "providers:\n  azure: {api_key: test-key, region: eastus, endpoint: 'https://example.com/speech'}\n")
	if err != nil {
		t.Fatal(err)
	}
	p, err = c.Resolve("azure", "", "")
	if err != nil || p.Endpoint != "https://example.com/speech" {
		t.Fatalf("explicit Azure endpoint: %v", err)
	}
	for _, id := range []string{"azure", "qwen"} {
		c, err = loadTest(t, "providers:\n  "+id+": {api_key: test-key}\n")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Resolve(id, "", ""); err == nil {
			t.Fatalf("accepted missing %s endpoint", id)
		}
	}
}

func TestRedactedOutputPreservesCredentials(t *testing.T) {
	for _, id := range []string{"stub", "openai", "elevenlabs", "minimax", "azure", "qwen", "gemini", "fish"} {
		t.Run(id, func(t *testing.T) {
			c, err := loadTest(t, "providers:\n  "+id+": {api_key: FAKE-SECRET-DO-NOT-LOG}\n")
			if err != nil {
				t.Fatal(err)
			}
			data, err := c.RedactedYAML()
			if err != nil || strings.Contains(string(data), "FAKE-SECRET-DO-NOT-LOG") || !strings.Contains(string(data), "api_key: REDACTED") {
				t.Fatal("configuration output must redact credentials")
			}
			if c.Providers[id].APIKey != "FAKE-SECRET-DO-NOT-LOG" {
				t.Fatal("displaying configuration changed its credentials")
			}
		})
	}
}

func TestExampleConfiguration(t *testing.T) {
	if _, err := load("../../examples/timbrex.yaml", func() []string { return nil }); err != nil {
		t.Fatal(err)
	}
}
