package cli

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
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
	for _, name := range []string{"OPENAI_MODEL", "ELEVENLABS_MODEL", "MINIMAX_MODEL", "QWEN_MODEL", "GEMINI_MODEL", "FISH_MODEL", "OPENAI_VOICE", "ELEVENLABS_VOICE_ID", "MINIMAX_VOICE_ID", "AZURE_SPEECH_VOICE", "QWEN_VOICE", "GEMINI_VOICE", "FISH_VOICE_ID"} {
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
					if vendor.id != "qwen" || r.URL.Path != fmt.Sprintf("/audio-%d.pcm", downloads) || r.Header.Get("Authorization") != "" {
						t.Errorf("unexpected audio download: %s %s", r.Method, r.URL.Path)
					}
					_, _ = w.Write(contractPCM(downloads))
					return
				}
				calls++
				wantPath := vendor.path
				if vendor.id == "elevenlabs" {
					wantPath += "en-US-FirstNeural"
					if calls == 3 {
						wantPath = vendor.path + "en-US-SecondNeural"
					}
				}
				if r.Method != http.MethodPost || r.URL.Path != wantPath {
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
					_, _ = fmt.Fprintf(w, `{"base_resp":{"status_code":0},"data":{"status":2,"audio":%q}}`, fmt.Sprintf("%x", contractPCM(calls)))
				case "qwen":
					_ = json.NewEncoder(w).Encode(map[string]any{"output": map[string]any{"audio": map[string]string{"url": fmt.Sprintf("%s/audio-%d.pcm", serverURL, calls)}, "finish_reason": "stop"}})
				default:
					w.Header().Set("Content-Type", "application/octet-stream")
					_, _ = w.Write(contractPCM(calls))
				}
			}))
			defer srv.Close()
			serverURL = srv.URL
			t.Setenv(vendor.keyEnv, "test-key")
			t.Setenv(vendor.endpointEnv, srv.URL)
			dir := t.TempDir()
			path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "scene.wav")
			src := fmt.Sprintf("---\nprovider: %s\ncast:\n  a: {voices: {%s: en-US-FirstNeural}}\n  b: {voices: {%s: en-US-SecondNeural}}\n---\na: [happy] One [pause 10ms] [speed 1.2|Two]\nb: Three\n", vendor.id, vendor.id, vendor.id)
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
			wantPCM := append(contractPCM(1), make([]byte, 480)...)
			wantPCM = append(wantPCM, contractPCM(2)...)
			wantPCM = append(wantPCM, contractPCM(3)...)
			if !bytes.Equal(wav[44:], wantPCM) {
				t.Fatalf("PCM order or pause changed: got %x, want %x", wav[44:], wantPCM)
			}
			if strings.TrimSpace(stdout.String()) != out {
				t.Fatalf("stdout = %q", stdout.String())
			}
		})
	}
}

// Distinct samples make dropped, repeated, and reordered responses observable.
func contractPCM(call int) []byte { return []byte{byte(call), 0, byte(call + 10), 0} }

