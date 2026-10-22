package cli

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type vendorTest struct{ id, keyEnv, endpointEnv, path, authHeader, authValue string }

var vendors = []vendorTest{
	{"openai", "OPENAI_API_KEY", "OPENAI_BASE_URL", "/audio/speech", "Authorization", "Bearer test-key"},
	{"elevenlabs", "ELEVENLABS_API_KEY", "ELEVENLABS_BASE_URL", "/text-to-speech/", "xi-api-key", "test-key"},
	{"minimax", "MINIMAX_API_KEY", "MINIMAX_BASE_URL", "/t2a_v2", "Authorization", "Bearer test-key"},
	{"azure", "AZURE_SPEECH_KEY", "AZURE_SPEECH_ENDPOINT", "/", "Ocp-Apim-Subscription-Key", "test-key"},
	{"qwen", "DASHSCOPE_API_KEY", "QWEN_ENDPOINT", "/", "Authorization", "Bearer test-key"},
}

func clearProviderDefaults(t *testing.T) {
	t.Helper()
	for _, name := range []string{"OPENAI_MODEL", "ELEVENLABS_MODEL", "MINIMAX_MODEL", "QWEN_MODEL", "OPENAI_VOICE", "ELEVENLABS_VOICE_ID", "MINIMAX_VOICE_ID", "AZURE_SPEECH_VOICE", "QWEN_VOICE"} {
		t.Setenv(name, "")
	}
}

func TestRealProviderContracts(t *testing.T) {
	for _, vendor := range vendors {
		t.Run(vendor.id, func(t *testing.T) {
			clearProviderDefaults(t)
			var calls, downloads int
			var serverURL string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					downloads++
					w.Header().Set("Content-Type", "application/octet-stream")
					if vendor.id != "qwen" || r.URL.Path != "/audio.pcm" || r.Header.Get("Authorization") != "" {
						t.Errorf("unexpected audio download: %s %s", r.Method, r.URL.Path)
					}
					_, _ = w.Write([]byte{1, 0, 2, 0})
					return
				}
				calls++
				if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, vendor.path) {
					t.Errorf("method/path = %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get(vendor.authHeader) != vendor.authValue {
					t.Error("wrong authentication header")
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					return
				}
				voice := "en-US-FirstNeural"
				if calls == 3 {
					voice = "en-US-SecondNeural"
				}
				assertRequest(t, vendor.id, r, body, calls, voice)
				switch vendor.id {
				case "minimax":
					_, _ = io.WriteString(w, `{"base_resp":{"status_code":0},"data":{"status":2,"audio":"01000200"}}`)
				case "qwen":
					_ = json.NewEncoder(w).Encode(map[string]any{"output": map[string]any{"audio": map[string]string{"url": serverURL + "/audio.pcm"}, "finish_reason": "stop"}})
				default:
					w.Header().Set("Content-Type", "application/octet-stream")
					_, _ = w.Write([]byte{1, 0, 2, 0})
				}
			}))
			defer srv.Close()
			serverURL = srv.URL
			t.Setenv(vendor.keyEnv, "test-key")
			t.Setenv(vendor.endpointEnv, srv.URL)
			dir := t.TempDir()
			path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "scene.wav")
			src := fmt.Sprintf("---\nprovider: %s\nlang: en-US\ncast:\n  a: {voices: {%s: en-US-FirstNeural}}\n  b: {voices: {%s: en-US-SecondNeural}}\n---\na: [happy] One [pause 10ms] [speed 1.2|Two]\nb: Three\n", vendor.id, vendor.id, vendor.id)
			if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			if err := run(context.Background(), []string{"synth", path, "-o", out}, strings.NewReader(""), &stdout, &stderr); err != nil {
				t.Fatalf("%v; %s", err, stderr.String())
			}
			if calls != 3 || (vendor.id == "qwen" && downloads != 3) {
				t.Fatalf("calls=%d downloads=%d", calls, downloads)
			}
			wav, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if len(wav) != 44+12+480 || string(wav[:4]) != "RIFF" || binary.LittleEndian.Uint32(wav[24:28]) != 24000 || binary.LittleEndian.Uint32(wav[40:44]) != 492 {
				t.Fatalf("bad WAV: %d bytes", len(wav))
			}
			if !bytes.Equal(wav[48:528], make([]byte, 480)) {
				t.Fatal("pause is not exact PCM silence")
			}
			if strings.TrimSpace(stdout.String()) != out {
				t.Fatalf("stdout = %q", stdout.String())
			}
		})
	}
}

