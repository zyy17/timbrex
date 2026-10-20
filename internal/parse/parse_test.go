package parse

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ast"
)

func TestPlainText(t *testing.T) {
	script, err := Parse("t.tbx", "Hello there.\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(script.Blocks) != 1 || script.Blocks[0].Kind != ast.Narration {
		t.Fatalf("blocks = %+v", script.Blocks)
	}
	if script.Blocks[0].Nodes[0].Text != "Hello there." {
		t.Fatalf("text = %q", script.Blocks[0].Nodes[0].Text)
	}
}

func TestSpeakerTagAndSpan(t *testing.T) {
	src := "" +
		"---\n" +
		"timbrex: 1\n" +
		"cast:\n" +
		"  lin: { name: Lin }\n" +
		"---\n" +
		"lin: [laugh] Really? [whisper|yes]\n"
	script, err := Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if script.Front.Timbrex != 1 || script.Front.Cast["lin"].Name != "Lin" {
		t.Fatalf("front = %+v", script.Front)
	}
	turn := script.Blocks[0]
	if turn.Kind != ast.Turn || turn.Speaker != "lin" {
		t.Fatalf("turn = %+v", turn)
	}
	if turn.Nodes[0].Name != "laugh" || turn.Nodes[0].Span {
		t.Fatalf("first tag = %+v", turn.Nodes[0])
	}
	span := turn.Nodes[2]
	if span.Name != "whisper" || !span.Span || span.Content[0].Text != "yes" {
		t.Fatalf("span = %+v", span)
	}
}

func TestCommentAndContinuation(t *testing.T) {
	src := "---\ncast:\n  lin: { name: Lin }\n---\n// aside\nlin: hello\nthere\n"
	script, err := Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if script.Blocks[0].Kind != ast.Comment || script.Blocks[0].Comment != "aside" {
		t.Fatalf("comment = %+v", script.Blocks[0])
	}
	turn := script.Blocks[1]
	if turn.Speaker != "lin" || turn.Nodes[0].Text != "hello\nthere" {
		t.Fatalf("turn = %+v", turn)
	}
}

func TestUnknownSpeakerIsNarration(t *testing.T) {
	src := "---\ncast:\n  lin: { name: Lin }\n---\nnope: hi\n"
	script, err := Parse("scene.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if script.Blocks[0].Kind != ast.Narration || script.Blocks[0].Nodes[0].Text != "nope: hi" {
		t.Fatalf("block = %+v", script.Blocks[0])
	}
}

func TestCastAuto(t *testing.T) {
	src := "---\ncast: auto\n---\nlin: hi\n"
	script, err := Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if !script.Front.CastAuto || script.Blocks[0].Speaker != "lin" {
		t.Fatalf("script = %+v", script)
	}
}

func TestAdjacentSpeakers(t *testing.T) {
	src := "---\ncast:\n  lin: { name: Lin }\n  chen: { name: Chen }\n---\nlin: one\nchen: two\n"
	script, err := Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(script.Blocks) != 2 || script.Blocks[0].Speaker != "lin" || script.Blocks[1].Speaker != "chen" {
		t.Fatalf("blocks = %+v", script.Blocks)
	}
}

func TestEscapedColonIsNotSpeaker(t *testing.T) {
	src := "---\ncast:\n  lin: { name: Lin }\n---\nlin\\: not a speaker\n"
	script, err := Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if script.Blocks[0].Kind != ast.Narration || script.Blocks[0].Nodes[0].Text != "lin: not a speaker" {
		t.Fatalf("block = %+v text=%q", script.Blocks[0], textOf(script.Blocks[0]))
	}
}

func TestTagArguments(t *testing.T) {
	src := "[pause 500ms] [pitch +2st] [volume -3dB] [speed 120%] [sfx file=door.wav] [act \"holding back tears\"]\n"
	script, err := Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	tags := script.Blocks[0].Nodes
	want := []string{"duration", "semitone", "decibel", "percent", "word", "quoted"}
	// nodes alternate tag, space, tag...
	var got []string
	for _, node := range tags {
		if node.Kind != ast.Tag || len(node.Args) == 0 {
			continue
		}
		got = append(got, node.Args[0].Kind)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("kinds = %v", got)
	}
	if tags[0].Args[0].Raw != "500ms" {
		t.Fatalf("pause = %+v", tags[0].Args)
	}
}

func TestNumberSuffixBoundary(t *testing.T) {
	script, err := Parse("t.tbx", "[pause 10stone] [pitch +2st] [wait 3minutes]\n")
	if err != nil {
		t.Fatal(err)
	}
	tags := tagArgs(script.Blocks[0].Nodes)
	if tags[0][0] != "word:10stone" {
		t.Fatalf("10stone = %v", tags[0])
	}
	if tags[1][0] != "semitone:+2st" {
		t.Fatalf("+2st = %v", tags[1])
	}
	if tags[2][0] != "word:3minutes" {
		t.Fatalf("3minutes = %v", tags[2])
	}
}

