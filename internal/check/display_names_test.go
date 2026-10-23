package check

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/zyy17/timbrex/internal/ast"
)

func TestDisplayNameMatching(t *testing.T) {
	for _, tt := range []struct {
		name string
		cast map[string]ast.Cast
		text string
		id   string
		want string
	}{
		{"ASCII colon", map[string]ast.Cast{"lin": {Name: "Lin"}}, "Lin: hello", "lin", "Lin:"},
		{"full-width colon", map[string]ast.Cast{"lin": {Name: "林"}}, "林：你好", "lin", "林："},
		{"trim Unicode whitespace", map[string]ast.Cast{"lin": {Name: "\t\u2003林\u00a0 "}}, "林: hello", "lin", "林:"},
		{"duplicate names", map[string]ast.Cast{"z": {Name: "林"}, "aa": {Name: " 林 "}, "a": {Name: "林"}}, "林：hello", "a", "林："},
		{"ASCII colon in name", map[string]ast.Cast{"short": {Name: "林"}, "long": {Name: "林: Anna"}}, "林: Anna：hello", "long", "林: Anna："},
		{"full-width colon in name", map[string]ast.Cast{"short": {Name: "Lin"}, "long": {Name: "Lin：安"}}, "Lin：安: hello", "long", "Lin：安:"},
		{"both colons in name", map[string]ast.Cast{"short": {Name: "林"}, "middle": {Name: "林:"}, "long": {Name: "林:："}}, "林:：:hello", "long", "林:：:"},
		{"colon-only name", map[string]ast.Cast{"short": {Name: ":"}, "long": {Name: ":："}}, ":：：hello", "long", ":：："},
		{"longer name without delimiter", map[string]ast.Cast{"short": {Name: "林"}, "long": {Name: "林: Ann"}}, "林: Anna: hello", "short", "林:"},
		{"name is exact prefix", map[string]ast.Cast{"lin": {Name: "林"}}, "林：", "lin", "林："},
		{"empty name", map[string]ast.Cast{"empty": {Name: " \t\u2003"}}, ":hello", "", ""},
		{"missing colon", map[string]ast.Cast{"lin": {Name: "Lin"}}, "Lin", "", ""},
		{"space before colon", map[string]ast.Cast{"lin": {Name: "Lin"}}, "Lin : hello", "", ""},
		{"leading whitespace", map[string]ast.Cast{"lin": {Name: "Lin"}}, " Lin: hello", "", ""},
		{"case sensitive", map[string]ast.Cast{"lin": {Name: "Lin"}}, "lin: hello", "", ""},
		{"not a complete name", map[string]ast.Cast{"ann": {Name: "Ann"}}, "Anna: hello", "", ""},
		{"unrelated text", map[string]ast.Cast{"lin": {Name: "Lin"}}, "Other: hello", "", ""},
		{"no cast", nil, "Lin: hello", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			script := &ast.Script{
				Front: ast.Front{Cast: tt.cast},
				Blocks: []ast.Block{{Kind: ast.Narration, Line: 7, Col: 3,
					Nodes: []ast.Node{{Kind: ast.Text, Text: tt.text, Line: 8, Col: 9}}}},
			}
			c := &checker{file: "names.tbx"}
			c.warnDisplayNames(script)
			var want []Diag
			if tt.id != "" {
				want = []Diag{{File: "names.tbx", Line: 7, Col: 3, Severity: "warning", Code: "W103",
					Msg: `"` + tt.want + `" looks like speaker ` + tt.id + ` (display name ` + strings.TrimSpace(tt.cast[tt.id].Name) + `); treated as narration`}}
			}
			if !reflect.DeepEqual(c.diags, want) {
				t.Fatalf("diagnostics = %#v, want %#v", c.diags, want)
			}
		})
	}
}

func TestDisplayNameBlockEligibility(t *testing.T) {
	text := []ast.Node{{Kind: ast.Text, Text: "Lin: hello"}}
	for _, tt := range []struct {
		name  string
		block ast.Block
		warn  bool
	}{
		{"narration", ast.Block{Kind: ast.Narration, Nodes: text}, true},
		{"turn", ast.Block{Kind: ast.Turn, Nodes: text}, false},
		{"scene", ast.Block{Kind: ast.Scene, Nodes: text}, false},
		{"comment", ast.Block{Kind: ast.Comment, Nodes: text}, false},
		{"fence", ast.Block{Kind: ast.Fence, Nodes: text}, false},
		{"empty paragraph", ast.Block{Kind: ast.Narration}, false},
		{"tag first", ast.Block{Kind: ast.Narration, Nodes: []ast.Node{{Kind: ast.Tag}, text[0]}}, false},
		{"empty text first", ast.Block{Kind: ast.Narration, Nodes: []ast.Node{{Kind: ast.Text}, text[0]}}, false},
		{"split prefix", ast.Block{Kind: ast.Narration, Nodes: []ast.Node{{Kind: ast.Text, Text: "Lin"}, {Kind: ast.Text, Text: ": hello"}}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			script := &ast.Script{Front: ast.Front{Cast: map[string]ast.Cast{"lin": {Name: "Lin"}}}, Blocks: []ast.Block{tt.block}}
			c := &checker{}
			c.warnDisplayNames(script)
			if (len(c.diags) == 1) != tt.warn {
				t.Fatalf("diagnostics = %#v, want warning: %v", c.diags, tt.warn)
			}
		})
	}
}

