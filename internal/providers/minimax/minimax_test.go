package minimax

import (
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func TestAPIErrorAndMalformedAudio(t *testing.T) {
	for _, body := range []string{`{`, `{}`, `{"base_resp":{"status_code":1008}}`, `{"base_resp":{"status_code":0},"data":{"status":1,"audio":"0000"}}`, `{"base_resp":{"status_code":0},"data":{"status":2,"audio":"zz"}}`} {
		if _, err := decode([]byte(body)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestWhisperAndEventsAreModelSpecific(t *testing.T) {
	seg := providers.Segment{Voice: "voice", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: []ir.Node{{Name: "whisper"}}}
	if _, err := build(seg, "speech-2.8-hd"); err == nil {
		t.Fatal("2.8 does not support whisper")
	}
	if r, err := build(seg, "speech-2.6-hd"); err != nil || r.Voice.Emotion != "whisper" {
		t.Fatalf("request=%+v err=%v", r, err)
	}
	seg.Controls = nil
	seg.Nodes = append(seg.Nodes, ir.Node{Kind: ir.Event, Name: "laugh"})
	if _, err := build(seg, "speech-2.6-hd"); err == nil {
		t.Fatal("nonverbal should require 2.8")
	}
	if r, err := build(seg, "speech-2.8-hd"); err != nil || r.Text != "Hello(laughs)" {
		t.Fatalf("request=%+v err=%v", r, err)
	}
}

func TestControlsAndLanguages(t *testing.T) {
	seg := providers.Segment{Voice: "voice", Lang: "zh-HK", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: []ir.Node{{Name: "pitch", Args: []ir.Arg{{Raw: "+2st", Kind: "semitone"}}}, {Name: "volume", Args: []ir.Arg{{Raw: "0dB", Kind: "decibel"}}}}}
	r, err := build(seg, "speech-2.8-hd")
	if err != nil || r.Language != "Chinese,Yue" || r.Voice.Pitch != 2 || r.Voice.Volume != 1 {
		t.Fatalf("%+v: %v", r, err)
	}
	seg.Controls[0].Args[0].Raw = "+0.5st"
	if _, err := build(seg, r.Model); err == nil {
		t.Fatal("accepted fractional pitch")
	}
	for _, controls := range [][]ir.Node{{{Name: "whisper"}, {Name: "happy"}}, {{Name: "happy"}, {Name: "whisper"}}} {
		seg.Controls = controls
		if _, err := build(seg, "speech-2.6-hd"); err == nil {
			t.Fatal("silently overwrote emotion or whisper")
		}
	}
	seg.Controls = nil
	seg.Lang = "unknown"
	if _, err := build(seg, r.Model); err == nil {
		t.Fatal("silently ignored language hint")
	}
}

func TestLiteralParenthesesAndReservedEvents(t *testing.T) {
	seg := providers.Segment{Voice: "voice", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello (again)."}}}
	if r, err := build(seg, "speech-2.8-hd"); err != nil || r.Text != "Hello (again)." {
		t.Fatalf("%+v: %v", r, err)
	}
	for _, text := range []string{"Literal (laughs)", "Hello <#0.5#>"} {
		seg.Nodes[0].Text = text
		if _, err := build(seg, "speech-2.8-hd"); err == nil {
			t.Fatalf("accepted native markup %q", text)
		}
	}
}
