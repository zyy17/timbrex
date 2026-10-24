package fish

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

func TestSynthesizeRejectsUnsupportedModelsBeforeOutput(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write(fixtureWAV())
	}))
	defer srv.Close()
	pause := ir.Node{Kind: ir.Event, Name: "pause", Args: []ir.Arg{{Raw: "1ms", Kind: "duration"}}}
	for _, model := range []string{"", "unknown", "s1", "drama-3", "s2.1-pro-FREE", " s2-pro", "s2-pro "} {
		for _, tc := range []struct {
			name  string
			nodes []ir.Node
		}{
			{"empty", nil},
			{"silence", []ir.Node{pause}},
			{"mixed", []ir.Node{pause, {Text: "Hello"}}},
		} {
			t.Run(model+"/"+tc.name, func(t *testing.T) {
				for _, concurrency := range []int{0, 2} {
					out := filepath.Join(t.TempDir(), "out.wav")
					if err := os.WriteFile(out, []byte("KEEP"), 0600); err != nil {
						t.Fatal(err)
					}
					doc := &ir.Document{Utterances: []ir.Utterance{{Nodes: tc.nodes}}}
					_, err := (provider{}).Synthesize(context.Background(), doc, providers.Options{
						Model: model, Voice: "test-reference", Endpoint: srv.URL, Out: out, Concurrency: concurrency,
					})
					if err == nil || !strings.Contains(err.Error(), "unsupported Fish TTS model") {
						t.Fatalf("concurrency=%d: model validation bypassed: %v", concurrency, err)
					}
					data, err := os.ReadFile(out)
					if err != nil || string(data) != "KEEP" {
						t.Fatalf("output changed: %q, %v", data, err)
					}
				}
			})
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("unsupported models made %d requests", calls.Load())
	}
}

func TestSynthesizeSupportedModelsWithOnlySilence(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write(fixtureWAV())
	}))
	defer srv.Close()
	for _, model := range []string{"s2.1-pro", "s2.1-pro-free", "s2-pro"} {
		t.Run(model, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out.wav")
			doc := &ir.Document{Utterances: []ir.Utterance{{Nodes: []ir.Node{{Kind: ir.Event, Name: "pause", Args: []ir.Arg{{Raw: "1ms", Kind: "duration"}}}}}}}
			_, err := (provider{}).Synthesize(context.Background(), doc, providers.Options{Model: model, Endpoint: srv.URL, Out: out})
			if err != nil {
				t.Fatal(err)
			}
			wav, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			pcm, err := providers.DecodeWAV(wav)
			if err != nil || !bytes.Equal(pcm, make([]byte, 48)) {
				t.Fatalf("incorrect local silence: %x, %v", pcm, err)
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("local silence made %d requests", calls.Load())
	}
}
