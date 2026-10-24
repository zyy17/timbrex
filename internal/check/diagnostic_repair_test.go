package check

import (
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/parse"
)

func TestStrictActingRepairPreservesText(t *testing.T) {
	for _, tt := range []struct {
		name string
		src  string
	}{
		{"plain", `[thoughtfully]`},
		{"near miss", `[paus]`},
		{"quotes", `[unknown "softly"]`},
		{"escaped quotes", `[unknown "say \"hello\""]`},
		{"backslashes", `[unknown "C:\\temp\\file"]`},
		{"unquoted backslashes", `[unknown \\\\path]`},
		{"trailing backslash", `[unknown\\]`},
		{"literal markup", `[unknown left\]right\|end]`},
		{"newline", "[unknown \"line one\nline two\"]"},
		{"tab", "[unknown\tsoftly]"},
		{"Unicode", `[轻轻地]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			script, err := parse.Parse("original.tbx", tt.src)
			if err != nil {
				t.Fatal(err)
			}
			if diags := Apply(script, false); Failed(diags) != nil {
				t.Fatal(diags)
			}
			want := script.Blocks[0].Nodes[0].Note
			diags := Apply(script, true)
			if len(diags) != 1 || diags[0].Code != "V041" {
				t.Fatalf("diagnostics = %+v", diags)
			}
			_, repair, ok := strings.Cut(diags[0].Msg, "for free-form acting write ")
			if !ok {
				t.Fatalf("missing repair in %q", diags[0].Msg)
			}
			fixed, err := parse.Parse("repair.tbx", repair)
			if err != nil {
				t.Fatalf("recommended repair %q: %v", repair, err)
			}
			if diags := Apply(fixed, true); Failed(diags) != nil {
				t.Fatalf("recommended repair %q: %+v", repair, diags)
			}
			if len(fixed.Blocks) != 1 || len(fixed.Blocks[0].Nodes) != 1 {
				t.Fatalf("repair changed structure: %+v", fixed.Blocks)
			}
			node := fixed.Blocks[0].Nodes[0]
			if node.Name != "act" || node.Free || node.Note != "" || len(node.Args) != 1 || node.Args[0].Key != "" || node.Args[0].Kind != "quoted" || node.Args[0].Raw != want {
				t.Fatalf("repair = %+v, want explicit acting text %q", node, want)
			}
		})
	}
}
