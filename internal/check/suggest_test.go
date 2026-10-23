package check

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/lower"
	"github.com/zyy17/timbrex/internal/parse"
	"github.com/zyy17/timbrex/internal/vocab"
)

func TestSuggestMatchesReference(t *testing.T) {
	words := []string{"", "paus", "dreamy", "laff", "hut", "PAUS", "quik_breath", "é", "轻轻地", "two words", "pause\t", "el:paus", "pause\n", "pause\u00a0"}
	for _, name := range vocab.Names() {
		words = append(words, name, strings.ToUpper(name), strings.ReplaceAll(name, "-", "_"))
		for i := range len(name) {
			words = append(words, name[:i]+name[i+1:], name[:i]+"z"+name[i+1:], name[:i]+"z"+name[i:])
		}
	}
	c := &checker{}
	for _, word := range words {
		if got, want := c.suggest(word), referenceSuggestion(word); got != want {
			t.Fatalf("suggest(%q) = %q, want %q", word, got, want)
		}
	}
	// "hut" is equally close to "cut" and "hum"; map iteration must not decide.
	for range 20 {
		if got := (&checker{}).suggest("hut"); got != "cut" {
			t.Fatalf("tie suggestion = %q, want cut", got)
		}
	}
}

// Keep a straightforward, independent byte-distance oracle for matching semantics.
func referenceSuggestion(raw string) string {
	if strings.ContainsAny(raw, " \t:") {
		return ""
	}
	name := vocab.Canon(raw)
	best, bestD := "", 3
	for _, candidate := range vocab.Names() {
		matrix := make([][]int, len(name)+1)
		for i := range matrix {
			matrix[i] = make([]int, len(candidate)+1)
			matrix[i][0] = i
		}
		for j := range matrix[0] {
			matrix[0][j] = j
		}
		for i := 1; i <= len(name); i++ {
			for j := 1; j <= len(candidate); j++ {
				cost := 1
				if name[i-1] == candidate[j-1] {
					cost = 0
				}
				matrix[i][j] = min(matrix[i-1][j]+1, matrix[i][j-1]+1, matrix[i-1][j-1]+cost)
			}
		}
		d := matrix[len(name)][len(candidate)]
		if d < bestD || (d == bestD && (best == "" || candidate < best)) {
			best, bestD = candidate, d
		}
	}
	if bestD > 2 {
		return ""
	}
	return best
}

func TestRepeatedTagResolution(t *testing.T) {
	for _, raw := range []string{"paus", "dreamy", "PAUS", "  paus  ", "quik_breath", "hut", "strong French accent", "el:paus", "quick_breath", "PAUSE"} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/strict=%v", raw, strict), func(t *testing.T) {
				src := strings.Repeat("Hello ["+raw+"] world.\n\n", 3)
				script, err := parse.Parse("repeated.tbx", src)
				if err != nil {
					t.Fatal(err)
				}
				wantScript, err := parse.Parse("repeated.tbx", src)
				if err != nil {
					t.Fatal(err)
				}
				var wantDiags []Diag
				for i := range wantScript.Blocks {
					n := &wantScript.Blocks[i].Nodes[1]
					if spec, ok := vocab.Lookup(n.Name); ok {
						n.Name = spec.Name
						continue
					}
					n.Name, n.Free, n.Note, n.Args = "act", true, strings.TrimSpace(n.Raw), nil
					suggestion := referenceSuggestion(n.Note)
					if !strict && suggestion == "" {
						continue
					}
					d := Diag{File: "repeated.tbx", Line: n.Line, Col: n.Col, Severity: "warning", Code: "W040"}
					d.Msg = "unknown [" + n.Note + "] used as free-form acting; did you mean [" + suggestion + "]?"
					if strict {
						d.Severity, d.Code = "error", "V041"
						d.Msg = `unregistered tag "` + n.Note + `"; for free-form acting write [act "` + n.Note + `"]`
						if suggestion != "" {
							d.Msg = `unregistered tag "` + n.Note + `"; did you mean [` + suggestion + `]? for free-form acting write [act "` + n.Note + `"]`
						}
					}
					wantDiags = append(wantDiags, d)
				}
				// Rechecking the same tree must not retain check mode or diagnostic state.
				for range 2 {
					got := Apply(script, strict)
					if !reflect.DeepEqual(got, wantDiags) {
						t.Fatalf("diagnostics = %#v, want %#v", got, wantDiags)
					}
					if !reflect.DeepEqual(script, wantScript) {
						t.Fatalf("resolved script = %#v, want %#v", script, wantScript)
					}
					if Failed(got) == nil {
						// Rechecking must preserve the independently resolved reference tree.
						if err := Failed(Apply(wantScript, strict)); err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(lower.Script(script), lower.Script(wantScript)) {
							t.Fatal("lowered IR differs from reference resolution")
						}
					}
				}
			})
		}
	}
}

