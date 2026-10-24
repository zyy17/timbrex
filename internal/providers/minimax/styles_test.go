package minimax

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func TestFluentRequires26(t *testing.T) {
	for _, model := range modelCases {
		for _, kind := range []string{"word", "quoted"} {
			t.Run(model.name+"/"+kind, func(t *testing.T) {
				seg := providers.Segment{Voice: "voice", Nodes: []ir.Node{{Text: "Hello"}}, Controls: []ir.Node{{Name: "style", Args: []ir.Arg{{Raw: "fluent", Kind: kind}}}}}
				r, err := build(seg, model.name)
				if model.name == "speech-2.6-hd" || model.name == "speech-2.6-turbo" {
					if err != nil || r.Voice.Emotion != "fluent" || r.Text != "Hello" {
						t.Fatalf("request=%+v error=%v", r, err)
					}
				} else if err == nil || !strings.Contains(err.Error(), "requires speech-2.6") {
					t.Fatalf("accepted fluent on %s: error=%v", model.name, err)
				}
			})
		}
	}
}

func TestFluentRejectsMalformedOrUnknownStyles(t *testing.T) {
	for _, n := range []ir.Node{
		{Name: "style"}, {Name: "style", Args: []ir.Arg{{Raw: ""}}},
		{Name: "style", Args: []ir.Arg{{Raw: "Fluent"}}},
		{Name: "style", Args: []ir.Arg{{Raw: "whisper"}}},
		{Name: "style", Args: []ir.Arg{{Raw: "fluent "}}},
		{Name: "style", Args: []ir.Arg{{Raw: "unknown"}}},
		{Name: "style", Args: []ir.Arg{{Raw: "fluent"}, {Raw: "fluent"}}},
		{Name: "style", Args: []ir.Arg{{Key: "name", Raw: "fluent"}}},
		{Name: "style", Args: []ir.Arg{{Raw: "fluent"}}, Note: "quickly"},
		{Name: "fluent"},
	} {
		seg := providers.Segment{Voice: "voice", Nodes: []ir.Node{{Text: "Hello"}}, Controls: []ir.Node{n}}
		if _, err := build(seg, "speech-2.6-hd"); err == nil {
			t.Errorf("accepted unsupported style %+v", n)
		}
	}
}

func TestFluentConflictsWithEmotionAndWhisper(t *testing.T) {
	fluent := ir.Node{Name: "style", Args: []ir.Arg{{Raw: "fluent"}}}
	for _, name := range []string{"happy", "sad", "angry", "fear", "disgust", "surprise", "calm", "whisper"} {
		for _, controls := range [][]ir.Node{{fluent, {Name: name}}, {{Name: name}, fluent}} {
			seg := providers.Segment{Voice: "voice", Nodes: []ir.Node{{Text: "Hello"}}, Controls: controls}
			if _, err := build(seg, "speech-2.6-turbo"); err == nil || !strings.Contains(err.Error(), "cannot combine") {
				t.Errorf("accepted incompatible controls %+v: error=%v", controls, err)
			}
		}
	}
}

func TestLateInvalidFluentMakesNoRequests(t *testing.T) {
	for _, concurrency := range []int{1, 3} {
		for _, model := range []string{"speech-2.8-hd", "speech-2.6-hd", "speech-01-turbo"} {
			t.Run(fmt.Sprintf("%s/concurrency=%d", model, concurrency), func(t *testing.T) {
				nodes := []ir.Node{{Kind: ir.State, Name: "style", Args: []ir.Arg{{Raw: "fluent"}}}}
				if model == "speech-2.6-hd" {
					nodes = append(nodes, ir.Node{Kind: ir.State, Name: "happy"})
				}
				nodes = append(nodes, ir.Node{Text: "Late invalid"})
				assertLatePreflightFailure(t, model, concurrency, nodes)
			})
		}
	}
}