func assertRequest(t *testing.T, id string, r *http.Request, body []byte, call int, voice string) {
	t.Helper()
	if call < 1 || call > 3 {
		t.Errorf("unexpected request %d: %s", call, body)
		return
	}
	text := []string{" One ", "Two", "Three"}[call-1]
	if id == "azure" {
		if r.Header.Get("Content-Type") != "application/ssml+xml" || r.Header.Get("X-Microsoft-OutputFormat") != "raw-24khz-16bit-mono-pcm" {
			t.Error("wrong Azure headers")
		}
		if call == 2 {
			text = `<prosody rate="1.2">` + text + `</prosody>`
		}
		if call < 3 {
			text = `<mstts:express-as style="cheerful">` + text + `</mstts:express-as>`
		}
		want := `<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="https://www.w3.org/2001/mstts" xml:lang="en-US"><voice name="` + voice + `">` + text + `</voice></speak>`
		if string(body) != want {
			t.Errorf("SSML = %s, want %s", body, want)
		}
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		t.Error("wrong JSON Content-Type")
	}
	switch id {
	case "openai":
		want := struct {
			Model        string  `json:"model"`
			Input        string  `json:"input"`
			Voice        string  `json:"voice"`
			Format       string  `json:"response_format"`
			Instructions string  `json:"instructions,omitempty"`
			Speed        float64 `json:"speed,omitempty"`
		}{Model: "gpt-4o-mini-tts", Input: text, Voice: voice, Format: "pcm"}
		if call < 3 {
			want.Instructions = "happy"
		}
		if call == 2 {
			want.Speed = 1.2
		}
		assertJSONRequest(t, body, want)
	case "elevenlabs":
		type settings struct {
			Speed float64 `json:"speed"`
		}
		want := struct {
			Text     string    `json:"text"`
			Model    string    `json:"model_id"`
			Settings *settings `json:"voice_settings,omitempty"`
		}{Text: text, Model: "eleven_v3"}
		if !strings.HasSuffix(r.URL.Path, voice) || r.URL.RawQuery != "output_format=pcm_24000" {
			t.Errorf("URL = %s", r.URL)
		}
		if call < 3 {
			want.Text = "[happy]" + text
		}
		if call == 2 {
			want.Settings = &settings{Speed: 1.2}
		}
		assertJSONRequest(t, body, want)
	case "minimax":
		type voiceSettings struct {
			ID      string  `json:"voice_id"`
			Speed   float64 `json:"speed"`
			Volume  float64 `json:"vol"`
			Pitch   int     `json:"pitch"`
			Emotion string  `json:"emotion,omitempty"`
		}
		type audioSettings struct {
			Rate    int    `json:"sample_rate"`
			Format  string `json:"format"`
			Channel int    `json:"channel"`
		}
		want := struct {
			Model  string        `json:"model"`
			Text   string        `json:"text"`
			Stream bool          `json:"stream"`
			Output string        `json:"output_format"`
			Voice  voiceSettings `json:"voice_setting"`
			Audio  audioSettings `json:"audio_setting"`
		}{Model: "speech-2.8-hd", Text: text, Output: "hex", Voice: voiceSettings{ID: voice, Speed: 1, Volume: 1}, Audio: audioSettings{Rate: 24000, Format: "pcm", Channel: 1}}
		if call < 3 {
			want.Voice.Emotion = "happy"
		}
		if call == 2 {
			want.Voice.Speed = 1.2
		}
		assertJSONRequest(t, body, want)
	case "qwen":
		type input struct {
			Text        string  `json:"text"`
			Voice       string  `json:"voice"`
			Format      string  `json:"format"`
			SampleRate  int     `json:"sample_rate"`
			Instruction string  `json:"instruction,omitempty"`
			Rate        float64 `json:"rate,omitempty"`
		}
		want := struct {
			Model string `json:"model"`
			Input input  `json:"input"`
		}{Model: "qwen-audio-3.1-tts-flash", Input: input{Text: text, Voice: voice, Format: "pcm", SampleRate: 24000}}
		if call < 3 {
			want.Input.Instruction = "happy"
		}
		if call == 2 {
			want.Input.Rate = 1.2
		}
		assertJSONRequest(t, body, want)
	default:
		t.Errorf("no request contract for %q", id)
	}
}

func assertJSONRequest(t *testing.T, body []byte, want any) {
	t.Helper()
	expected, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	// Compare complete JSON values, including absent versus null/empty fields,
	// without depending on object key order or unchecked type assertions.
	var gotJSON, wantJSON any
	if err := json.Unmarshal(body, &gotJSON); err != nil {
		t.Error(err)
		return
	}
	if err := json.Unmarshal(expected, &wantJSON); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotJSON, wantJSON) {
		t.Errorf("request = %s, want %s", body, expected)
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