func assertRequest(t *testing.T, id string, r *http.Request, body []byte, call int, voice string) {
	t.Helper()
	if id == "azure" {
		if r.Header.Get("Content-Type") != "application/ssml+xml" || r.Header.Get("X-Microsoft-OutputFormat") != "raw-24khz-16bit-mono-pcm" {
			t.Error("wrong Azure headers")
		}
		var doc struct {
			Voice struct {
				Name string `xml:"name,attr"`
			} `xml:"voice"`
		}
		if err := xml.Unmarshal(body, &doc); err != nil {
			t.Error(err)
		}
		if doc.Voice.Name != voice {
			t.Errorf("voice = %q", doc.Voice.Name)
		}
		if call < 3 && !bytes.Contains(body, []byte(`style="cheerful"`)) {
			t.Errorf("missing happy style: %s", body)
		}
		if call == 2 && !bytes.Contains(body, []byte(`rate="1.2"`)) {
			t.Errorf("missing speed: %s", body)
		}
		if call == 3 && bytes.Contains(body, []byte(`express-as style`)) {
			t.Error("emotion leaked to next turn")
		}
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		t.Error("wrong JSON Content-Type")
	}
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		t.Error(err)
		return
	}
	switch id {
	case "openai":
		if obj["model"] != "gpt-4o-mini-tts" || obj["voice"] != voice || obj["response_format"] != "pcm" {
			t.Errorf("request = %s", body)
		}
		if call < 3 && !strings.Contains(fmt.Sprint(obj["instructions"]), "happy") {
			t.Errorf("missing instructions: %s", body)
		}
		if call == 2 && obj["speed"] != 1.2 {
			t.Error("missing speed")
		}
	case "elevenlabs":
		if obj["model_id"] != "eleven_v3" || !strings.HasSuffix(r.URL.Path, voice) || r.URL.Query().Get("output_format") != "pcm_24000" {
			t.Errorf("request = %s %s", r.URL, body)
		}
		if call < 3 && !strings.HasPrefix(fmt.Sprint(obj["text"]), "[happy]") {
			t.Error("missing audio tag")
		}
		if call == 2 && obj["voice_settings"].(map[string]any)["speed"] != 1.2 {
			t.Error("missing speed")
		}
	case "minimax":
		v := obj["voice_setting"].(map[string]any)
		a := obj["audio_setting"].(map[string]any)
		if obj["model"] != "speech-2.8-hd" || obj["stream"] != false || obj["output_format"] != "hex" || v["voice_id"] != voice || a["format"] != "pcm" || a["sample_rate"] != float64(24000) || a["channel"] != float64(1) {
			t.Errorf("request = %s", body)
		}
		if call < 3 && v["emotion"] != "happy" {
			t.Error("missing emotion")
		}
		if call == 2 && v["speed"] != 1.2 {
			t.Error("missing speed")
		}
	case "qwen":
		in := obj["input"].(map[string]any)
		if obj["model"] != "qwen-audio-3.1-tts-flash" || in["voice"] != voice || in["format"] != "pcm" || in["sample_rate"] != float64(24000) {
			t.Errorf("request = %s", body)
		}
		if call < 3 && !strings.Contains(fmt.Sprint(in["instruction"]), "happy") {
			t.Error("missing instruction")
		}
		if call == 2 && in["rate"] != 1.2 {
			t.Error("missing speed")
		}
	}
}

