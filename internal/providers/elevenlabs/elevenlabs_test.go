package elevenlabs

import (
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func TestModelSpecificTags(t *testing.T) {
	seg := providers.Segment{Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}, {Kind: ir.Event, Name: "laugh"}}, Controls: []ir.Node{{Name: "whisper"}}}
	r, err := build(seg, "eleven_v3")
	if err != nil || r.Text != "[whispers]Hello[laughs]" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := build(seg, "eleven_multilingual_v2"); err == nil {
		t.Fatal("accepted unsupported nonverbal")
	}
	seg.Nodes = seg.Nodes[:1]
	if _, err := build(seg, "eleven_multilingual_v2"); err == nil {
		t.Fatal("accepted unsupported direction")
	}
	seg.Controls = nil
	if _, err := build(seg, "eleven_multilingual_v2"); err != nil {
		t.Fatal(err)
	}
	seg.Lang = "en"
	if _, err := build(seg, "eleven_multilingual_v2"); err == nil {
		t.Fatal("silently ignored language hint")
	}
}

func TestRejectNativeMarkupInLiteralTextAndInstructions(t *testing.T) {
	for _, seg := range []providers.Segment{
		{Nodes: []ir.Node{{Kind: ir.Text, Text: "[laughs]"}}},
		{Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: []ir.Node{{Name: "act", Args: []ir.Arg{{Raw: "[laughs]"}}}}},
	} {
		if _, err := build(seg, "eleven_v3"); err == nil {
			t.Fatalf("accepted %+v", seg)
		}
	}
}
