package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGeminiModelsFullHTTPContract(t *testing.T) {
	for _, model := range []string{"gemini-3.8-flash-tts", "gemini-3.8-flash-lite-tts"} {
		for _, workers := range []int{1, 4} {
			t.Run(fmt.Sprintf("%s/concurrency_%d", model, workers), func(t *testing.T) {
				t.Setenv("GEMINI_API_KEY", "FAKE-GEMINI-KEY")
				t.Setenv("GEMINI_MODEL", "")
				t.Setenv("GEMINI_VOICE", "")
				const eventSource = "[laugh][sigh][cough][breath][gasp][exhale][chuckle][giggle][cackle][snort][cry][sob][whimper][pant][groan][grunt][moan][sneeze][yawn][hiss]"
				const eventText = "<laugh><sigh><cough><breath><gasp><exhales><chuckle><giggle><cackle><snort><cry><sob><whimper><pant><groan><grunt><moan><sneeze><yawn><hiss>"
				texts := []string{" One \"hi\" & \\ [literal] " + eventText, "Two <throat-clearing>", "Three "}
				styles := []string{`{"style":"happy"}`, `{"style":"happy. whisper"}`, `{}`}
				voices := []string{"Kore", "Kore", "Puck"}
				var calls atomic.Int32
				var seen [3]atomic.Int32
				lastReturned := make(chan struct{})
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					call := calls.Add(1)
					if r.Method != "POST" || r.URL.Path != "/models/"+model+":generateContent" || r.URL.RawQuery != "" || r.Header.Get("x-goog-api-key") != "FAKE-GEMINI-KEY" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Authorization") != "" {
						t.Errorf("incorrect request method/path/auth: %s %s", r.Method, r.URL)
					}
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
						return
					}
					var req struct {
						Contents []struct{ Parts []struct{ Text string } }
					}
					if err := json.Unmarshal(body, &req); err != nil || len(req.Contents) != 1 || len(req.Contents[0].Parts) != 1 {
						t.Errorf("invalid content: %s, %v", body, err)
						return
					}
					index := -1
					for i, text := range texts {
						if req.Contents[0].Parts[0].Text == text {
							index = i
						}
					}
					if index < 0 {
						t.Errorf("unexpected transcript: %q", req.Contents[0].Parts[0].Text)
						return
					}
					if seen[index].Add(1) != 1 || (workers == 1 && int(call) != index+1) {
						t.Errorf("duplicate or out-of-order serial request: %d", index)
					}
					assertJSONRequest(t, body, json.RawMessage(fmt.Sprintf(`{"contents":[{"role":"user","parts":[{"text":%q,"speechMetadata":%s}]}],"generationConfig":{"responseModalities":["AUDIO"],"responseFormat":{"audio":{"mimeType":"AUDIO_WAV","sampleRate":24000,"delivery":"INLINE"}},"speechConfig":{"voiceConfig":{"prebuiltVoiceConfig":{"voiceName":%q}},"languageCode":"en-GB"}}}`, texts[index], styles[index], voices[index])))
					// Force the first concurrent segment to finish after the last.
					if workers > 1 && index == 0 {
						select {
						case <-lastReturned:
						case <-time.After(3 * time.Second):
							t.Error("concurrent requests did not overlap")
						}
					}
					concurrencyReply(w, "gemini", "", index+1)
					if workers > 1 && index == 2 && seen[index].Load() == 1 {
						close(lastReturned)
					}
				}))
				defer srv.Close()
				t.Setenv("GEMINI_BASE_URL", srv.URL)
				dir := t.TempDir()
				path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "out.wav")
				src := "---\nprovider: gemini\nlang: en-GB\nstrict_tags: true\ncast:\n  a: {voices: {gemini: Kore}}\n  b: {voices: {gemini: Puck}}\n---\na: [pause 10ms] [happy] One \"hi\" & \\\\ \\[literal\\] " + eventSource + "[pause 10ms][whisper|Two [clear-throat]]\nb: Three [pause 10ms]\n"
				if err := os.WriteFile(path, []byte(src), 0600); err != nil {
					t.Fatal(err)
				}
				var stdout bytes.Buffer
				if err := run(context.Background(), []string{"synth", path, "--model", model, "--concurrency", fmt.Sprint(workers), "-o", out}, strings.NewReader(""), &stdout, &bytes.Buffer{}); err != nil {
					t.Fatal(err)
				}
				if calls.Load() != 3 || strings.TrimSpace(stdout.String()) != out {
					t.Fatalf("calls=%d stdout=%q", calls.Load(), stdout.String())
				}
				wantPCM := append(make([]byte, 480), concurrencyPCM(1)...)
				wantPCM = append(wantPCM, make([]byte, 480)...)
				wantPCM = append(wantPCM, concurrencyPCM(2)...)
				wantPCM = append(wantPCM, concurrencyPCM(3)...)
				wantPCM = append(wantPCM, make([]byte, 480)...)
				wav, err := os.ReadFile(out)
				if err != nil || !bytes.Equal(wav, concurrencyWAV(wantPCM)) {
					t.Fatalf("incorrect WAV/PCM order or local pauses: %x, %v", wav, err)
				}
			})
		}
	}
}