func TestDisplayNamesMatchReference(t *testing.T) {
	rng := rand.New(rand.NewPCG(103, 1))
	parts := []string{"A", "a", "林", "🙂", "e\u0301", ":", "：", " "}
	for trial := range 100 {
		cast := make(map[string]ast.Cast)
		texts := []string{"", ":", "：", "No match: hello"}
		for i := range 40 {
			name := ""
			for range 1 + rng.IntN(8) {
				name += parts[rng.IntN(len(parts))]
			}
			cast[fmt.Sprintf("speaker%02d", i)] = ast.Cast{Name: name}
			if i%3 == 0 {
				cast[fmt.Sprintf("duplicate%02d", i)] = ast.Cast{Name: "\u2003" + name + "\t"}
			}
			name = strings.TrimSpace(name)
			texts = append(texts, name+": speech", name+"：speech", name+"x: speech", name, " "+name+": speech")
		}
		script := &ast.Script{File: "reference.tbx", Front: ast.Front{Cast: cast}}
		for i, text := range texts {
			script.Blocks = append(script.Blocks, ast.Block{Kind: ast.Narration, Line: i + 1, Col: i%7 + 1,
				Nodes: []ast.Node{{Kind: ast.Text, Text: text}}})
		}
		c := &checker{file: script.File}
		c.warnDisplayNames(script)
		if want := referenceDisplayNames(script); !reflect.DeepEqual(c.diags, want) {
			t.Fatalf("trial %d: diagnostics = %#v, want %#v", trial, c.diags, want)
		}
	}
}

// Keep the original scan as an independent oracle for prefix and tie semantics.
func referenceDisplayNames(script *ast.Script) []Diag {
	var diags []Diag
	for _, block := range script.Blocks {
		if block.Kind != ast.Narration || len(block.Nodes) == 0 || block.Nodes[0].Kind != ast.Text {
			continue
		}
		bestLen := 0
		bestID, bestName, bestPrefix := "", "", ""
		for id, member := range script.Front.Cast {
			name := strings.TrimSpace(member.Name)
			if name == "" {
				continue
			}
			prefix := name + ":"
			if !strings.HasPrefix(block.Nodes[0].Text, prefix) {
				prefix = name + "："
				if !strings.HasPrefix(block.Nodes[0].Text, prefix) {
					continue
				}
			}
			n := utf8.RuneCountInString(name)
			if n > bestLen || (n == bestLen && (bestID == "" || id < bestID)) {
				bestLen, bestID, bestName, bestPrefix = n, id, name, prefix
			}
		}
		if bestID != "" {
			diags = append(diags, Diag{File: script.File, Line: block.Line, Col: block.Col, Severity: "warning", Code: "W103",
				Msg: `"` + bestPrefix + `" looks like speaker ` + bestID + ` (display name ` + bestName + `); treated as narration`})
		}
	}
	return diags
}

func TestDisplayNamesAllocationGrowth(t *testing.T) {
	script := displayNameScript(256, 512, false)
	// A per-check index may allocate for cast names, but must not build prefixes
	// for every cast member at every narration paragraph.
	allocs := testing.AllocsPerRun(1, func() { Apply(script, false) })
	if allocs > 10000 {
		t.Fatalf("checking 256 names and 512 paragraphs used %.0f allocations", allocs)
	}
}

func BenchmarkDisplayNames(b *testing.B) {
	for _, size := range []int{1, 16, 256} {
		for _, matched := range []bool{false, true} {
			b.Run(fmt.Sprintf("cast%d_paragraphs%d_matched%v", size, size, matched), func(b *testing.B) {
				script := displayNameScript(size, size, matched)
				b.ReportAllocs()
				for b.Loop() {
					Apply(script, false)
				}
			})
		}
	}
}

func displayNameScript(castCount, paragraphs int, matched bool) *ast.Script {
	script := &ast.Script{Front: ast.Front{Cast: make(map[string]ast.Cast, castCount)}}
	for i := range castCount {
		script.Front.Cast[fmt.Sprintf("char%d", i)] = ast.Cast{Name: fmt.Sprintf("Character %d", i)}
	}
	for i := range paragraphs {
		text := "Ordinary narration without a speaker prefix."
		if matched {
			text = fmt.Sprintf("Character %d: narration with a display name.", i%castCount)
		}
		script.Blocks = append(script.Blocks, ast.Block{Kind: ast.Narration, Nodes: []ast.Node{{Kind: ast.Text, Text: text}}})
	}
	return script
}

func TestDisplayNamesSkipNonNarration(t *testing.T) {
	script := displayNameScript(256, 512, true)
	for i := range script.Blocks {
		script.Blocks[i].Kind = ast.Turn
	}
	if allocs := testing.AllocsPerRun(1, func() { Apply(script, false) }); allocs != 0 {
		t.Fatalf("checking only turns used %.0f allocations", allocs)
	}
}

func BenchmarkDisplayNamesOnlyTurns(b *testing.B) {
	script := displayNameScript(256, 256, true)
	for i := range script.Blocks {
		script.Blocks[i].Kind = ast.Turn
	}
	b.ReportAllocs()
	for b.Loop() {
		Apply(script, false)
	}
}

func TestDisplayNamesRebuildAfterCastChanges(t *testing.T) {
	script := displayNameScript(1, 1, true)
	for _, name := range []string{"Character 0", "Another name", ""} {
		script.Front.Cast["char0"] = ast.Cast{Name: name}
		if got, want := Apply(script, false), referenceDisplayNames(script); !reflect.DeepEqual(got, want) {
			t.Fatalf("name %q: diagnostics = %#v, want %#v", name, got, want)
		}
	}
}