func TestLexErrorKeepsFileLine(t *testing.T) {
	src := "---\ntimbrex: 1\n---\n```ssml azure\n<speak>\n"
	_, err := Parse("scene.tbx", src)
	if err == nil {
		t.Fatal("expected unclosed fence")
	}
	var pe *Error
	if !errors.As(err, &pe) {
		t.Fatalf("err = %T %v", err, err)
	}
	if pe.File != "scene.tbx" || pe.Line != 4 || !strings.Contains(pe.Msg, "unclosed fence") {
		t.Fatalf("err = %+v", pe)
	}
}

func tagArgs(nodes []ast.Node) [][]string {
	var out [][]string
	for _, node := range nodes {
		if node.Kind != ast.Tag {
			continue
		}
		var args []string
		for _, arg := range node.Args {
			args = append(args, arg.Kind+":"+arg.Raw)
		}
		out = append(out, args)
	}
	return out
}

func TestUnclosedQuote(t *testing.T) {
	_, err := Parse("t.tbx", "[act \"hi]\n")
	if err == nil || !strings.Contains(err.Error(), "unclosed quote") {
		t.Fatalf("err = %v", err)
	}
}

func TestEmptyTag(t *testing.T) {
	_, err := Parse("t.tbx", "[]\n")
	if err == nil || !strings.Contains(err.Error(), "empty tag") {
		t.Fatalf("err = %v", err)
	}
}

func TestFencePassthrough(t *testing.T) {
	src := "```ssml azure\n<speak>[not a tag]</speak>\n```\nHello\n"
	script, err := Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if script.Blocks[0].Kind != ast.Fence || script.Blocks[0].Info != "ssml azure" {
		t.Fatalf("fence = %+v", script.Blocks[0])
	}
	if script.Blocks[0].Raw != "<speak>[not a tag]</speak>\n" {
		t.Fatalf("raw = %q", script.Blocks[0].Raw)
	}
	if script.Blocks[1].Nodes[0].Text != "Hello" {
		t.Fatalf("after = %+v", script.Blocks[1])
	}
}

func TestSceneLine(t *testing.T) {
	script, err := Parse("t.tbx", "[bgm rain.wav vol=-22dB]\nHello\n")
	if err != nil {
		t.Fatal(err)
	}
	if script.Blocks[0].Kind != ast.Scene || script.Blocks[0].Nodes[0].Name != "bgm" {
		t.Fatalf("scene = %+v", script.Blocks[0])
	}
	if script.Blocks[1].Kind != ast.Narration {
		t.Fatalf("next = %+v", script.Blocks[1])
	}
}

func textOf(block ast.Block) string {
	if len(block.Nodes) == 0 {
		return ""
	}
	return block.Nodes[0].Text
}

