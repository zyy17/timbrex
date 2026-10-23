package fish

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func fixtureWAV() []byte {
	b, _ := hex.DecodeString("524946462800000057415645666d74201000000001000100c05d000080bb000002001000646174610400000001000200")
	return b
}

func TestRequestJSONMatchesOfficialFields(t *testing.T) {
	r, err := build(providers.Segment{Voice: "test-reference", Nodes: []ir.Node{{Text: "Hello"}}}, "s2.1-pro")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"language", "references", "temperature", "top_p", "latency"} {
		if _, ok := obj[key]; ok {
			t.Fatalf("unexpected field %s: %s", key, data)
		}
	}
	if obj["text"] != "Hello" || obj["reference_id"] != "test-reference" || obj["format"] != "wav" || obj["sample_rate"] != float64(24000) || obj["normalize"] != false {
		t.Fatalf("body=%s", data)
	}
	prosody, _ := obj["prosody"].(map[string]any)
	if prosody["speed"] != float64(1) || prosody["volume"] != float64(0) || prosody["normalize_loudness"] != true {
		t.Fatalf("prosody=%v", prosody)
	}
}

func TestBuildContract(t *testing.T) {
	seg := providers.Segment{Voice: "test-reference", Nodes: []ir.Node{{Text: "Hello (world) "}, {Kind: ir.Event, Name: "sigh"}}, Controls: []ir.Node{{Name: "happy"}, {Name: "whisper"}, {Name: "speed", Args: []ir.Arg{{Raw: "125%", Kind: "percent"}}}, {Name: "volume", Args: []ir.Arg{{Raw: "-6dB", Kind: "decibel"}}}}}
	r, err := build(seg, "s2.1-pro")
	if err != nil || r.Text != "[happy] [whispering] Hello (world) [sighing]" || r.ReferenceID != "test-reference" || r.Prosody.Speed != 1.25 || r.Prosody.Volume != -6 || !r.Prosody.NormalizeLoudness || r.Normalize || r.Format != "wav" || r.SampleRate != 24000 {
		t.Fatalf("%+v %v", r, err)
	}
	for name, speed := range map[string]float64{"fast": 1.2, "slow": 0.8} {
		seg.Controls = []ir.Node{{Name: name}}
		r, err := build(seg, "s2.1-pro")
		if err != nil || r.Prosody.Speed != speed {
			t.Fatalf("%s: %+v %v", name, r, err)
		}
	}
	for name, cue := range cues {
		seg.Controls = []ir.Node{{Name: name}}
		r, err := build(seg, "s2.1-pro")
		if err != nil || !strings.HasPrefix(r.Text, "["+cue+"] ") {
			t.Fatalf("cue %s: %v", name, err)
		}
	}
	for name, cue := range events {
		seg.Controls = nil
		seg.Nodes = []ir.Node{{Kind: ir.Event, Name: name}}
		r, err := build(seg, "s2.1-pro")
		if err != nil || r.Text != cue {
			t.Fatalf("event %s: %v", name, err)
		}
	}
}