func TestSuggestionCacheLifetime(t *testing.T) {
	first := &checker{}
	for _, raw := range []string{"paus", "PAUS", "dreamy", "dreamy", "quik_breath", "QUIK-BREATH", "two words", "el:paus", "pause\t"} {
		first.suggest(raw)
	}
	want := map[string]string{"paus": "pause", "dreamy": "", "quik-breath": "quick-breath"}
	if !reflect.DeepEqual(first.suggestions, want) {
		t.Fatalf("cached suggestions = %#v, want %#v", first.suggestions, want)
	}
	second := &checker{}
	if second.suggestions != nil {
		t.Fatal("new checker already has cached suggestions")
	}
	if got := second.suggest("hut"); got != "cut" {
		t.Fatalf("second checker suggestion = %q, want cut", got)
	}
	if len(second.suggestions) != 1 || !reflect.DeepEqual(first.suggestions, want) {
		t.Fatal("suggestion cache is shared between checkers")
	}
}

func TestSuggestionCacheFastPaths(t *testing.T) {
	for _, raw := range []string{"PAUSE", "quick_breath", "act dreamy", "two words", "pause\tmore", "el:paus"} {
		for _, strict := range []bool{false, true} {
			c := &checker{strict: strict}
			script, err := parse.Parse("fast.tbx", "Hello ["+raw+"] world.")
			if err != nil {
				t.Fatal(err)
			}
			c.walk(script.Blocks[0].Nodes, false, false)
			if c.suggestions != nil {
				t.Fatalf("%q strict=%v populated the suggestion cache", raw, strict)
			}
		}
	}
}

func TestRepeatedSuggestionAllocationGrowth(t *testing.T) {
	const count = 1024
	for _, raw := range []string{"paus", "dreamy"} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/strict=%v", raw, strict), func(t *testing.T) {
				allocations := func(n int) float64 {
					script, err := parse.Parse("repeated.tbx", strings.Repeat("["+raw+"] ", n))
					if err != nil {
						t.Fatal(err)
					}
					return testing.AllocsPerRun(1, func() { Apply(script, strict) })
				}
				single, repeated := allocations(1), allocations(count)
				t.Logf("one tag = %.0f allocations, %d tags = %.0f", single, count, repeated)
				// Diagnostics still need their own messages and slice entries, but
				// matching work must not allocate once per repeated tag.
				limit := single + 4*count
				if !strict && raw == "dreamy" {
					limit = single + 16
				}
				if repeated > limit {
					t.Fatalf("one tag = %.0f allocations, %d tags = %.0f; limit %.0f", single, count, repeated, limit)
				}
			})
		}
	}
}

func BenchmarkCompileRepeatedTags(b *testing.B) {
	for _, tag := range []string{"dreamy", "paus", "act dreamy"} {
		b.Run(tag, func(b *testing.B) {
			src := strings.Repeat("Hello ["+tag+"] world.\n\n", 1024)
			b.ReportAllocs()
			b.SetBytes(int64(len(src)))
			b.ResetTimer()
			for b.Loop() {
				script, err := parse.Parse("benchmark.tbx", src)
				if err != nil {
					b.Fatal(err)
				}
				if err := Failed(Apply(script, false)); err != nil {
					b.Fatal(err)
				}
				if doc := lower.Script(script); len(doc.Utterances) != 1024 {
					b.Fatalf("utterances = %d, want 1024", len(doc.Utterances))
				}
			}
		})
	}
}