func TestProvidersRejectUnsupportedBeforeRequests(t *testing.T) {
	for _, vendor := range vendors {
		t.Run(vendor.id, func(t *testing.T) {
			clearProviderDefaults(t)
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
			defer srv.Close()
			t.Setenv(vendor.keyEnv, "test-key")
			t.Setenv(vendor.endpointEnv, srv.URL)
			dir := t.TempDir()
			path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "scene.wav")
			if err := os.WriteFile(path, []byte("Hello\n\nLater [sfx missing.wav]"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			err := run(context.Background(), []string{"synth", path, "--provider", vendor.id, "--voice", "en-US-JennyNeural"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), "sfx") || calls != 0 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != "KEEP" {
				t.Fatalf("output = %q %v", data, err)
			}
		})
	}
}

func TestSynthCannotOverwriteSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.wav")
	if err := os.WriteFile(path, []byte("Hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := run(context.Background(), []string{"synth", path}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "source script") {
		t.Fatalf("err = %v", err)
	}
}

func TestSynthConfigurationPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name, front string
		flags       []string
		model       string
		voices      []string
	}{
		{name: "environment", model: "env-model", voices: []string{"env-voice", "env-voice"}},
		{name: "front", front: "model: front-model\nvoice: front-voice\n", model: "front-model", voices: []string{"front-voice", "front-voice"}},
		{name: "flags", front: "model: front-model\nvoice: front-voice\n", flags: []string{"--model", "flag-model", "--voice", "flag-voice"}, model: "flag-model", voices: []string{"flag-voice", "flag-voice"}},
		{name: "cast", front: "model: front-model\nvoice: front-voice\ncast:\n  a: {voices: {openai: cast-voice}}\n", flags: []string{"--voice", "flag-voice"}, model: "front-model", voices: []string{"cast-voice", "flag-voice"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearProviderDefaults(t)
			t.Setenv("OPENAI_API_KEY", "test-key")
			t.Setenv("OPENAI_MODEL", "env-model")
			t.Setenv("OPENAI_VOICE", "env-voice")
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var obj struct{ Model, Voice string }
				if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
					t.Error(err)
				}
				if calls >= len(tc.voices) || obj.Model != tc.model || obj.Voice != tc.voices[calls] {
					t.Errorf("call %d: %+v", calls, obj)
				}
				calls++
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write([]byte{0, 0})
			}))
			defer srv.Close()
			t.Setenv("OPENAI_BASE_URL", srv.URL)
			path := filepath.Join(t.TempDir(), "script.tbx")
			src := "---\nprovider: missing\n" + tc.front + "---\na: Hello\n\nNarration\n"
			if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"synth", path, "--provider", "openai"}, tc.flags...)
			if err := run(context.Background(), args, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatalf("calls = %d", calls)
			}
		})
	}
}

func TestProvidersRejectInvalidNumbersBeforeRequests(t *testing.T) {
	for _, vendor := range vendors {
		t.Run(vendor.id, func(t *testing.T) {
			clearProviderDefaults(t)
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
			defer srv.Close()
			t.Setenv(vendor.keyEnv, "test-key")
			t.Setenv(vendor.endpointEnv, srv.URL)
			path := filepath.Join(t.TempDir(), "script.tbx")
			for _, tag := range []string{`[speed "1.2"|Hi]`, `[speed 1e0|Hi]`, `[pause 1m]`, `[pause 1s250ms]`} {
				if err := os.WriteFile(path, []byte("Hello\n\n"+tag), 0o600); err != nil {
					t.Fatal(err)
				}
				err := run(context.Background(), []string{"synth", path, "--provider", vendor.id, "--voice", "en-US-JennyNeural"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
				if err == nil || calls != 0 {
					t.Fatalf("%s: error=%v, requests=%d", tag, err, calls)
				}
			}
		})
	}
}
