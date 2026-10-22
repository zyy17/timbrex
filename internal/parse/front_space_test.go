package parse

import (
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ast"
)

func TestFrontMatterAllowsTrailingWhitespace(t *testing.T) {
	for _, open := range []string{"--- \n", "---\t\n", "---  \t\n"} {
		src := open + "cast: auto\n---\nlin: hi\n"
		script, err := Parse("space.tbx", src)
		if err != nil {
			t.Fatalf("%q parse: %v", open, err)
		}
		if !script.Front.CastAuto || len(script.Blocks) != 1 || script.Blocks[0].Kind != ast.Turn || script.Blocks[0].Speaker != "lin" {
			t.Fatalf("%q CastAuto=%v blocks=%+v", open, script.Front.CastAuto, script.Blocks)
		}
	}
}

func TestFourDashesIsNotFrontMatter(t *testing.T) {
	script, err := Parse("dash.tbx", "----\ncast: auto\n---\nlin: hi\n")
	if err != nil {
		t.Fatal(err)
	}
	if script.Front.CastAuto || script.Blocks[0].Kind != ast.Narration {
		t.Fatalf("script = %+v", script)
	}
	if !strings.Contains(script.Blocks[0].Nodes[0].Text, "cast: auto") {
		t.Fatalf("text = %q", script.Blocks[0].Nodes[0].Text)
	}
}

func TestQuotedCloserStaysInsideArgument(t *testing.T) {
	script, err := Parse("q.tbx", `[act "hello]world"]`)
	if err != nil {
		t.Fatal(err)
	}
	tag := script.Blocks[0].Nodes[0]
	if tag.Name != "act" || tag.Span || len(tag.Args) != 1 || tag.Args[0].Raw != "hello]world" {
		t.Fatalf("tag = %+v", tag)
	}
}
