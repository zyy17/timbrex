package openai

import (
	"strings"
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
