package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func TestInstructionsStayOutOfSpokenText(t *testing.T) {
	seg := providers.Segment{Voice: "coral", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: []ir.Node{{Name: "act", Args: []ir.Arg{{Raw: `say "hello" softly`, Kind: "quoted"}}}}}
	r, err := build(seg, "gpt-4o-mini-tts")
	if err != nil || r.Input != "Hello" || r.Instructions != `say "hello" softly` {
		t.Fatalf("request=%+v err=%v", r, err)
	}
	seg.Controls = nil
	seg.Nodes[0].Text = strings.Repeat("中", 4096)
	if _, err := build(seg, "tts-1"); err != nil {
		t.Fatal(err)
	}
	seg.Nodes[0].Text += "中"
	if _, err := build(seg, "tts-1"); err == nil {
		t.Fatal("accepted oversized input")
	}
}

func TestLegacyModelControls(t *testing.T) {
	for _, model := range []string{"tts-1", "tts-1-hd"} {
		for _, tt := range []struct {
			name     string
			controls []ir.Node
			lang     string
			speed    float64
			wantErr  bool
		}{
			{name: "plain text"},
			{name: "speed", controls: []ir.Node{{Name: "speed", Args: []ir.Arg{{Raw: "1.2", Kind: "number"}}}}, speed: 1.2},
			{name: "acting", controls: []ir.Node{{Name: "happy"}}, wantErr: true},
			{name: "language only", lang: "zh-CN", wantErr: true},
		} {
			t.Run(model+"/"+tt.name, func(t *testing.T) {
				seg := providers.Segment{Voice: "alloy", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: tt.controls, Lang: tt.lang}
				r, err := build(seg, model)
				if tt.wantErr {
					if err == nil || !strings.Contains(err.Error(), model+" does not support acting instructions") {
						t.Fatalf("err = %v", err)
					}
					return
				}
				if err != nil || r.Instructions != "" || r.Speed != tt.speed {
					t.Fatalf("request = %+v, err = %v", r, err)
				}
			})
		}
	}
}

func TestAssembledInstructionCharacterLimit(t *testing.T) {
	// This is the adapter's local character bound, not a vendor token budget.
	const suffix = ". happy. Speak in zh-CN"
	for _, size := range []int{4096, 4097} {
		acting := strings.Repeat("中", size-utf8.RuneCountInString(suffix))
		seg := providers.Segment{
			Voice: "coral", Lang: "zh-CN", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}},
			Controls: []ir.Node{{Name: "act", Args: []ir.Arg{{Raw: acting, Kind: "quoted"}}}, {Name: "happy"}},
		}
		r, err := build(seg, "gpt-4o-mini-tts")
		if size > 4096 {
			if err == nil || !strings.Contains(err.Error(), "acting instructions exceed 4096 characters") {
				t.Fatalf("%d characters: err = %v", size, err)
			}
			continue
		}
		if err != nil || r.Instructions != acting+suffix {
			t.Fatalf("%d characters: instructions = %q, err = %v", size, r.Instructions, err)
		}
	}
}

func TestSpeedBoundaries(t *testing.T) {
	for _, tt := range []struct {
		value   string
		want    float64
		wantErr bool
	}{
		{"0.249", 0, true},
		{"0.25", 0.25, false},
		{"4", 4, false},
		{"4.001", 0, true},
	} {
		t.Run(tt.value, func(t *testing.T) {
			seg := providers.Segment{
				Voice: "coral", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}},
				Controls: []ir.Node{{Name: "speed", Args: []ir.Arg{{Raw: tt.value, Kind: "number"}}}},
			}
			r, err := build(seg, "gpt-4o-mini-tts")
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "value from 0.25 to 4") {
					t.Fatalf("err = %v", err)
				}
				return
			}
			if err != nil || r.Speed != tt.want {
				t.Fatalf("speed = %g, err = %v", r.Speed, err)
			}
		})
	}
}

func TestJSONSuccessResponsePreservesOutput(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Content-Type", "audio/pcm")
			_, _ = w.Write([]byte{1, 0})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// An even-length body would otherwise pass raw PCM's byte-count check.
		_, _ = w.Write([]byte(`{"error":"private-text"}`))
	}))
	defer srv.Close()
	dir := t.TempDir()
	out := filepath.Join(dir, "out.wav")
	if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
		t.Fatal(err)
	}
	doc := &ir.Document{Utterances: []ir.Utterance{
		{Nodes: []ir.Node{{Kind: ir.Text, Text: "First"}}},
		{Nodes: []ir.Node{{Kind: ir.Text, Text: "Second"}}},
	}}
	_, err := (provider{}).Synthesize(context.Background(), doc, providers.Options{
		APIKey: "test-key", Model: "gpt-4o-mini-tts", Voice: "coral", Endpoint: srv.URL, Out: out,
	})
	if err == nil || !strings.Contains(err.Error(), "expected raw PCM response, got application/json") || calls.Load() != 2 {
		t.Fatalf("err = %v, calls = %d", err, calls.Load())
	}
	if strings.Contains(err.Error(), "private-text") {
		t.Fatalf("response body leaked: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "KEEP" {
		t.Fatalf("output = %q, err = %v", data, err)
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 1 {
		t.Fatalf("temporary files remain: %v, err = %v", files, err)
	}
}

func TestSynthesizeRejectsEmptyVoiceBeforeHTTP(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
	}))
	defer srv.Close()
	doc := &ir.Document{Utterances: []ir.Utterance{{Nodes: []ir.Node{{Text: "Hello"}}}}}
	_, err := (provider{}).Synthesize(context.Background(), doc, providers.Options{
		APIKey: "test-key", Model: "gpt-4o-mini-tts", Voice: "", Endpoint: srv.URL, Out: t.TempDir() + "/out.wav",
	})
	if err == nil || !strings.Contains(err.Error(), "providers.openai.voice") || calls.Load() != 0 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}