func TestGeminiModelsLateInvalidZeroNetwork(t *testing.T) {
	for _, model := range []string{"gemini-3.8-flash-tts", "gemini-3.8-flash-lite-tts"} {
		for _, workers := range []int{1, 3} {
			for _, tc := range []struct{ name, invalid, want string }{
				{"speed", "[speed 1.2] Invalid", "speed"},
				{"pitch", "[pitch +2st] Invalid", "pitch"},
				{"volume", "[volume -3dB] Invalid", "volume"},
				{"unsupported event", "Invalid [burp]", "unsupported event"},
				{"event note", "Invalid [chuckle quietly]", "arguments"},
				{"event number", "Invalid [exhale 0.5]", "arguments"},
				{"event named argument", "Invalid [clear-throat intensity=0.5]", "arguments"},
				{"literal tag", "Invalid <chuckle>", "literal"},
				{"escaped literal tag", `Invalid \<chuckle\>`, "literal"},
				{"text bound", strings.Repeat("中", 4097), "4096"},
				{"style bound", "[act \"" + strings.Repeat("a", 4096) + "\"|Invalid]", "4096"},
				{"language", "[lang zh-CN|Invalid]", "language"},
				{"Thai", "[lang th-TH|Invalid]", "language"},
			} {
				if tc.name == "Thai" && model == "gemini-3.8-flash-tts" {
					continue
				}
				t.Run(fmt.Sprintf("%s/concurrency_%d/%s", model, workers, tc.name), func(t *testing.T) {
					t.Setenv("GEMINI_API_KEY", "FAKE-GEMINI-KEY")
					t.Setenv("GEMINI_VOICE", "")
					var calls atomic.Int32
					srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						calls.Add(1)
						concurrencyReply(w, "gemini", "", 1)
					}))
					defer srv.Close()
					t.Setenv("GEMINI_BASE_URL", srv.URL)
					dir := t.TempDir()
					path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "out.wav")
					src := "---\nprovider: gemini\nstrict_tags: true\n---\nValid [chuckle]\n\nValid [pause 10ms] again\n\n" + tc.invalid + "\n"
					if err := os.WriteFile(path, []byte(src), 0600); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(out, []byte("KEEP"), 0600); err != nil {
						t.Fatal(err)
					}
					err := run(context.Background(), []string{"synth", path, "--model", model, "--concurrency", fmt.Sprint(workers), "-o", out}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
					if err == nil || !strings.Contains(err.Error(), tc.want) || calls.Load() != 0 {
						t.Fatalf("err=%v, calls=%d", err, calls.Load())
					}
					data, readErr := os.ReadFile(out)
					if readErr != nil || string(data) != "KEEP" {
						t.Fatalf("existing output changed: %q, %v", data, readErr)
					}
				})
			}
		}
	}
}
