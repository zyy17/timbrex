package lex

import (
	"strconv"
	"strings"
	"testing"
)

func TestScanLongIndentation(t *testing.T) {
	indent := strings.Repeat(" \t", 50000)
	for _, tc := range []struct {
		name string
		src  string
		kind Kind
		lit  string
	}{
		{"text", indent + "hello", Text, indent + "hello"},
		{"comment", indent + "// hello", Comment, " hello"},
		{"fence", indent + "```ssml\nraw\n" + indent + "```" + indent + "\n", Fence, "ssml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			toks := mustScan(t, tc.src)
			if toks[0].Kind != tc.kind || toks[0].Lit != tc.lit {
				t.Fatalf("first token kind = %v, length = %d", toks[0].Kind, len(toks[0].Lit))
			}
			if tc.kind == Fence && toks[0].Body != "raw\n" {
				t.Fatalf("body = %q", toks[0].Body)
			}
		})
	}
}

func BenchmarkScanIndentation(b *testing.B) {
	for _, n := range []int{1000, 10000} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			src := strings.Repeat(" ", n) + "hello"
			b.SetBytes(int64(len(src)))
			b.ReportAllocs()
			for b.Loop() {
				if _, err := Scan(src); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestScanTagsAndText(t *testing.T) {
	toks := mustScan(t, "Hi [laugh] there")
	got := kinds(toks)
	want := []Kind{Text, LBrack, Text, RBrack, Text, EOF}
	if !sameKinds(got, want) {
		t.Fatalf("kinds = %v, want %v", got, want)
	}
	if toks[2].Lit != "laugh" {
		t.Fatalf("tag lit = %q", toks[2].Lit)
	}
}

func TestScanPipeInsideTag(t *testing.T) {
	toks := mustScan(t, "[whisper|secret]")
	got := kinds(toks)
	want := []Kind{LBrack, Text, Pipe, Text, RBrack, EOF}
	if !sameKinds(got, want) {
		t.Fatalf("kinds = %v, want %v", got, want)
	}
}

func TestScanPipeOutsideTagIsText(t *testing.T) {
	toks := mustScan(t, "a|b")
	if len(toks) != 2 || toks[0].Kind != Text || toks[0].Lit != "a|b" {
		t.Fatalf("toks = %+v", toks)
	}
}

func TestScanNestedCloseIsTwoBrackets(t *testing.T) {
	toks := mustScan(t, "[whisper|say [pause 1s]]")
	got := kinds(toks)
	want := []Kind{LBrack, Text, Pipe, Text, LBrack, Text, RBrack, RBrack, EOF}
	if !sameKinds(got, want) {
		t.Fatalf("kinds = %v, want %v", got, want)
	}
	if toks[5].Lit != "pause 1s" {
		t.Fatalf("inner = %q", toks[5].Lit)
	}
}

func TestScanEscapes(t *testing.T) {
	toks := mustScan(t, "say [[hi]] and \\[not\\]")
	if joinText(toks) != "say [hi] and [not]" {
		t.Fatalf("lit = %q", joinText(toks))
	}
}

func TestScanComment(t *testing.T) {
	toks := mustScan(t, "  // note\nhello")
	if toks[0].Kind != Comment || toks[0].Lit != " note" {
		t.Fatalf("first = %+v", toks[0])
	}
	if toks[1].Kind != Newline || toks[2].Lit != "hello" {
		t.Fatalf("toks = %+v", toks)
	}
}

func TestScanCommentOnlyAtLineStart(t *testing.T) {
	toks := mustScan(t, "say // still text")
	if toks[0].Kind != Text || toks[0].Lit != "say // still text" {
		t.Fatalf("lit = %q", toks[0].Lit)
	}
}

func TestScanFence(t *testing.T) {
	toks := mustScan(t, "```ssml azure\n<speak>[x]</speak>\n```\n")
	if len(toks) != 2 || toks[0].Kind != Fence {
		t.Fatalf("toks = %+v", toks)
	}
	if toks[0].Lit != "ssml azure" || toks[0].Body != "<speak>[x]</speak>\n" {
		t.Fatalf("fence = %+v", toks[0])
	}
}

func TestFenceCloserRequiresEndOfLine(t *testing.T) {
	for _, suffix := range []string{"text", "\x00", "\u2003"} {
		body := "```" + suffix + "\n"
		toks := mustScan(t, "```ssml\n"+body+"```\n")
		if len(toks) != 2 || toks[0].Kind != Fence || toks[0].Body != body {
			t.Fatalf("suffix %q: tokens = %+v", suffix, toks)
		}
	}
}

func TestScanUnclosedFence(t *testing.T) {
	_, err := Scan("```ssml azure\n<speak>\n")
	if err == nil || err.Error() != "unclosed fence" {
		t.Fatalf("err = %v", err)
	}
}

func TestScanFenceNeedsInfo(t *testing.T) {
	_, err := Scan("```\nraw\n```\n")
	if err == nil || err.Error() != "fence needs an info string" {
		t.Fatalf("err = %v", err)
	}
}

func mustScan(t *testing.T, src string) []Token {
	t.Helper()
	toks, err := Scan(src)
	if err != nil {
		t.Fatal(err)
	}
	return toks
}

func joinText(toks []Token) string {
	var b []byte
	for _, tok := range toks {
		if tok.Kind == Text {
			b = append(b, tok.Lit...)
		}
	}
	return string(b)
}

func kinds(toks []Token) []Kind {
	out := make([]Kind, len(toks))
	for i, tok := range toks {
		out[i] = tok.Kind
	}
	return out
}

func sameKinds(got, want []Kind) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
