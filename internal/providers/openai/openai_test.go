package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func TestInstructionsStayOutOfSpokenText(t *testing.T) {
	seg := providers.Segment{Voice: "coral", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: []ir.Node{{Name: "act", Args: []ir.Arg{{Raw: `say "hello" softly`, Kind: "quoted"}}}}}
	r, err := build(seg, "gpt-4o-mini-tts")
	if err != nil || r.Input != "Hello" || r.Instructions != `say "hello" softly` {
		t.Fatalf("request=%+v err=%v", r, err)
	}
	if _, err := build(seg, "tts-1"); err == nil {
		t.Fatal("tts-1 cannot use instructions")
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
