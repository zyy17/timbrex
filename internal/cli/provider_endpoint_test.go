package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func endpointOptions(t *testing.T, id, endpoint string) providers.Options {
	t.Helper()
	models := map[string]string{"openai": "gpt-4o-mini-tts", "elevenlabs": "eleven_v3", "minimax": "speech-2.8-hd", "qwen": "qwen-audio-3.1-tts-flash", "gemini": "gemini-3.8-flash-tts", "fish": "s2.1-pro"}
	voice := "en-US-GuyNeural"
	if id == "gemini" {
		voice = "Kore"
	}
	return providers.Options{Out: filepath.Join(t.TempDir(), "out.wav"), Endpoint: endpoint, Model: models[id], Voice: voice, APIKey: "test-key"}
}

func endpointDocument() *ir.Document {
	return &ir.Document{Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello."}}}}}
}

func TestDirectProviderRejectsAmbiguousEndpoints(t *testing.T) {
	for _, vendor := range concurrencyVendors() {
		suffixes := []string{"#", "#secret", "?#"}
		if vendor.id != "azure" && vendor.id != "qwen" {
			suffixes = append(suffixes, "?", "?secret=value")
		}
		for _, suffix := range suffixes {
			t.Run(vendor.id+"/"+suffix, func(t *testing.T) {
				var calls atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
				defer srv.Close()
				opt := endpointOptions(t, vendor.id, srv.URL+"/proxy"+suffix)
				if err := os.WriteFile(opt.Out, []byte("KEEP"), 0o600); err != nil {
					t.Fatal(err)
				}
				p, err := providers.Get(vendor.id)
				if err != nil {
					t.Fatal(err)
				}
				_, err = p.Synthesize(context.Background(), endpointDocument(), opt)
				if err == nil || calls.Load() != 0 || strings.Contains(err.Error(), "secret") {
					t.Fatalf("ambiguous endpoint reached network or leaked: calls=%d error=%v", calls.Load(), err)
				}
				data, err := os.ReadFile(opt.Out)
				if err != nil || string(data) != "KEEP" {
					t.Fatalf("output changed: %q %v", data, err)
				}
				files, err := os.ReadDir(filepath.Dir(opt.Out))
				if err != nil || len(files) != 1 {
					t.Fatalf("left partial files: %v %v", files, err)
				}
			})
		}
	}
}

func TestDirectCompleteEndpointsPreserveQueries(t *testing.T) {
	for _, id := range []string{"azure", "qwen"} {
		for _, query := range []string{"?", "?existing=value%3F%23"} {
			t.Run(id+"/"+query, func(t *testing.T) {
				var posts, downloads atomic.Int32
				var base string
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodGet {
						downloads.Add(1)
						if id != "qwen" || r.Header.Get("Authorization") != "" {
							t.Error("download authentication changed")
						}
						w.Header().Set("Content-Type", "audio/pcm")
						_, _ = w.Write(concurrencyPCM(1))
						return
					}
					posts.Add(1)
					if r.URL.RequestURI() != "/proxy%3F%23"+query {
						t.Errorf("complete endpoint changed: %s", r.URL.RequestURI())
					}
					concurrencyReply(w, id, base, 1)
				}))
				defer srv.Close()
				base = srv.URL
				opt := endpointOptions(t, id, base+"/proxy%3F%23"+query)
				p, err := providers.Get(id)
				if err != nil {
					t.Fatal(err)
				}
				_, err = p.Synthesize(context.Background(), endpointDocument(), opt)
				if err != nil || posts.Load() != 1 {
					t.Fatalf("complete endpoint query rejected: posts=%d error=%v", posts.Load(), err)
				}
				wantDownloads := int32(0)
				if id == "qwen" {
					wantDownloads = 1
				}
				if downloads.Load() != wantDownloads {
					t.Fatalf("downloads=%d", downloads.Load())
				}
				data, err := os.ReadFile(opt.Out)
				if err != nil || !bytes.Equal(data, concurrencyWAV(concurrencyPCM(1))) {
					t.Fatalf("audio changed: %v", err)
				}
			})
		}
	}
}
