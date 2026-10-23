package check_test

import (
	"reflect"
	"testing"

	"github.com/zyy17/timbrex/internal/ast"
	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/lower"
	"github.com/zyy17/timbrex/internal/parse"
)

func TestDisplayNameWarningPreservesNarration(t *testing.T) {
	const src = "---\nnarrator: storyteller\ncast:\n  lin: {name: '林:安'}\n---\n林:安：hello\n"
	script, err := parse.Parse("narration.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	original, err := parse.Parse("narration.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		diags := check.Apply(script, false)
		if check.Failed(diags) != nil || len(diags) != 1 || diags[0].Code != "W103" {
			t.Fatalf("diagnostics = %#v", diags)
		}
		if !reflect.DeepEqual(script, original) || script.Blocks[0].Kind != ast.Narration {
			t.Fatal("warning changed the parsed narration")
		}
		doc := lower.Script(script)
		want := []ir.Utterance{{Speaker: "storyteller", Nodes: []ir.Node{{Kind: ir.Text, Text: "林:安：hello"}}}}
		if !reflect.DeepEqual(doc.Utterances, want) {
			t.Fatalf("utterances = %#v, want %#v", doc.Utterances, want)
		}
	}
}
