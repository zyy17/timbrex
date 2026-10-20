package lower

import (
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/parse"
)

func TestNarratorConfig(t *testing.T) {
	src := "---\nnarrator: host\ncast:\n  lin: { name: Lin }\n---\nHello\nlin: Hi\n"
	script, err := parse.Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if script.Front.Narrator != "host" {
		t.Fatalf("narrator = %q", script.Front.Narrator)
	}
	doc := Script(script)
	if len(doc.Utterances) != 2 || doc.Utterances[0].Speaker != "host" || doc.Utterances[1].Speaker != "lin" {
		t.Fatalf("doc = %+v", doc)
	}
}

func TestLowerHello(t *testing.T) {
	src := "---\ncast:\n  lin: { name: Lin }\n---\nlin: [laugh] Hi [whisper|there]\n"
	script, err := parse.Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	doc := Script(script)
	if len(doc.Utterances) != 1 || doc.Utterances[0].Speaker != "lin" {
		t.Fatalf("doc = %+v", doc)
	}
	kinds := make([]ir.Kind, len(doc.Utterances[0].Nodes))
	for i, node := range doc.Utterances[0].Nodes {
		kinds[i] = node.Kind
	}
	want := []ir.Kind{ir.Event, ir.Text, ir.SpanStart, ir.Text, ir.SpanEnd}
	if len(kinds) != len(want) {
		t.Fatalf("kinds = %v", kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kinds = %v, want %v", kinds, want)
		}
	}
	if doc.Utterances[0].Nodes[0].Name != "laugh" {
		t.Fatalf("event = %+v", doc.Utterances[0].Nodes[0])
	}
}

func TestLowerSceneFenceAndArgs(t *testing.T) {
	src := "" +
		"---\nprovider: stub\n---\n" +
		"// dropped\n" +
		"[bgm rain.wav vol=-22dB]\n" +
		"She waited [act \"holding back tears\"]\n" +
		"```ssml azure\n<speak>hi</speak>\n```\n"
	script, err := parse.Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	doc := Script(script)
	if doc.Provider != "stub" || len(doc.Utterances) != 3 {
		t.Fatalf("doc = %+v", doc)
	}
	scene := doc.Utterances[0].Nodes
	if len(scene) != 1 || scene[0].Kind != ir.Event || scene[0].Scope != "scene" || scene[0].Args != "rain.wav vol=-22dB" {
		t.Fatalf("scene = %+v", scene)
	}
	act := doc.Utterances[1].Nodes
	if doc.Utterances[1].Speaker != "narrator" || len(act) != 2 || act[1].Kind != ir.State || act[1].Scope != "turn" || act[1].Args != `"holding back tears"` {
		t.Fatalf("act = %+v speaker %q", act, doc.Utterances[1].Speaker)
	}
	raw := doc.Utterances[2].Nodes
	if len(raw) != 1 || raw[0].Kind != ir.Raw || raw[0].Name != "ssml azure" || raw[0].Text != "<speak>hi</speak>\n" {
		t.Fatalf("raw = %+v", raw)
	}
}

func TestLowerTurnTagsAreTurnScope(t *testing.T) {
	src := "---\ncast:\n  lin: { name: Lin }\n---\n[overlap -200ms]\nlin: [gap 1s] Hello\n[pause]\n"
	script, err := parse.Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if diags := check.Apply(script, false); check.Failed(diags) != nil {
		t.Fatal(diags)
	}
	doc := Script(script)
	if len(doc.Utterances) != 3 {
		t.Fatalf("doc = %+v", doc)
	}
	overlap := doc.Utterances[0].Nodes[0]
	if overlap.Kind != ir.Event || overlap.Name != "overlap" || overlap.Scope != "turn" || overlap.Args != "-200ms" {
		t.Fatalf("overlap = %+v", overlap)
	}
	gap := doc.Utterances[1].Nodes[0]
	if doc.Utterances[1].Speaker != "lin" || gap.Kind != ir.Event || gap.Name != "gap" || gap.Scope != "turn" || gap.Args != "1s" {
		t.Fatalf("gap = %+v", gap)
	}
	if strings.TrimSpace(doc.Utterances[1].Nodes[1].Text) != "Hello" {
		t.Fatalf("turn = %+v", doc.Utterances[1].Nodes)
	}
	pause := doc.Utterances[2].Nodes[0]
	if pause.Kind != ir.Event || pause.Scope != "scene" || pause.Name != "pause" {
		t.Fatalf("pause = %+v", pause)
	}
}

func TestLowerCanonicalName(t *testing.T) {
	script, err := parse.Parse("t.tbx", "Say [PAUSE 1s]\n")
	if err != nil {
		t.Fatal(err)
	}
	doc := Script(script)
	ev := doc.Utterances[0].Nodes[1]
	if ev.Kind != ir.Event || ev.Name != "pause" || ev.Scope != "here" || ev.Args != "1s" {
		t.Fatalf("event = %+v", ev)
	}
}

func TestLowerStateEventAndSpanNote(t *testing.T) {
	src := "[slow]\n[pause]\nShe [laugh nervously|while laughing] now\n"
	script, err := parse.Parse("t.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if diags := check.Apply(script, false); check.Failed(diags) != nil {
		t.Fatal(diags)
	}
	doc := Script(script)
	if len(doc.Utterances) != 3 {
		t.Fatalf("doc = %+v", doc)
	}
	slow := doc.Utterances[0].Nodes[0]
	if slow.Kind != ir.State || slow.Name != "slow" || slow.Scope != "scene" {
		t.Fatalf("slow = %+v", slow)
	}
	pause := doc.Utterances[1].Nodes[0]
	if pause.Kind != ir.Event || pause.Name != "pause" || pause.Scope != "scene" {
		t.Fatalf("pause = %+v", pause)
	}
	laugh := doc.Utterances[2].Nodes[1]
	if doc.Utterances[2].Speaker != "narrator" || laugh.Kind != ir.SpanStart || laugh.Note != "nervously" || laugh.Name != "laugh" {
		t.Fatalf("laugh = %+v", laugh)
	}
	if doc.Utterances[2].Nodes[2].Text != "while laughing" || doc.Utterances[2].Nodes[3].Kind != ir.SpanEnd {
		t.Fatalf("span = %+v", doc.Utterances[2].Nodes)
	}
}
