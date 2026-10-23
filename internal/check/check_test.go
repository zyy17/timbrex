package check

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ast"
	"github.com/zyy17/timbrex/internal/parse"
)

func TestApplyIsRepeatableAcrossModes(t *testing.T) {
	for _, tt := range []struct {
		name string
		src  string
	}{
		{
			name: "point tags",
			src:  `[thoughtfully] [act holding back tears] [laugh nervously]`,
		},
		{
			name: "nested notes arguments and diagnostics",
			src: "---\ncast: auto\n---\nlin: " +
				`[thoughtfully|[ACT holding back tears "café" intensity=50%|[LaUgH nervously volume=-3dB|hello [轻轻地]]]]`,
		},
		{
			name: "nested invalid tags",
			src:  `[thoughtfully|[laugh nervously|[pinyin] [/whisper] [pause -1s]]]`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, initialStrict := range []bool{false, true} {
				script, err := parse.Parse("test.tbx", tt.src)
				if err != nil {
					t.Fatal(err)
				}
				for _, strict := range []bool{initialStrict, initialStrict, !initialStrict, !initialStrict, initialStrict} {
					// A shallow node copy aliases nested Content and Args. Parse an
					// independent tree so in-place mutations cannot hide regressions.
					fresh, err := parse.Parse("test.tbx", tt.src)
					if err != nil {
						t.Fatal(err)
					}
					want := Apply(fresh, strict)
					got := Apply(script, strict)
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("strict=%v: diagnostics = %#v, want %#v", strict, got, want)
					}
					if !reflect.DeepEqual(script, fresh) {
						t.Fatalf("strict=%v: repeated check differs from a fresh checked tree", strict)
					}
				}
			}
		})
	}
}

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

func TestPauseDurationKinds(t *testing.T) {
	for _, src := range []string{
		"Hi [pause 0ms]", "Hi [pause]", "Hi [pause short]", "Hi [silence 1s]", "Hi [break 0.5min]",
	} {
		script, err := parse.Parse("t.tbx", src)
		if err != nil {
			t.Fatal(err)
		}
		if diags := Apply(script, false); Failed(diags) != nil {
			t.Fatalf("%q: %v", src, diags)
		}
	}
	for _, src := range []string{
		"Hi [pause 0]", "Hi [pause .5s]", "Hi [pause 1m]", "Hi [pause 1s250ms]", `Hi [pause "1s"]`,
		"Hi [pause -1s]", "Hi [pause 61s]", "Hi [pause 1 2]", "Hi [silence NaN]", "Hi [break 2min]",
	} {
		script, err := parse.Parse("t.tbx", src)
		if err != nil {
			t.Fatal(err)
		}
		if err := Failed(Apply(script, false)); err == nil {
			t.Fatalf("accepted %q", src)
		}
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

func TestDisplayNamePrefixWarns(t *testing.T) {
	src := "---\ncast:\n  lin: { name: Lin }\n  anna: { name: Anna }\n  ann: { name: Ann }\n---\nLin: hello\n\nlin: real\n\nAnna: longer\n\nAnn: short\n"
	script, err := parse.Parse("scene.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if script.Blocks[0].Kind != ast.Narration || script.Blocks[1].Kind != ast.Turn {
		t.Fatalf("blocks = %+v", script.Blocks)
	}
	diags := Apply(script, false)
	if Failed(diags) != nil {
		t.Fatal(diags)
	}
	var got []string
	for _, d := range diags {
		if d.Code != "W103" || d.Severity != "warning" {
			t.Fatalf("diags = %#v", diags)
		}
		got = append(got, d.Msg)
	}
	want := []string{
		`"Lin:" looks like speaker lin (display name Lin); treated as narration`,
		`"Anna:" looks like speaker anna (display name Anna); treated as narration`,
		`"Ann:" looks like speaker ann (display name Ann); treated as narration`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got %v", got)
	}

	// A full-width colon is still the display name, not a speaker.
	src = "---\ncast:\n  lin: { name: Lin }\n---\nLin\uFF1Ahello\n"
	script, err = parse.Parse("scene.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	diags = Apply(script, false)
	if len(diags) != 1 || diags[0].Code != "W103" || !strings.Contains(diags[0].Msg, "lin") {
		t.Fatalf("diags = %#v", diags)
	}

	// cast: auto takes an ident prefix, so the display name is a speaker.
	src = "---\ncast: auto\n---\nLin: hello\n"
	script, err = parse.Parse("scene.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if script.Blocks[0].Kind != ast.Turn || script.Blocks[0].Speaker != "Lin" {
		t.Fatalf("block = %+v", script.Blocks[0])
	}
	diags = Apply(script, false)
	for _, d := range diags {
		if d.Code == "W103" {
			t.Fatalf("diags = %#v", diags)
		}
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

func TestActingInfoSources(t *testing.T) {
	for _, tt := range []struct {
		name   string
		src    string
		strict bool
		codes  []string
	}{
		{name: "ascii", src: `[act "" "softly" gently]`},
		{name: "quoted", src: `[act "café"]`, codes: []string{"I050"}},
		{name: "note", src: `[act doucement café]`, codes: []string{"I050"}},
		{name: "keyed quoted", src: `[act note="café"]`, codes: []string{"I050"}},
		{name: "keyed word", src: `[act note=café]`},
		{name: "multiple sources", src: `[act "" "café" "轻轻地" doucement]`, codes: []string{"I050"}},
		{name: "spoken content", src: `[act "softly"|你好]`},
		{name: "free", src: `[轻轻地]`, codes: []string{"I050"}},
		{name: "strict free", src: `[轻轻地]`, strict: true, codes: []string{"V041"}},
		{name: "strict explicit", src: `[act "轻轻地"]`, strict: true, codes: []string{"I050"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			script, err := parse.Parse("test.tbx", tt.src)
			if err != nil {
				t.Fatal(err)
			}
			var codes []string
			for _, diag := range Apply(script, tt.strict) {
				codes = append(codes, diag.Code)
				if diag.Code == "I050" && diag.Severity != "info" {
					t.Fatalf("diagnostic = %+v", diag)
				}
			}
			if !reflect.DeepEqual(codes, tt.codes) {
				t.Fatalf("codes = %v, want %v", codes, tt.codes)
			}
		})
	}
}

func TestActingInfoAllocationGrowth(t *testing.T) {
	const count = 2000
	script, err := parse.Parse("test.tbx", "[act "+strings.Repeat(`"a" `, count)+"]")
	if err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(1, func() {
		if diags := Apply(script, false); len(diags) != 0 {
			t.Fatal(diags)
		}
	})
	// Detecting non-ASCII acting must not build a growing copy for every argument.
	if allocs > 100 {
		t.Fatalf("checking %d quoted arguments used %.0f allocations", count, allocs)
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
