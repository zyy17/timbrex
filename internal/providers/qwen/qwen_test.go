package qwen

import (
	"context"
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
