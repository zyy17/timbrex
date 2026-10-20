package lower

import (
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/parse"
)

func TestLowerHello(t *testing.T) {
	src := "---\ncast:\n  lin: { name: Lin }\n---\nlin: [laugh] Hi [whisper|there]\n"
	script, err := parse.Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	doc := Script(script)
	if len(doc.Utterances) != 1 || doc.Utterances[0].Speaker != "lin" {
		t.Fatalf("doc = %+v", doc)
	}
	kinds := make([]ir.Kind, len(doc.Utterances[0].Nodes))
	for i, node := range doc.Utterances[0].Nodes {
		kinds[i] = node.Kind
	}
	want := []ir.Kind{ir.Event, ir.Text, ir.SpanStart, ir.Text, ir.SpanEnd}
	if len(kinds) != len(want) {
		t.Fatalf("kinds = %v", kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kinds = %v, want %v", kinds, want)
		}
	}
	if doc.Utterances[0].Nodes[0].Name != "laugh" {
		t.Fatalf("event = %+v", doc.Utterances[0].Nodes[0])
	}
}
