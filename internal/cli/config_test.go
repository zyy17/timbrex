package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigurationCommandRedactsCredentials(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "secret-value")
	var out bytes.Buffer
	if err := run(context.Background(), []string{"config"}, nil, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "secret-value") || !strings.Contains(out.String(), "api_key: REDACTED") {
		t.Fatal("config command must redact API keys")
	}
}

func TestHelpAndVersionIgnoreBrokenConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(path, []byte("unknown: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"version"}, {"--version"}, {"--help"}, {"synth", "--help"}} {
		args = append([]string{"--config", path}, args...)
		if err := run(context.Background(), args, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Errorf("%v: %v", args, err)
		}
	}
	if err := run(context.Background(), []string{"--config", path, "config"}, nil, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("config command accepted invalid configuration")
	}
}

func TestStrictConfigurationPrecedence(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("strict_tags: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, front, env string
		flags            []string
		wantError        bool
	}{
		{name: "file", wantError: true},
		{name: "environment", env: "false"},
		{name: "front matter", front: "---\nstrict_tags: false\n---\n", env: "true"},
		{name: "flag", flags: []string{"--no-strict-tags"}},
		{name: "explicit false flag", flags: []string{"--strict-tags=false"}},
		{name: "negative false flag", front: "---\nstrict_tags: false\n---\n", flags: []string{"--no-strict-tags=false"}, wantError: true},
		{name: "strict flag", front: "---\nstrict_tags: false\n---\n", flags: []string{"--strict-tags"}, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TIMBREX_STRICT_TAGS", tt.env)
			path := filepath.Join(dir, "script.tbx")
			if err := os.WriteFile(path, []byte(tt.front+"[thoughtfully|Hello]"), 0o600); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"check", path, "--config", configPath}, tt.flags...)
			err := run(context.Background(), args, nil, &bytes.Buffer{}, &bytes.Buffer{})
			if (err != nil) != tt.wantError {
				t.Fatalf("error=%v, wantError=%t", err, tt.wantError)
			}
		})
	}
}

func TestFileSettingsReachProviderAndFlagsOverrideScript(t *testing.T) {
	for _, tt := range []struct {
		name, front, envModel, envProvider, wantModel, wantVoice string
		flags                                                    []string
	}{
		{name: "file", wantModel: "file-model", wantVoice: "file-voice"},
		{name: "environment", envModel: "env-model", wantModel: "env-model", wantVoice: "file-voice"},
		{name: "script", envModel: "env-model", envProvider: "stub", front: "provider: openai\nmodel: script-model\nvoice: script-voice\n", wantModel: "script-model", wantVoice: "script-voice"},
		{name: "flags", front: "provider: stub\nmodel: script-model\nvoice: script-voice\n", flags: []string{"--provider", "openai", "--model", "flag-model", "--voice", "flag-voice"}, wantModel: "flag-model", wantVoice: "flag-voice"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OPENAI_MODEL", tt.envModel)
			t.Setenv("TIMBREX_PROVIDER", tt.envProvider)
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var request struct{ Model, Voice string }
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				if r.Header.Get("Authorization") != "Bearer file-key" || r.URL.Path != "/v1/audio/speech" || request.Model != tt.wantModel || request.Voice != tt.wantVoice {
					t.Error("resolved settings did not reach the provider request")
				}
				w.Header().Set("Content-Type", "audio/pcm")
				_, _ = w.Write([]byte{1, 0, 2, 0})
			}))
			defer srv.Close()
			dir := t.TempDir()
			configPath, path := filepath.Join(dir, "config.yaml"), filepath.Join(dir, "script.tbx")
			body := fmt.Sprintf("provider: openai\nproviders:\n  openai:\n    api_key: file-key\n    model: file-model\n    voice: file-voice\n    endpoint: %s/v1\n", srv.URL)
			if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("---\n"+tt.front+"---\nHello"), 0o600); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"synth", path, "--config", configPath}, tt.flags...)
			if err := run(context.Background(), args, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("requests=%d, want 1", calls)
			}
		})
	}
}
