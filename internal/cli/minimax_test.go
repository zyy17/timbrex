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
)

func TestMiniMaxLanguageAndFluentCLIContract(t *testing.T) {
	for _, concurrency := range []int{1, 3} {
		for _, late := range []string{"", "[lang en-foo|Invalid]", "[lang auto-US|Invalid]", "[style unknown|Invalid]", "[happy][style fluent]Invalid", "[style fluent][whisper]Invalid"} {
			t.Run(fmt.Sprintf("concurrency=%d/late=%s", concurrency, late), func(t *testing.T) {
				clearProviderDefaults(t)
				t.Setenv("MINIMAX_API_KEY", "FAKE-MINIMAX-KEY")
				var calls atomic.Int32
				want := map[string]struct {
					language, emotion string
					sample            int
				}{
					"One": {"English", "fluent", 1}, "Two": {"Turkish", "fluent", 2},
					"Three": {"auto", "fluent", 3}, "Four": {"Turkish", "fluent", 4},
					"Five": {"English", "fluent", 5}, "Six\n": {"English", "", 6},
					"Seven": {"Nynorsk", "", 7}, "Eight": {"Norwegian", "", 8},
				}
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != http.MethodPost || r.URL.Path != "/t2a_v2" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != "Bearer FAKE-MINIMAX-KEY" || r.Header.Get("Content-Type") != "application/json" {
						t.Errorf("wrong request %s %s", r.Method, r.URL)
					}
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
						return
					}
					var got struct{ Text string }
					if err := json.Unmarshal(body, &got); err != nil {
						t.Error(err)
						return
					}
					expected, ok := want[got.Text]
					if !ok {
						t.Errorf("unexpected text %q", got.Text)
						return
					}
					emotion := ""
					if expected.emotion != "" {
						emotion = `,"emotion":"fluent"`
					}
					assertJSONRequest(t, body, json.RawMessage(fmt.Sprintf(`{"model":"speech-2.6-hd","text":%q,"stream":false,"output_format":"hex","language_boost":%q,"voice_setting":{"voice_id":"custom-voice","speed":1,"vol":1,"pitch":0%s},"audio_setting":{"sample_rate":24000,"format":"pcm","channel":1}}`, got.Text, expected.language, emotion)))
					_, _ = fmt.Fprintf(w, `{"base_resp":{"status_code":0},"data":{"status":2,"audio":"%x"}}`, contractPCM(expected.sample))
				}))
				defer srv.Close()
				t.Setenv("MINIMAX_BASE_URL", srv.URL)
				dir := t.TempDir()
				path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "out.wav")
				src := "---\nprovider: minimax\nmodel: speech-2.6-hd\nvoice: custom-voice\nlang: en-US\n---\n[style fluent|One[lang tr-TR|Two[lang auto|Three]Four]Five]Six\n[lang nn-NO|Seven]\n[lang no-NO|Eight]\n" + late
				if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
					t.Fatal(err)
				}
				err := run(context.Background(), []string{"synth", path, "-o", out, "--concurrency", fmt.Sprint(concurrency)}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
				data, readErr := os.ReadFile(out)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if late != "" {
					if err == nil || calls.Load() != 0 || string(data) != "KEEP" {
						t.Fatalf("error=%v calls=%d output=%q", err, calls.Load(), data)
					}
					return
				}
				if err != nil || calls.Load() != 8 {
					t.Fatalf("error=%v calls=%d", err, calls.Load())
				}
				var pcm []byte
				for i := 1; i <= 8; i++ {
					pcm = append(pcm, contractPCM(i)...)
				}
				if len(data) != 44+len(pcm) || !bytes.Equal(data[44:], pcm) {
					t.Fatalf("incorrect WAV order: %x", data)
				}
			})
		}
	}
}
