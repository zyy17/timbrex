package providers

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/zyy17/timbrex/internal/ir"
)

// Segment is a run with one voice and one set of controls, or a local pause.
type Segment struct {
	Voice    string
	Lang     string
	Controls []ir.Node
	Nodes    []ir.Node
	Pause    time.Duration
}

// Plan resolves scene, turn and nested span scopes before any paid requests.
func Plan(doc *ir.Document, id, defaultVoice string) ([]Segment, error) {
	if doc == nil {
		return nil, fmt.Errorf("empty document")
	}
	var out []Segment
	var scene []ir.Node
	for _, utt := range doc.Utterances {
		state := slices.Clone(scene)
		base := scene
		if utt.Speaker == "" {
			base = nil
		}
		var stack [][]ir.Node
		var nodes []ir.Node
		flush := func() error {
			if !speakable(nodes) {
				nodes = nil
				return nil
			}
			seg := Segment{Voice: voiceFor(doc, id, utt.Speaker, defaultVoice), Lang: doc.Lang, Nodes: nodes}
			for _, n := range state {
				switch n.Name {
				case "voice", "as", "lang":
					v, err := Value(n)
					if err != nil {
						return err
					}
					switch n.Name {
					case "voice":
						seg.Voice = v
					case "as":
						seg.Voice = voiceFor(doc, id, v, defaultVoice)
					case "lang":
						seg.Lang = v
					}
				default:
					seg.Controls = append(seg.Controls, n)
				}
			}
			out = append(out, seg)
			nodes = nil
			return nil
		}
		for _, n := range utt.Nodes {
			switch n.Kind {
			case ir.Text:
				nodes = append(nodes, n)
			case ir.Raw:
				return nil, fmt.Errorf("raw fence %q is not supported by %s", n.Name, id)
			case ir.State, ir.SpanStart:
				if err := flush(); err != nil {
					return nil, err
				}
				if n.Kind == ir.SpanStart {
					stack = append(stack, slices.Clone(state))
				}
				var err error
				state, err = setControl(state, base, n)
				if err != nil {
					return nil, err
				}
			case ir.SpanEnd:
				if err := flush(); err != nil {
					return nil, err
				}
				if len(stack) == 0 {
					return nil, fmt.Errorf("unmatched span end %q", n.Name)
				}
				state = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			case ir.Event:
				if n.Name != "pause" && n.Name != "silence" && n.Name != "break" && n.Name != "gap" {
					nodes = append(nodes, n)
					continue
				}
				if err := flush(); err != nil {
					return nil, err
				}
				d, err := pauseDuration(n)
				if err != nil {
					return nil, err
				}
				if d > 0 {
					out = append(out, Segment{Pause: d})
				}
			}
		}
		if len(stack) != 0 {
			return nil, fmt.Errorf("unclosed span in IR")
		}
		if err := flush(); err != nil {
			return nil, err
		}
		if utt.Speaker == "" {
			scene = state
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("script has no speech or pauses")
	}
	return out, nil
}

func voiceFor(doc *ir.Document, id, speaker, fallback string) string {
	if v := doc.Voices[speaker][id]; v != "" {
		return v
	}
	return fallback
}

func speakable(nodes []ir.Node) bool {
	for _, n := range nodes {
		if n.Kind != ir.Text || strings.TrimSpace(n.Text) != "" {
			return true
		}
	}
	return false
}

func setControl(state, base []ir.Node, n ir.Node) ([]ir.Node, error) {
	key := feature(n.Name)
	if n.Name == "reset" {
		if len(n.Args) == 0 {
			return slices.Clone(base), nil
		}
		v, err := Value(n)
		if err != nil {
			return nil, err
		}
		key = feature(v)
	}
	out := make([]ir.Node, 0, len(state)+1)
	for _, prev := range state {
		if feature(prev.Name) != key {
			out = append(out, prev)
		}
	}
	if n.Name != "reset" {
		out = append(out, n)
	} else {
		for _, prev := range base {
			if feature(prev.Name) == key {
				out = append(out, prev)
			}
		}
	}
	return out, nil
}

func feature(name string) string {
	switch name {
	case "fast", "slow", "speed":
		return "speed"
	case "voice", "as":
		return "voice"
	case "whisper", "shout":
		return "delivery"
	case "emotion", "neutral", "happy", "sad", "angry", "fear", "surprise", "disgust", "calm",
		"excited", "tender", "serious", "curious", "sarcastic", "nervous", "tired", "bored", "contempt",
		"confused", "proud", "embarrassed", "hopeful", "disappointed", "panicked", "crying":
		return "emotion"
	default:
		return name
	}
}

func pauseDuration(n ir.Node) (time.Duration, error) {
	v := "250ms"
	if len(n.Args) > 0 {
		var err error
		v, err = Value(n)
		if err != nil {
			return 0, err
		}
	}
	switch v {
	case "short":
		v = "250ms"
	case "medium":
		v = "500ms"
	case "long":
		v = "1s"
	}
	if strings.HasSuffix(v, "min") {
		v = strings.TrimSuffix(v, "min") + "m"
	}
	d, err := time.ParseDuration(v)
	if err != nil || d < 0 || d > time.Minute {
		return 0, fmt.Errorf("[%s] needs a duration from 0ms to 1min", n.Name)
	}
	return d, nil
}
