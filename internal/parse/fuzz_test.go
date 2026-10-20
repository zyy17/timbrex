package parse

import (
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/check"
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
		string([]byte{0xff, 0xfe, '[', 'a'}),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, src string) {
		script, err := Parse("fuzz.tbx", src)
		if err != nil || script == nil {
			return
		}
		_ = check.Apply(script, false)
		_ = check.Apply(script, true)
		_ = lower.Script(script)
	})
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
