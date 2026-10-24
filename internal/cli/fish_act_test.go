package cli

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestFishS2ActCLIContract(t *testing.T) {
	for _, model := range []string{"s2.1-pro", "s2.1-pro-free", "s2-pro"} {
		for _, concurrency := range []int{1, 3} {
			t.Run(fmt.Sprintf("%s/%d", model, concurrency), func(t *testing.T) {
				t.Setenv("FISH_API_KEY", "FAKE-FISH-KEY")
				t.Setenv("FISH_MODEL", "")
				t.Setenv("FISH_VOICE_ID", "test-reference")
				want := []string{"[scene direction] A", "[outer direction] B", "[轻轻地] C", "[outer direction] D", "[scene direction] E", "[holding back tears] F", "[scene direction] G", `[say "yes" quietly] H`, "[scene direction] I", "J"}
				var mu sync.Mutex
				seen := make(map[string]int)
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "POST" || r.URL.Path != "/tts" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != "Bearer FAKE-FISH-KEY" || r.Header.Get("model") != model || r.Header.Get("Content-Type") != "application/json" {
						t.Errorf("wrong method/path/headers: %s %s", r.Method, r.URL.Path)
					}
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
						return
					}
					var obj struct{ Text string }
					if err := json.Unmarshal(body, &obj); err != nil {
						t.Error(err)
						return
					}
					index := -1
					for i, text := range want {
						if obj.Text == text {
							index = i
							break
						}
					}
					if index < 0 {
						t.Errorf("unexpected request text %q", obj.Text)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					assertJSONRequest(t, body, json.RawMessage(fmt.Sprintf(`{"text":%q,"reference_id":"test-reference","format":"wav","sample_rate":24000,"normalize":false,"prosody":{"speed":1,"volume":0,"normalize_loudness":true}}`, want[index])))
					mu.Lock()
					seen[obj.Text]++
					mu.Unlock()
					wav, err := hex.DecodeString("524946462800000057415645666d74201000000001000100c05d000080bb000002001000646174610400000001000200")
					if err != nil {
						t.Error(err)
						return
					}
					copy(wav[44:], contractPCM(index+1))
					w.Header().Set("Content-Type", "audio/wav")
					_, _ = w.Write(wav)
				}))
				defer srv.Close()
				t.Setenv("FISH_BASE_URL", srv.URL)
				dir := t.TempDir()
				path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "out.wav")
				src := "---\nprovider: fish\nmodel: " + model + "\ncast: auto\nstrict_tags: true\n---\n" +
					"[act \"scene direction\"]\n" +
					"lin: A[act \"outer direction\"|B[act \"轻轻地\"|C]D]E[act holding back tears]F[reset act]G\n" +
					"lin: [act \"say \\\"yes\\\" quietly\"]H[reset]I\n" +
					"[reset]\nlin: J\n"
				if err := os.WriteFile(path, []byte(src), 0600); err != nil {
					t.Fatal(err)
				}
				if err := run(context.Background(), []string{"synth", path, "--concurrency", strconv.Itoa(concurrency), "-o", out}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
					t.Fatal(err)
				}
				var wantPCM []byte
				mu.Lock()
				defer mu.Unlock()
				for i, text := range want {
					if seen[text] != 1 {
						t.Errorf("requests for %q = %d, want 1", text, seen[text])
					}
					wantPCM = append(wantPCM, contractPCM(i+1)...)
				}
				data, err := os.ReadFile(out)
				if err != nil || len(data) != 44+len(wantPCM) || !bytes.Equal(data[44:], wantPCM) {
					t.Fatalf("output=%x err=%v", data, err)
				}
			})
		}
	}
}

func TestFishInvalidLaterActPreservesOutput(t *testing.T) {
	for _, model := range []string{"s2.1-pro", "s2.1-pro-free", "s2-pro"} {
		for _, concurrency := range []int{1, 3} {
			for _, invalid := range []string{`[act]`, `[act ""]`, `[act "  "]`, `[act "quietly" "slowly"]`, `[act direction="quietly"]`, `[act "quietly" slowly]`, `[act "[laughing]"]`, `[act "soft]"]`, `[act "<|speaker:0|>"]`, `[act \[laughing\]]`} {
				t.Run(fmt.Sprintf("%s/%d/%s", model, concurrency, invalid), func(t *testing.T) {
					t.Setenv("FISH_API_KEY", "FAKE-FISH-KEY")
					t.Setenv("FISH_MODEL", model)
					t.Setenv("FISH_VOICE_ID", "test-reference")
					var calls atomic.Int32
					srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						calls.Add(1)
						w.WriteHeader(http.StatusBadRequest)
					}))
					defer srv.Close()
					t.Setenv("FISH_BASE_URL", srv.URL)
					dir := t.TempDir()
					path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "out.wav")
					src := "---\nprovider: fish\nstrict_tags: true\n---\n[act \"quietly\"|First]\n\nLater " + invalid + "invalid.\n"
					if err := os.WriteFile(path, []byte(src), 0600); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(out, []byte("KEEP"), 0600); err != nil {
						t.Fatal(err)
					}
					err := run(context.Background(), []string{"synth", path, "--concurrency", strconv.Itoa(concurrency), "-o", out}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
					data, readErr := os.ReadFile(out)
					if err == nil || calls.Load() != 0 || readErr != nil || string(data) != "KEEP" {
						t.Fatalf("error=%v calls=%d output=%q read=%v", err, calls.Load(), data, readErr)
					}
				})
			}
		}
	}
}
