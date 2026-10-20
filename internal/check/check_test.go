package check

import (
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/parse"
)

func TestStrictRejectsUnknownAndAllowsAct(t *testing.T) {
	script, err := parse.Parse("batch.tbx", "Hello [laff] [act \"holding back tears\"] [thoughtfully]\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, true)
	var codes []string
	for _, d := range diags {
		if d.Code == "V041" {
			codes = append(codes, d.Msg)
		}
	}
	if len(codes) != 2 {
		t.Fatalf("v041 = %#v all=%#v", codes, diags)
	}
	if strings.Contains(codes[0], "[laugh]") {
		t.Fatalf("laff is distance 3 and must not suggest laugh: %q", codes[0])
	}
	if !strings.Contains(codes[1], `[act "thoughtfully"]`) {
		t.Fatalf("phrase msg = %q", codes[1])
	}
	if Failed(diags) == nil {
		t.Fatal("expected failure")
	}
}

func TestStrictSuggestsNearMiss(t *testing.T) {
	script, err := parse.Parse("batch.tbx", "[paus]\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, true)
	if len(diags) != 1 || diags[0].Code != "V041" || !strings.Contains(diags[0].Msg, "[pause]") {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestSingleScriptKeepsFreeAct(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "[strong French accent]\n")
	if err != nil {
		t.Fatal(err)
	}
	if diags := Apply(script, false); Failed(diags) != nil {
		t.Fatal(diags)
	}
	tag := script.Blocks[0].Nodes[0]
	if !tag.Free || tag.Name != "act" || tag.Note != "strong French accent" {
		t.Fatalf("tag = %+v", tag)
	}
}

func TestClosingTag(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "Say [/whisper] please\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, false)
	if len(diags) != 1 || diags[0].Code != "V013" {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestWrapNeedsContent(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "[pinyin chong2]\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, false)
	if len(diags) != 1 || !strings.Contains(diags[0].Msg, "needs content") {
		t.Fatalf("diags = %#v", diags)
	}
	script, err = parse.Parse("scene.tbx", "[pinyin chong2|zhong]\n")
	if err != nil {
		t.Fatal(err)
	}
	if diags := Apply(script, false); Failed(diags) != nil {
		t.Fatal(diags)
	}
}

func TestPauseRejectsSpan(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "[pause|wait]\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, false)
	if len(diags) != 1 || !strings.Contains(diags[0].Msg, "does not take content") {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestCastAutoListsSpeakers(t *testing.T) {
	src := "---\ncast: auto\n---\nlin: hi\nchen: yo\nlin: again\n"
	script, err := parse.Parse("scene.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, false)
	if Failed(diags) != nil {
		t.Fatal(diags)
	}
	var got []string
	for _, d := range diags {
		if d.Severity != "info" {
			t.Fatalf("diags = %#v", diags)
		}
		got = append(got, d.Msg)
	}
	want := []string{
		`cast auto recognized speaker "lin"`,
		`cast auto recognized speaker "chen"`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got %v", got)
	}
}

func TestNarrationRejectsLeadingTurnTag(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "[overlap -200ms] hello\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, false)
	if len(diags) != 1 || !strings.Contains(diags[0].Msg, "start of a turn") {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestClosingWrapReportsOnlyV013(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "Say [/pinyin] please\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, false)
	if len(diags) != 1 || diags[0].Code != "V013" {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestOverlapMustLeadTurn(t *testing.T) {
	src := "---\ncast:\n  lin: { name: Lin }\n---\nlin: hello [overlap -200ms]\n"
	script, err := parse.Parse("scene.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, false)
	if len(diags) != 1 || !strings.Contains(diags[0].Msg, "start of a turn") {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestActNoteAndLaughNote(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "[act holding back tears] [laugh nervously]\n")
	if err != nil {
		t.Fatal(err)
	}
	Apply(script, false)
	var notes []string
	for _, node := range script.Blocks[0].Nodes {
		if node.Name == "" {
			continue
		}
		notes = append(notes, node.Name+"="+node.Note)
	}
	if strings.Join(notes, "|") != "act=holding back tears|laugh=nervously" {
		t.Fatalf("notes = %#v", notes)
	}
}

func TestNearMissWarnsWithoutFailing(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "Hi [paus] there\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, false)
	if Failed(diags) != nil || len(diags) != 1 || diags[0].Code != "W040" || diags[0].Severity != "warning" {
		t.Fatalf("diags = %#v", diags)
	}
	if !strings.Contains(diags[0].Msg, "[pause]") {
		t.Fatalf("msg = %q", diags[0].Msg)
	}
	tag := script.Blocks[0].Nodes[1]
	if !tag.Free || tag.Name != "act" || tag.Note != "paus" {
		t.Fatalf("tag = %+v", tag)
	}
}

func TestPhraseHasNoNearMissWarning(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "[strong French accent]\n")
	if err != nil {
		t.Fatal(err)
	}
	if diags := Apply(script, false); len(diags) != 0 {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestNonASCIIActingInfo(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "[act \"caf\u00e9\"] [\u00e9lan]\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, false)
	if Failed(diags) != nil {
		t.Fatal(diags)
	}
	var infos int
	for _, d := range diags {
		if d.Code == "I050" && d.Severity == "info" {
			infos++
		}
	}
	if infos != 2 {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestCanonicalNameAndLaughSpan(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "She [PAUSE 500ms] [laugh nervously|while laughing] [pinyin]\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, false)
	if len(diags) != 1 || !strings.Contains(diags[0].Msg, "needs content") {
		t.Fatalf("diags = %#v", diags)
	}
	pause := script.Blocks[0].Nodes[1]
	if pause.Name != "pause" || pause.Free || len(pause.Args) != 1 || pause.Args[0].Kind != "duration" {
		t.Fatalf("pause = %+v", pause)
	}
	laugh := script.Blocks[0].Nodes[3]
	if laugh.Name != "laugh" || laugh.Note != "nervously" || !laugh.Span || laugh.Content[0].Text != "while laughing" {
		t.Fatalf("laugh = %+v", laugh)
	}
}

func TestFreeSpanStillChecksInnerTags(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "[whatever|say [pinyin] now]\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, false)
	if len(diags) != 1 || !strings.Contains(diags[0].Msg, "needs content") {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestOverlapAtStartAndSceneLine(t *testing.T) {
	src := "---\ncast:\n  lin: { name: Lin }\n---\n[overlap -200ms]\nlin: [gap 1s] hello\n"
	script, err := parse.Parse("scene.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if diags := Apply(script, false); len(diags) != 0 {
		t.Fatalf("diags = %#v", diags)
	}
	script, err = parse.Parse("scene.tbx", "[overlap|nope]\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, false)
	if len(diags) != 1 || !strings.Contains(diags[0].Msg, "does not take content") {
		t.Fatalf("diags = %#v", diags)
	}
}

func TestNamespaceIsFreeUntilRegistered(t *testing.T) {
	script, err := parse.Parse("scene.tbx", "[el:strong French accent]\n")
	if err != nil {
		t.Fatal(err)
	}
	diags := Apply(script, true)
	if len(diags) != 1 || diags[0].Code != "V041" {
		t.Fatalf("diags = %#v", diags)
	}
}