func TestUnclosedTag(t *testing.T) {
	_, err := Parse("scene.tbx", "Hello [slow\n")
	if err == nil || !strings.Contains(err.Error(), "unclosed tag") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnclosedFrontMatter(t *testing.T) {
	_, err := Parse("scene.tbx", "---\ntimbrex: 1\nhello\n")
	if err == nil || !strings.Contains(err.Error(), "unclosed front matter") {
		t.Fatalf("err = %v", err)
	}
}

func TestGrammarVersion(t *testing.T) {
	_, err := Parse("scene.tbx", "---\ntimbrex: 2\n---\nHi\n")
	if err == nil || !strings.Contains(err.Error(), "unsupported grammar version") {
		t.Fatalf("err = %v", err)
	}
}

func TestFullWidthIsText(t *testing.T) {
	const src = "See \u3010laugh\u3011 and note\uff1ahi\n"
	script, err := Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if script.Blocks[0].Kind != ast.Narration {
		t.Fatalf("kind = %v", script.Blocks[0].Kind)
	}
	if script.Blocks[0].Nodes[0].Text != strings.TrimSuffix(src, "\n") {
		t.Fatalf("text = %q", script.Blocks[0].Nodes[0].Text)
	}
}

func TestEscapedPipeIsNotSpan(t *testing.T) {
	script, err := Parse("t.tbx", "Say [whisper\\|hi] now\n")
	if err != nil {
		t.Fatal(err)
	}
	tag := script.Blocks[0].Nodes[1]
	if tag.Span || tag.Raw != `whisper|hi` {
		t.Fatalf("tag = %+v", tag)
	}
}

func TestEscapedBracketInContent(t *testing.T) {
	script, err := Parse("t.tbx", "[whisper|a\\]b]\n")
	if err != nil {
		t.Fatal(err)
	}
	tag := script.Blocks[0].Nodes[0]
	if !tag.Span || tag.Content[0].Text != "a]b" {
		t.Fatalf("tag = %+v", tag)
	}
}

func TestNestedTag(t *testing.T) {
	script, err := Parse("t.tbx", "[whisper|say [pause 1s] now]\n")
	if err != nil {
		t.Fatal(err)
	}
	span := script.Blocks[0].Nodes[0]
	if !span.Span || span.Name != "whisper" || len(span.Content) != 3 {
		t.Fatalf("span = %+v", span)
	}
	pause := span.Content[1]
	if pause.Name != "pause" || pause.Args[0].Raw != "1s" || pause.Args[0].Kind != "duration" {
		t.Fatalf("pause = %+v", pause)
	}
}

func TestSpanClosesAcrossLinesInParagraph(t *testing.T) {
	src := "---\ncast:\n  lin: { name: Lin }\n---\nlin: [whisper|hello\nthere]\n"
	script, err := Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(script.Blocks) != 1 || script.Blocks[0].Speaker != "lin" {
		t.Fatalf("blocks = %+v", script.Blocks)
	}
	span := script.Blocks[0].Nodes[0]
	if !span.Span || span.Content[0].Text != "hello\nthere" {
		t.Fatalf("span = %+v", span)
	}
}

func TestUnclosedTagStopsAtParagraph(t *testing.T) {
	_, err := Parse("scene.tbx", "[whisper|hello\n\nthere]\n")
	var pe *Error
	if err == nil || !errors.As(err, &pe) || pe.Code != "V012" || pe.Line != 1 {
		t.Fatalf("err = %v", err)
	}
}

func TestUnknownSpeakerStaysInTurn(t *testing.T) {
	src := "---\ncast:\n  lin: { name: Lin }\n---\nlin: hello\nnope: there\n"
	script, err := Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(script.Blocks) != 1 || script.Blocks[0].Speaker != "lin" {
		t.Fatalf("blocks = %+v", script.Blocks)
	}
	if script.Blocks[0].Nodes[0].Text != "hello\nnope: there" {
		t.Fatalf("text = %q", script.Blocks[0].Nodes[0].Text)
	}
}

func TestFrontMatterOnlyAtStart(t *testing.T) {
	script, err := Parse("t.tbx", "Hello\n---\ntimbrex: 1\n---\n")
	if err != nil {
		t.Fatal(err)
	}
	if script.Front.Timbrex != 0 || script.Blocks[0].Kind != ast.Narration {
		t.Fatalf("script = %+v", script)
	}
	if !strings.Contains(script.Blocks[0].Nodes[0].Text, "---") {
		t.Fatalf("text = %q", script.Blocks[0].Nodes[0].Text)
	}
}

func TestBOMAndCRLF(t *testing.T) {
	src := "\uFEFF---\r\ncast: auto\r\n---\r\nlin: hi\r\n"
	script, err := Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if !script.Front.CastAuto || script.Blocks[0].Speaker != "lin" {
		t.Fatalf("script = %+v", script)
	}
}

func TestMoreValues(t *testing.T) {
	src := "[pause 1min] [happy 0.8] [sfx \"distant thunder\" dur=3s]\n"
	script, err := Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	tags := tagArgs(script.Blocks[0].Nodes)
	if tags[0][0] != "duration:1min" || tags[1][0] != "number:0.8" {
		t.Fatalf("tags = %#v", tags)
	}
	if len(tags[2]) != 2 || tags[2][0] != "quoted:distant thunder" || tags[2][1] != "duration:3s" {
		t.Fatalf("sfx = %#v", tags[2])
	}
	var sfx ast.Node
	for _, node := range script.Blocks[0].Nodes {
		if node.Name == "sfx" {
			sfx = node
		}
	}
	if sfx.Args[1].Key != "dur" {
		t.Fatalf("sfx args = %+v", sfx.Args)
	}
}

func TestUnescapedBracketInsideTag(t *testing.T) {
	_, err := Parse("t.tbx", "[whisper [pause]]\n")
	if err == nil || !strings.Contains(err.Error(), "unescaped [ inside tag") {
		t.Fatalf("err = %v", err)
	}
}

func TestExample(t *testing.T) {
	b, err := os.ReadFile("../../examples/hello.tbx")
	if err != nil {
		t.Fatal(err)
	}
	script, err := Parse("examples/hello.tbx", string(b))
	if err != nil {
		t.Fatal(err)
	}
	if len(script.Blocks) != 3 {
		t.Fatalf("blocks = %d", len(script.Blocks))
	}
	if script.Blocks[1].Speaker != "lin" || script.Blocks[2].Speaker != "chen" {
		t.Fatalf("speakers = %q %q", script.Blocks[1].Speaker, script.Blocks[2].Speaker)
	}
}
