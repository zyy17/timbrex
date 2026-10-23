package providers_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/lower"
	"github.com/zyy17/timbrex/internal/parse"
	"github.com/zyy17/timbrex/internal/providers"
)

func document(t testing.TB, src string) *ir.Document {
	t.Helper()
	script, err := parse.Parse("test.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if err := check.Failed(check.Apply(script, false)); err != nil {
		t.Fatal(err)
	}
	return lower.Script(script)
}

func TestPlanScopesAndReset(t *testing.T) {
	doc := document(t, "---\ncast: auto\n---\n[slow]\nlin: A [happy] B [speed 1.5|C [sad|D] E] F [reset] G\nchen: H\n[reset]\nI\n")
	segs, err := providers.Plan(doc, "openai", "coral")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"A:slow", "B:slow,happy", "C:happy,speed", "D:speed,sad", "E:happy,speed", "F:slow,happy", "G:slow", "H:slow", "I:"}
	if len(segs) != len(want) {
		t.Fatalf("segments = %+v", segs)
	}
	for i, seg := range segs {
		text, err := providers.Text(seg.Nodes, nil)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, n := range seg.Controls {
			names = append(names, n.Name)
		}
		if got := strings.TrimSpace(text) + ":" + strings.Join(names, ","); got != want[i] {
			t.Fatalf("segment %d = %q, want %q", i, got, want[i])
		}
	}
}

func TestPlanVoicesLanguageAndPauses(t *testing.T) {
	doc := document(t, "---\nlang: zh-CN\ncast:\n  lin: {voices: {qwen: a}}\n  chen: {voices: {qwen: b}}\n---\nlin: One [as chen|Two] [voice direct|Three] [lang en-US|Four] [pause 500ms] Five\n")
	segs, err := providers.Plan(doc, "qwen", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "direct", "a", "", "a"}
	if len(segs) != len(want) {
		t.Fatalf("segments = %+v", segs)
	}
	for i, seg := range segs {
		if seg.Voice != want[i] {
			t.Fatalf("segment %d voice %q", i, seg.Voice)
		}
	}
	if segs[0].Lang != "zh-CN" || segs[3].Lang != "en-US" || segs[5].Lang != "zh-CN" || segs[4].Pause != 500*time.Millisecond {
		t.Fatalf("segments = %+v", segs)
	}
}

func TestResetFeatureRestoresSceneValue(t *testing.T) {
	doc := document(t, "[speed 0.9]\nHello [fast] quick [reset speed] normal\n")
	segs, err := providers.Plan(doc, "openai", "coral")
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 3 || segs[2].Controls[0].Name != "speed" || segs[2].Controls[0].Args[0].Raw != "0.9" {
		t.Fatalf("segments = %+v", segs)
	}
}

func TestPlanRejectsBadPausesAndFences(t *testing.T) {
	for _, tt := range []struct {
		source string
		args   []ir.Arg
	}{
		{"Hi [pause -1s]", []ir.Arg{{Raw: "-1s", Kind: "duration"}}},
		{"Hi [pause 61s]", []ir.Arg{{Raw: "61s", Kind: "duration"}}},
		{"Hi [pause 1 2]", []ir.Arg{{Raw: "1", Kind: "number"}, {Raw: "2", Kind: "number"}}},
		{"Hi [pause NaN]", []ir.Arg{{Raw: "NaN", Kind: "word"}}},
		{"Hi [pause 1m]", []ir.Arg{{Raw: "1m", Kind: "word"}}},
		{"Hi [pause 1s250ms]", []ir.Arg{{Raw: "1s250ms", Kind: "word"}}},
		{`Hi [pause "1s"]`, []ir.Arg{{Raw: "1s", Kind: "quoted"}}},
		{"Hi [pause 0]", []ir.Arg{{Raw: "0", Kind: "number"}}},
	} {
		script, err := parse.Parse("test.tbx", tt.source)
		if err != nil {
			t.Fatal(err)
		}
		if err := check.Failed(check.Apply(script, false)); err == nil {
			t.Fatalf("check accepted %q", tt.source)
		}
		// Invalid checked ASTs never enter lowering. Exercise the planner's
		// defensive boundary with malformed IR supplied directly instead.
		doc := &ir.Document{Utterances: []ir.Utterance{{Nodes: []ir.Node{
			{Kind: ir.Text, Text: "Hi "},
			{Kind: ir.Event, Name: "pause", Args: tt.args},
		}}}}
		if _, err := providers.Plan(doc, "openai", "coral"); err == nil {
			t.Fatalf("plan accepted malformed pause IR for %q", tt.source)
		}
	}
	for _, src := range []string{"```ssml azure\n<speak/>\n```", "// empty"} {
		if _, err := providers.Plan(document(t, src), "openai", "coral"); err == nil {
			t.Fatalf("accepted %q", src)
		}
	}
}

