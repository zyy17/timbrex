package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func testWave() []byte {
	b, _ := hex.DecodeString("524946462800000057415645666d74201000000001000100c05d000080bb000002001000646174610400000001000200")
	return b
}
func response(pcm []byte) []byte {
	b, _ := json.Marshal(map[string]any{"candidates": []any{map[string]any{"finishReason": "STOP", "content": map[string]any{"parts": []any{map[string]any{"inlineData": map[string]string{"mimeType": "audio/wav", "data": base64.StdEncoding.EncodeToString(pcm)}}}}}}})
	return b
}
func TestBuildContract(t *testing.T) {
	seg := providers.Segment{Voice: "Kore", Lang: "en-GB", Nodes: []ir.Node{{Text: "Hello"}, {Kind: ir.Event, Name: "sigh"}}, Controls: []ir.Node{{Name: "act", Args: []ir.Arg{{Raw: "speak gently", Kind: "quoted"}}}, {Name: "slow"}}}
	r, err := build(seg, "gemini-3.8-flash-tts")
	if err != nil {
		t.Fatal(err)
	}
	p := r.Contents[0].Parts[0]
	if p.Text != "Hello<sigh>" || p.SpeechMetadata.Style != "speak gently. speak slower" || r.GenerationConfig.SpeechConfig.LanguageCode != "en-GB" {
		t.Fatalf("%+v", r)
	}
	for _, voice := range voices {
		seg.Voice = voice
		if _, err := build(seg, "gemini-3.8-flash-tts"); err != nil {
			t.Fatal(err)
		}
	}
}
func TestBuildRejectsUnsupported(t *testing.T) {
	base := providers.Segment{Voice: "Kore", Nodes: []ir.Node{{Text: "Hello"}}}
	for _, name := range []string{"speed", "pitch", "volume", "sfx", "oral"} {
		seg := base
		seg.Controls = []ir.Node{{Name: name}}
		if _, err := build(seg, "gemini-3.8-flash-tts"); err == nil {
			t.Fatalf("accepted %s", name)
		}
	}
	for _, tc := range []struct {
		name   string
		change func(*providers.Segment)
	}{
		{"literal markup", func(s *providers.Segment) { s.Nodes = []ir.Node{{Text: "<laugh>"}} }},
		{"voice", func(s *providers.Segment) { s.Voice = "voice_private-id" }},
		{"locale", func(s *providers.Segment) { s.Lang = "en" }},
		{"event", func(s *providers.Segment) { s.Nodes = []ir.Node{{Kind: ir.Event, Name: "burp"}} }},
		{"event args", func(s *providers.Segment) { s.Nodes = []ir.Node{{Kind: ir.Event, Name: "laugh", Note: "quietly"}} }},
		{"fast args", func(s *providers.Segment) { s.Controls = []ir.Node{{Name: "fast", Args: []ir.Arg{{Raw: "2"}}}} }},
		{"size", func(s *providers.Segment) { s.Nodes = []ir.Node{{Text: strings.Repeat("中", 4097)}} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seg := base
			tc.change(&seg)
			if _, err := build(seg, "gemini-3.8-flash-tts"); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	if _, err := build(base, "gemini-2.5-flash-preview-tts"); err == nil {
		t.Fatal("accepted old schema model")
	}
	base.Nodes = []ir.Node{{Text: strings.Repeat("中", 4096)}}
	if _, err := build(base, "gemini-3.8-flash-tts"); err != nil {
		t.Fatal(err)
	}
	base.Controls = []ir.Node{{Name: "happy"}}
	if _, err := build(base, "gemini-3.8-flash-tts"); err == nil {
		t.Fatal("style omitted from bound")
	}
}
func TestDecode(t *testing.T) {
	good := response(testWave())
	got, err := decode(good)
	if err != nil || !bytes.Equal(got, []byte{1, 0, 2, 0}) {
		t.Fatalf("%x %v", got, err)
	}
	for name, b := range map[string][]byte{
		"malformed": []byte(`{"candidates":"FAKE-SECRET"}`), "empty": []byte(`{}`), "null": []byte(`null`),
		"blocked":             append([]byte(`{"promptFeedback":{"blockReason":"SAFETY"},`), good[1:]...),
		"API error":           append([]byte(`{"error":{"message":"FAKE-SECRET"},`), good[1:]...),
		"unfinished":          bytes.Replace(good, []byte("STOP"), []byte("MAX_TOKENS"), 1),
		"missing completion":  bytes.Replace(good, []byte(`"finishReason":"STOP"`), []byte(`"unused":"STOP"`), 1),
		"wrong mime":          bytes.Replace(good, []byte("audio/wav"), []byte("audio/l16"), 1),
		"extra part":          bytes.Replace(good, []byte(`"parts":[`), []byte(`"parts":[{"text":"FAKE-SECRET"},`), 1),
		"union":               bytes.Replace(good, []byte(`"inlineData":`), []byte(`"text":"FAKE-SECRET","inlineData":`), 1),
		"multiple candidates": bytes.Replace(good, []byte(`"candidates":[`), []byte(`"candidates":[{},`), 1),
		"safety rating":       bytes.Replace(good, []byte(`"finishReason":"STOP"`), []byte(`"finishReason":"STOP","safetyRatings":[{"blocked":true}]`), 1),
		"base64":              bytes.Replace(good, []byte(base64.StdEncoding.EncodeToString(testWave())), []byte("%%%FAKE-SECRET"), 1),
		"raw PCM":             response([]byte{1, 0, 2, 0}), "truncated WAV": response(testWave()[:45]), "empty audio": response(nil),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := decode(b)
			if err == nil || strings.Contains(err.Error(), "FAKE-SECRET") {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
func TestFailuresPreserveOutput(t *testing.T) {
	for _, kind := range []string{"blocked", "partial", "redirect", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				switch kind {
				case "blocked":
					_, _ = w.Write([]byte(`{"promptFeedback":{"blockReason":"SAFETY"}}`))
				case "partial":
					w.WriteHeader(206)
					_, _ = w.Write(response(testWave()))
				case "redirect":
					w.Header().Set("Location", "/private")
					w.WriteHeader(307)
				case "cancel":
					_, _ = io.Copy(io.Discard, r.Body)
					cancel()
					select {
					case <-r.Context().Done():
					case <-time.After(2 * time.Second):
						t.Error("server did not observe cancellation")
					}
				}
			}))
			defer srv.Close()
			out := filepath.Join(t.TempDir(), "out.wav")
			if err := os.WriteFile(out, []byte("KEEP"), 0600); err != nil {
				t.Fatal(err)
			}
			doc := &ir.Document{Utterances: []ir.Utterance{{Nodes: []ir.Node{{Text: "Hello"}}}}}
			_, err := (provider{}).Synthesize(ctx, doc, providers.Options{APIKey: "FAKE-KEY", Model: "gemini-3.8-flash-tts", Voice: "Kore", Endpoint: srv.URL, Out: out})
			if err == nil || calls.Load() != 1 {
				t.Fatalf("err=%v calls=%d", err, calls.Load())
			}
			if kind == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancel chain: %v", err)
			}
			b, _ := os.ReadFile(out)
			if string(b) != "KEEP" {
				t.Fatal("replaced output")
			}
		})
	}
}
func FuzzDecode(f *testing.F) {
	f.Add(response(testWave()))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 65536 {
			t.Skip()
		}
		pcm, err := decode(data)
		if err == nil && (len(pcm) == 0 || len(pcm)%2 != 0) {
			t.Fatal("invalid PCM")
		}
	})
}
