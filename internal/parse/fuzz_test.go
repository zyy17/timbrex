package parse

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/lower"
)

func FuzzParse(f *testing.F) {
	seeds := []string{
		"",
		"Hello\n",
		"[whisper|say [pause 1s]]\n",
		"[whisper|a]]b]\n",
		"say [[hi]] and \\[not\\]\n",
		"\\\n",
		"```ssml azure\n<speak>[x]</speak>\n```\n",
		"```\nraw\n```\n",
		"---\ncast: auto\n---\nlin: hi\r\n",
		"---\ntimbrex: 1\nhello\n",
		"[act \"a|b\" \"say [hi]\"]\n",
		"lin\\: not\r\nnext\n",
		"\uFEFFhello\r\n",
		"[/pinyin]\n",
		`[thoughtfully|[ACT holding back tears "café"|[LaUgH nervously|hello]]]`,
		`[bgm rain.wav -3 250ms +50% -2st -22dB "say \"hi\" \\ there" label="a|b [c]"]`,
		"---\ncast: auto\n---\n[slow]\nlin: [gap 1s] Hi [happy|there]\n",
		"---\ncast:\n  anna: {name: Ann}\n  ann: {name: Ann}\n---\nAnn: hi\n",
		string([]byte{0xff, 0xfe, '[', 'a'}),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, src string) {
		script, err := Parse("fuzz.tbx", src)
		again, againErr := Parse("fuzz.tbx", src)
		if (err == nil) != (againErr == nil) {
			t.Fatalf("parse outcome changed: %v, then %v", err, againErr)
		}
		if err != nil {
			return
		}
		if script == nil || again == nil {
			t.Fatal("parse succeeded without a script")
		}
		if !reflect.DeepEqual(script, again) {
			t.Fatal("parse is not deterministic")
		}
		for _, strict := range []bool{false, true, false} {
			fresh, err := Parse("fuzz.tbx", src)
			if err != nil || fresh == nil {
				t.Fatalf("fresh parse failed: script=%v, err=%v", fresh != nil, err)
			}
			want := check.Apply(fresh, strict)
			for range 2 {
				got := check.Apply(script, strict)
				if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(script, fresh) {
					t.Fatalf("strict=%v: repeated check differs from a fresh checked tree", strict)
				}
			}
			if check.Failed(want) != nil {
				continue
			}
			doc := lower.Script(script)
			if !reflect.DeepEqual(doc, lower.Script(fresh)) {
				t.Fatalf("strict=%v: lowering differs from a fresh checked tree", strict)
			}
			assertBalancedSpans(t, doc)
		}
	})
}

func assertBalancedSpans(t *testing.T, doc *ir.Document) {
	t.Helper()
	for i, utterance := range doc.Utterances {
		var stack []string
		for _, node := range utterance.Nodes {
			switch node.Kind {
			case ir.SpanStart:
				stack = append(stack, node.Name)
			case ir.SpanEnd:
				if len(stack) == 0 || stack[len(stack)-1] != node.Name {
					t.Fatalf("utterance %d: unmatched span end %q after %v", i, node.Name, stack)
				}
				stack = stack[:len(stack)-1]
			}
		}
		if len(stack) != 0 {
			t.Fatalf("utterance %d: unclosed spans %v", i, stack)
		}
	}
}

func TestDeepNestAndHugeTagDoNotPanic(t *testing.T) {
	const n = 4000
	var b strings.Builder
	b.Grow(n*3 + 4)
	for range n {
		b.WriteString("[w|")
	}
	b.WriteByte('x')
	for range n {
		b.WriteByte(']')
	}
	b.WriteByte('\n')
	if _, err := Parse("t.tbx", b.String()); err != nil {
		t.Fatal(err)
	}

	huge := "[" + strings.Repeat("a", 1<<20) + "]\n"
	script, err := Parse("t.tbx", huge)
	if err != nil || script == nil || len(script.Blocks) != 1 {
		t.Fatalf("huge err=%v script=%v", err, script != nil)
	}
}
