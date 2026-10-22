package providers_test

import (
	"strings"
	"testing"
	"time"

	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/lower"
	"github.com/zyy17/timbrex/internal/parse"
	"github.com/zyy17/timbrex/internal/providers"
)

func document(t *testing.T, src string) *ir.Document {
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
	for _, src := range []string{"Hi [pause -1s]", "Hi [pause 61s]", "Hi [pause 1 2]", "Hi [pause NaN]", "```ssml azure\n<speak/>\n```", "// empty"} {
		if _, err := providers.Plan(document(t, src), "openai", "coral"); err == nil {
			t.Fatalf("accepted %q", src)
		}
	}
}
