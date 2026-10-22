package minimax

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAudioMetadataMismatchPreservesOutput(t *testing.T) {
	p := provider{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"base_resp":{"status_code":0},"data":{"status":2,"audio":"00000000"},"extra_info":{"audio_sample_rate":16000}}`))}, nil
	})}}
	out := filepath.Join(t.TempDir(), "out.wav")
	if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
		t.Fatal(err)
	}
	doc := &ir.Document{Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}}}
	_, err := p.Synthesize(context.Background(), doc, providers.Options{Out: out, Voice: "test-voice", Model: "speech-2.8-hd", Endpoint: "https://example.invalid"})
	if err == nil || !strings.Contains(err.Error(), "sample rate") {
		t.Fatalf("error=%v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "KEEP" {
		t.Fatalf("output=%q error=%v", data, err)
	}
	partials, err := filepath.Glob(filepath.Join(filepath.Dir(out), ".timbrex-*.partial"))
	if err != nil || len(partials) != 0 {
		t.Fatalf("temporary files=%v error=%v", partials, err)
	}
}

func TestRejectContradictoryAudioMetadata(t *testing.T) {
	for _, info := range []string{
		`{"audio_sample_rate":16000,"audio_channel":1,"audio_format":"pcm"}`,
		`{"audio_sample_rate":24000,"audio_channel":2,"audio_format":"pcm"}`,
		`{"audio_sample_rate":24000,"audio_channel":1,"audio_format":"mp3"}`,
		`{"audio_sample_rate":0}`,
		`{"audio_channel":0}`,
		`{"audio_format":""}`,
	} {
		body := fmt.Sprintf(`{"base_resp":{"status_code":0},"data":{"status":2,"audio":"00000000"},"extra_info":%s}`, info)
		if _, err := decode([]byte(body)); err == nil {
			t.Errorf("accepted contradictory audio metadata: %s", info)
		}
	}
	for _, info := range []string{`null`, `{}`, `{"audio_sample_rate":24000,"audio_channel":1,"audio_format":"pcm"}`} {
		body := fmt.Sprintf(`{"base_resp":{"status_code":0},"data":{"status":2,"audio":"00000000"},"extra_info":%s}`, info)
		pcm, err := decode([]byte(body))
		if err != nil || len(pcm) != 4 {
			t.Errorf("metadata %s: pcm=%x, err=%v", info, pcm, err)
		}
	}
}

func TestAPIErrorAndMalformedAudio(t *testing.T) {
	for _, body := range []string{`{`, `{}`, `{"base_resp":{},"data":{"status":2,"audio":"0000"}}`, `{"base_resp":{"status_code":null},"data":{"status":2,"audio":"0000"}}`, `{"base_resp":{"status_code":1008}}`, `{"base_resp":{"status_code":0},"data":{"status":1,"audio":"0000"}}`, `{"base_resp":{"status_code":0},"data":{"status":2,"audio":"zz"}}`} {
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