func TestPlanRejectsEmptyVoice(t *testing.T) {
	doc := document(t, "Hello\n")
	if _, err := providers.Plan(doc, "openai", ""); err == nil || !strings.Contains(err.Error(), "providers.openai.voice") {
		t.Fatalf("err = %v", err)
	}
	if _, err := providers.Plan(doc, "openai", "  "); err == nil {
		t.Fatal("accepted whitespace voice")
	}
	mapped := document(t, "---\ncast:\n  lin: {}\n---\nlin: Hello\n")
	if _, err := providers.Plan(mapped, "openai", ""); err == nil {
		t.Fatal("accepted empty fallback for a known speaker")
	}
	direct, err := providers.Plan(document(t, `[voice coral|Hello]`), "openai", "")
	if err != nil || len(direct) != 1 || direct[0].Voice != "coral" {
		t.Fatalf("voice span: %+v %v", direct, err)
	}
	pause, err := providers.Plan(document(t, "[pause 200ms]\n"), "openai", "")
	if err != nil || len(pause) != 1 || pause[0].Pause != 200*time.Millisecond {
		t.Fatalf("pause only: %+v %v", pause, err)
	}
}

func TestPlanPauseUnitsAndAliases(t *testing.T) {
	for _, tt := range []struct {
		arg  string
		want time.Duration
	}{
		{"", 250 * time.Millisecond},
		{"short", 250 * time.Millisecond},
		{"medium", 500 * time.Millisecond},
		{"long", time.Second},
		{"250ms", 250 * time.Millisecond},
		{"1.25s", 1250 * time.Millisecond},
		{"0.5min", 30 * time.Second},
		{"0ms", 0},
	} {
		segs, err := providers.Plan(document(t, "Hello [pause "+tt.arg+"]"), "openai", "coral")
		if err != nil {
			t.Fatalf("%q: %v", tt.arg, err)
		}
		if got := segs[len(segs)-1].Pause; got != tt.want {
			t.Errorf("%q: pause=%v, want %v", tt.arg, got, tt.want)
		}
	}
}

func TestPlanRepeatedAutoSpeakerForwardReference(t *testing.T) {
	const count = 2000
	src := "---\ncast: auto\n---\n" + strings.Repeat("[as last|Hello]\n\n", count) + "last: Bye\n"
	segments, err := providers.Plan(document(t, src), "openai", "coral")
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != count+1 {
		t.Fatalf("segments = %d, want %d", len(segments), count+1)
	}
	for i, segment := range segments {
		if segment.Voice != "coral" {
			t.Fatalf("segment %d voice = %q", i, segment.Voice)
		}
	}
}

func BenchmarkPlanAutoSpeakerForwardReferences(b *testing.B) {
	for _, count := range []int{1000, 8000, 64000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			src := "---\ncast: auto\n---\n" + strings.Repeat("[as last|Hello]\n\n", count) + "last: Bye\n"
			doc := document(b, src)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if _, err := providers.Plan(doc, "openai", "coral"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
