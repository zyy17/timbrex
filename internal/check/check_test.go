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
