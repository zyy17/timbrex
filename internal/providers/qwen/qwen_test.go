package qwen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func TestRejectIncompleteOrInvalidResponse(t *testing.T) {
	for _, data := range []string{`{`, `{}`, `{"code":"InvalidApiKey"}`, `{"output":{"audio":{"url":"https://example.com/audio"}}}`, `{"output":{"finish_reason":"stop","audio":{"url":"file:///audio"}}}`, `{"output":{"finish_reason":"stop","audio":{"url":"https://key:secret@example.com/audio"}}}`} {
		if _, err := (provider{}).download(context.Background(), []byte(data)); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}

func TestControlsAndNativeText(t *testing.T) {
	seg := providers.Segment{Voice: "longanhuan_v3.1", Lang: "EN-US", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: []ir.Node{
		{Name: "pitch", Args: []ir.Arg{{Raw: "+12st", Kind: "semitone"}}}, {Name: "volume", Args: []ir.Arg{{Raw: "0dB", Kind: "decibel"}}}, {Name: "act", Args: []ir.Arg{{Raw: "warmly"}}},
	}}
	r, err := build(seg, "qwen-audio-3.1-tts-flash")
	if err != nil {
		t.Fatal(err)
	}
	if r.Input.Pitch != 2 || r.Input.Volume == nil || *r.Input.Volume != 50 || r.Input.Instruction != "warmly" || r.Input.Languages[0] != "en" {
		t.Fatalf("input = %+v", r.Input)
	}
	seg.Nodes[0].Text = "literal [laughing]"
	if _, err := build(seg, r.Model); err == nil || !strings.Contains(err.Error(), "literal") {
		t.Fatalf("err = %v", err)
	}
	seg.Nodes[0].Text = "Hello"
	seg.Lang = "unknown"
	if _, err := build(seg, r.Model); err == nil {
		t.Fatal("accepted unknown language")
	}
	if _, err := build(seg, "qwen3-tts-flash"); err == nil {
		t.Fatal("accepted incompatible API model")
	}
}

func TestIncompleteDownloadPreservesOutput(t *testing.T) {
	for _, status := range []int{http.StatusPartialContent, http.StatusOK} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			posts, downloads := 0, 0
			var endpoint string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					posts++
					if r.Header.Get("Authorization") != "Bearer FAKE-KEY" {
						t.Error("missing synthesis credential")
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{"output": map[string]any{"finish_reason": "stop", "audio": map[string]string{"url": endpoint + "/audio?token=FAKE-SIGNED-TOKEN"}}})
					return
				}
				downloads++
				if r.Header.Get("Authorization") != "" || r.Header.Get("Range") != "" {
					t.Error("unexpected download credentials or range")
				}
				w.Header().Set("Content-Type", "audio/pcm")
				if status == http.StatusPartialContent {
					w.Header().Set("Content-Range", "bytes 0-3/100")
				} else {
					w.Header().Set("Content-Length", "100")
				}
				w.WriteHeader(status)
				_, _ = w.Write([]byte{1, 0, 2, 0})
			}))
			defer srv.Close()
			endpoint = srv.URL
			out := filepath.Join(t.TempDir(), "out.wav")
			if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			doc := &ir.Document{Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}}}
			_, err := (provider{}).Synthesize(context.Background(), doc, providers.Options{Out: out, Voice: "longanhuan_v3.1", Model: "qwen-audio-3.1-tts-flash", Endpoint: endpoint, APIKey: "FAKE-KEY"})
			if err == nil || strings.Contains(err.Error(), "FAKE-") {
				t.Errorf("error = %v", err)
			}
			if posts != 1 || downloads != 1 {
				t.Errorf("requests = %d POST, %d GET", posts, downloads)
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != "KEEP" {
				t.Errorf("output = %q, error = %v", data, err)
			}
			partials, err := filepath.Glob(filepath.Join(filepath.Dir(out), ".timbrex-*.partial"))
			if err != nil || len(partials) != 0 {
				t.Errorf("temporary files = %v, error = %v", partials, err)
			}
		})
	}
}