func TestBuildRejectsUnsupported(t *testing.T) {
	base := providers.Segment{Voice: "test-reference", Nodes: []ir.Node{{Text: "Hello"}}}
	for _, model := range []string{"s1", "s2-pro", "s2.1-pro-free", "drama-3", "unknown", ""} {
		if _, err := build(base, model); err == nil {
			t.Fatalf("accepted model %q", model)
		}
	}
	for _, tc := range []struct {
		name   string
		change func(*providers.Segment)
	}{
		{"reference", func(s *providers.Segment) { s.Voice = " " }},
		{"lang", func(s *providers.Segment) { s.Lang = "en-US" }},
		{"brackets", func(s *providers.Segment) { s.Nodes = []ir.Node{{Text: "[happy]"}} }},
		{"speaker tag", func(s *providers.Segment) { s.Nodes = []ir.Node{{Text: "<|speaker:0|>"}} }},
		{"event", func(s *providers.Segment) { s.Nodes = []ir.Node{{Kind: ir.Event, Name: "cough"}} }},
		{"event args", func(s *providers.Segment) { s.Nodes = []ir.Node{{Kind: ir.Event, Name: "laugh", Note: "quietly"}} }},
		{"intensity", func(s *providers.Segment) {
			s.Controls = []ir.Node{{Name: "happy", Args: []ir.Arg{{Raw: "0.8", Kind: "number"}}}}
		}},
		{"pitch", func(s *providers.Segment) {
			s.Controls = []ir.Node{{Name: "pitch", Args: []ir.Arg{{Raw: "2st", Kind: "semitone"}}}}
		}},
		{"act", func(s *providers.Segment) { s.Controls = []ir.Node{{Name: "act", Note: "softly"}} }},
		{"speed low", func(s *providers.Segment) {
			s.Controls = []ir.Node{{Name: "speed", Args: []ir.Arg{{Raw: "0.49", Kind: "number"}}}}
		}},
		{"speed high", func(s *providers.Segment) {
			s.Controls = []ir.Node{{Name: "speed", Args: []ir.Arg{{Raw: "2.01", Kind: "number"}}}}
		}},
		{"volume", func(s *providers.Segment) {
			s.Controls = []ir.Node{{Name: "volume", Args: []ir.Arg{{Raw: "21dB", Kind: "decibel"}}}}
		}},
		{"size", func(s *providers.Segment) { s.Nodes = []ir.Node{{Text: strings.Repeat("中", 4097)}} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seg := base
			tc.change(&seg)
			if _, err := build(seg, "s2.1-pro"); err == nil {
				t.Fatal("accepted unsupported input")
			}
		})
	}
	base.Nodes = []ir.Node{{Text: strings.Repeat("中", 4096)}}
	if _, err := build(base, "s2.1-pro"); err != nil {
		t.Fatal(err)
	}
	base.Controls = []ir.Node{{Name: "happy"}}
	if _, err := build(base, "s2.1-pro"); err == nil {
		t.Fatal("cue omitted from length limit")
	}
}

func TestFailurePreservesOutput(t *testing.T) {
	for _, kind := range []string{"json error", "http 400", "http 401", "http 402", "http 429", "http 503", "partial", "redirect", "truncated", "streaming size", "wrong rate", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				body := fixtureWAV()
				switch kind {
				case "json error":
					body = []byte(`{"message":"FAKE-KEY private-text","status":401}`)
				case "http 400", "http 401", "http 402", "http 429", "http 503":
					w.Header().Set("Content-Type", "application/json")
					switch kind {
					case "http 400":
						w.WriteHeader(http.StatusBadRequest)
					case "http 401":
						w.WriteHeader(http.StatusUnauthorized)
					case "http 402":
						w.WriteHeader(http.StatusPaymentRequired)
					case "http 429":
						w.WriteHeader(http.StatusTooManyRequests)
					default:
						w.WriteHeader(http.StatusServiceUnavailable)
					}
					body = []byte(`{"message":"FAKE-KEY private-text","status":` + strings.TrimPrefix(kind, "http ") + `}`)
				case "partial":
					w.WriteHeader(206)
				case "redirect":
					w.Header().Set("Location", "/private")
					w.WriteHeader(307)
				case "truncated":
					body = body[:len(body)-2]
				case "streaming size":
					copy(body[4:8], []byte{255, 255, 255, 255})
				case "wrong rate":
					body[24] = 1
				case "cancel":
					_, _ = io.Copy(io.Discard, r.Body)
					cancel()
					select {
					case <-r.Context().Done():
					case <-time.After(2 * time.Second):
						t.Error("cancel not observed")
					}
					return
				}
				_, _ = w.Write(body)
			}))
			defer srv.Close()
			out := filepath.Join(t.TempDir(), "out.wav")
			if err := os.WriteFile(out, []byte("KEEP"), 0600); err != nil {
				t.Fatal(err)
			}
			doc := &ir.Document{Utterances: []ir.Utterance{{Nodes: []ir.Node{{Text: "private-text"}}}}}
			_, err := (provider{}).Synthesize(ctx, doc, providers.Options{APIKey: "FAKE-KEY", Model: "s2.1-pro", Voice: "test-reference", Endpoint: srv.URL, Out: out})
			if err == nil || calls.Load() != 1 || strings.Contains(err.Error(), "FAKE-KEY") || strings.Contains(err.Error(), "private-text") {
				t.Fatalf("err=%v calls=%d", err, calls.Load())
			}
			if kind == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancel chain: %v", err)
			}
			b, _ := os.ReadFile(out)
			if !bytes.Equal(b, []byte("KEEP")) {
				t.Fatal("replaced old output")
			}
		})
	}
}
