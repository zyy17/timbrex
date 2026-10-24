package fish

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func TestS2ModelsAndAct(t *testing.T) {
	for _, model := range []string{"s2.1-pro", "s2.1-pro-free", "s2-pro"} {
		t.Run(model, func(t *testing.T) {
			for _, direction := range []string{"whispers sweetly", "轻轻地", `say "yes" quietly`, "warm (not excited)", "use ［gentle］ phrasing"} {
				for _, note := range []bool{false, true} {
					control := ir.Node{Name: "act", Args: []ir.Arg{{Raw: direction, Kind: "quoted"}}}
					if note {
						control.Args = nil
						control.Note = direction
					}
					seg := providers.Segment{Voice: "test-reference", Nodes: []ir.Node{{Text: "Hello"}}, Controls: []ir.Node{control}}
					r, err := build(seg, model)
					if err != nil {
						t.Fatalf("direction %q note=%v: %v", direction, note, err)
					}
					want := request{Text: "[" + direction + "] Hello", ReferenceID: "test-reference", Format: "wav", SampleRate: 24000, Normalize: false, Prosody: prosody{Speed: 1, Volume: 0, NormalizeLoudness: true}}
					if r != want {
						t.Fatalf("got %+v, want %+v", r, want)
					}
					data, err := json.Marshal(r)
					if err != nil || !json.Valid(data) {
						t.Fatalf("JSON %q: %v", data, err)
					}
				}
			}
		})
	}
}

func TestActRejectsInvalidDirections(t *testing.T) {
	controls := []ir.Node{
		{Name: "act"},
		{Name: "act", Args: []ir.Arg{{Raw: "", Kind: "quoted"}}},
		{Name: "act", Args: []ir.Arg{{Raw: " \t\n\u3000", Kind: "quoted"}}},
		{Name: "act", Note: " \t\n\u3000"},
		{Name: "act", Args: []ir.Arg{{Key: "direction", Raw: "quietly", Kind: "quoted"}}},
		{Name: "act", Args: []ir.Arg{{Raw: "quietly"}, {Raw: "slowly"}}},
		{Name: "act", Args: []ir.Arg{{Raw: "quietly"}}, Note: "slowly"},
	}
	for _, text := range []string{"[laughing]", "quiet] loudly", "[quiet", "<|speaker:0|>", "<|phoneme_start|>", `\[laughing\]`} {
		controls = append(controls,
			ir.Node{Name: "act", Args: []ir.Arg{{Raw: text, Kind: "quoted"}}},
			ir.Node{Name: "act", Note: text})
	}
	for _, model := range []string{"s2.1-pro", "s2.1-pro-free", "s2-pro"} {
		for _, control := range controls {
			seg := providers.Segment{Voice: "test-reference", Nodes: []ir.Node{{Text: "Hello"}}, Controls: []ir.Node{control}}
			if _, err := build(seg, model); err == nil {
				t.Errorf("%s accepted %+v", model, control)
			}
		}
	}
}

func TestActCountsTowardTextLimit(t *testing.T) {
	for _, model := range []string{"s2.1-pro", "s2.1-pro-free", "s2-pro"} {
		control := ir.Node{Name: "act", Args: []ir.Arg{{Raw: "轻轻地", Kind: "quoted"}}}
		seg := providers.Segment{Voice: "test-reference", Controls: []ir.Node{control}, Nodes: []ir.Node{{Text: strings.Repeat("中", 4096-utf8.RuneCountInString("[轻轻地] "))}}}
		if _, err := build(seg, model); err != nil {
			t.Fatalf("%s: %v", model, err)
		}
		seg.Nodes[0].Text += "中"
		if _, err := build(seg, model); err == nil {
			t.Errorf("%s accepted oversized text with acting cue", model)
		}
	}
}
