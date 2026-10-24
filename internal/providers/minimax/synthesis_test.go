package minimax

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

func TestSynthesisCompleteRequest(t *testing.T) {
	mode := options.MiniMaxNormalizationQuality
	seg := providers.Segment{Voice: "voice", Lang: "zh-CN", Nodes: []ir.Node{{Kind: ir.Text, Text: "今天是10月24日。"}}, Controls: []ir.Node{
		{Name: "speed", Args: []ir.Arg{{Kind: "number", Raw: "1.1"}}},
		{Name: "pitch", Args: []ir.Arg{{Kind: "semitone", Raw: "+2st"}}},
		{Name: "volume", Args: []ir.Arg{{Kind: "decibel", Raw: "0dB"}}},
		{Name: "happy"},
	}}
	r, err := buildWithSynthesis(seg, "speech-2.8-hd", options.MiniMax{TextNormalizationMode: &mode})
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"model":"speech-2.8-hd","text":"今天是10月24日。","stream":false,"output_format":"hex","language_boost":"Chinese","voice_setting":{"voice_id":"voice","speed":1.1,"vol":1,"pitch":2,"emotion":"happy"},"audio_setting":{"sample_rate":24000,"format":"pcm","channel":1},"text_normalization_mode":"quality"}`
	if got := requestJSON(t, r); got != want {
		t.Fatalf("body = %s; want %s", got, want)
	}
}

func TestSynthesisOmissionAndModelEligibility(t *testing.T) {
	seg := providers.Segment{Voice: "voice", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}
	for _, model := range []string{"speech-2.8-hd", "speech-2.8-turbo", "speech-2.6-hd", "speech-2.6-turbo", "speech-02-hd", "speech-02-turbo", "speech-01-hd", "speech-01-turbo"} {
		base, err := build(seg, model)
		if err != nil {
			t.Fatal(err)
		}
		r, err := buildWithSynthesis(seg, model, options.MiniMax{})
		if err != nil || requestJSON(t, r) != requestJSON(t, base) {
			t.Fatalf("omission changed %s: %+v, %v", model, r, err)
		}
		for _, mode := range []options.MiniMaxTextNormalization{options.MiniMaxNormalizationBasic, options.MiniMaxNormalizationQuality} {
			value, err := synthesisOptions(options.MiniMax{TextNormalizationMode: &mode}, model)
			legacy := strings.HasPrefix(model, "speech-0")
			if mode == options.MiniMaxNormalizationQuality && legacy {
				if err == nil || !strings.Contains(err.Error(), "quality requires") {
					t.Fatalf("quality should not silently downgrade on %s: %v", model, err)
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			r, err := buildWithSynthesis(seg, model, value)
			if err != nil || r.Stream || r.TextNormalizationMode == nil || *r.TextNormalizationMode != mode {
				t.Fatalf("%s %s: %+v, %v", model, mode, r, err)
			}
		}
	}
	quality := options.MiniMaxNormalizationQuality
	for _, model := range []string{"", "speech-2.8-HD", "speech-2.8", "speech-2.8-hd-preview", "speech-2.9-hd", "speech-02.8-hd"} {
		if _, err := synthesisOptions(options.MiniMax{TextNormalizationMode: &quality}, model); err == nil {
			t.Fatalf("quality allowed for unverified model %q", model)
		}
	}
}

func TestSynthesisSnapshots(t *testing.T) {
	mode := options.MiniMaxNormalizationQuality
	value := options.MiniMax{TextNormalizationMode: &mode}
	seg := providers.Segment{Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}
	first, err := buildWithSynthesis(seg, "speech-2.8-hd", value)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildWithSynthesis(seg, "speech-2.8-hd", value)
	if err != nil {
		t.Fatal(err)
	}
	mode = options.MiniMaxNormalizationBasic
	if *first.TextNormalizationMode != options.MiniMaxNormalizationQuality || *second.TextNormalizationMode != options.MiniMaxNormalizationQuality {
		t.Fatal("request shares caller option")
	}
	*first.TextNormalizationMode = "changed"
	if *second.TextNormalizationMode != options.MiniMaxNormalizationQuality || mode != options.MiniMaxNormalizationBasic {
		t.Fatal("request mutation changed another request or caller")
	}
}

func TestSynthesisInvalidOptionsAndLastSegmentPreflight(t *testing.T) {
	quality := options.MiniMaxNormalizationQuality
	invalid := options.MiniMaxTextNormalization("Quality")
	for _, tc := range []struct {
		name  string
		value options.Value
		model string
		bad   bool
	}{
		{"invalid enum", options.MiniMax{TextNormalizationMode: &invalid}, "speech-2.8-hd", false},
		{"legacy quality", options.MiniMax{TextNormalizationMode: &quality}, "speech-02-hd", false},
		{"unknown quality", options.MiniMax{TextNormalizationMode: &quality}, "speech-new-hd", false},
		{"wrong options", options.ElevenLabs{}, "speech-2.8-hd", false},
		{"pointer options", &options.MiniMax{}, "speech-2.8-hd", false},
		{"invalid last segment", options.MiniMax{TextNormalizationMode: &quality}, "speech-2.8-hd", true},
	} {
		for _, concurrency := range []int{1, 4} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, concurrency), func(t *testing.T) {
				var calls atomic.Int32
				p := provider{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return nil, errors.New("unexpected HTTP call")
				})}}
				out := preservedOutput(t)
				doc := synthesisDocument()
				if tc.bad {
					doc.Utterances[2].Nodes = append(doc.Utterances[2].Nodes, ir.Node{Kind: ir.State, Name: "unknown"}, ir.Node{Kind: ir.Text, Text: "bad"})
				}
				opt := providers.Options{Out: out, Voice: "voice", Model: tc.model, Endpoint: "https://example.invalid", Synthesis: tc.value, Concurrency: concurrency}
				_, err := p.Synthesize(context.Background(), doc, opt)
				if err == nil || calls.Load() != 0 {
					t.Fatalf("error = %v, calls = %d", err, calls.Load())
				}
				assertPreservedOutput(t, out)
				missingDir := filepath.Join(t.TempDir(), "not-created")
				opt.Out = filepath.Join(missingDir, "out.wav")
				if _, err := p.Synthesize(context.Background(), doc, opt); err == nil || calls.Load() != 0 {
					t.Fatalf("new output preflight: error = %v, calls = %d", err, calls.Load())
				}
				if _, err := os.Stat(missingDir); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("preflight touched output directory: %v", err)
				}
			})
		}
	}
}

func TestSynthesisSerialConcurrentSnapshots(t *testing.T) {
	var baseline []byte
	var baselineBodies []string
	for _, concurrency := range []int{1, 4} {
		mode := options.MiniMaxNormalizationQuality
		var once sync.Once
		var mu sync.Mutex
		var bodies []string
		thirdStarted := make(chan struct{})
		p := provider{client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost || req.URL.Path != "/v1/t2a_v2" || req.Header.Get("Authorization") != "Bearer FAKE" || req.Header.Get("Content-Type") != "application/json" {
				t.Errorf("request metadata = %s %s %v", req.Method, req.URL, req.Header)
			}
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			var r request
			if err := json.Unmarshal(body, &r); err != nil {
				return nil, err
			}
			once.Do(func() { mode = options.MiniMaxNormalizationBasic })
			if r.TextNormalizationMode == nil || *r.TextNormalizationMode != options.MiniMaxNormalizationQuality || r.Stream {
				t.Errorf("request was not snapshotted: %s", body)
			}
			mu.Lock()
			bodies = append(bodies, string(body))
			mu.Unlock()
			if concurrency > 1 {
				switch r.Text {
				case "one":
					select {
					case <-thirdStarted:
					case <-req.Context().Done():
						return nil, req.Context().Err()
					}
				case "three":
					close(thirdStarted)
				}
			}
			sample := map[string]byte{"one": 1, "two": 2, "three": 3}[r.Text]
			response := fmt.Sprintf(`{"base_resp":{"status_code":0},"data":{"status":2,"audio":"%02x00"}}`, sample)
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(response))}, nil
		})}}
		out := filepath.Join(t.TempDir(), "out.wav")
		_, err := p.Synthesize(context.Background(), synthesisDocument(), providers.Options{Out: out, Voice: "voice", Model: "speech-2.8-hd", Endpoint: "https://example.invalid/v1/", APIKey: "FAKE", Synthesis: options.MiniMax{TextNormalizationMode: &mode}, Concurrency: concurrency})
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(out)
		if err != nil || len(bodies) != 3 {
			t.Fatalf("output read = %v, requests = %d", err, len(bodies))
		}
		if len(got) != 50 || !bytes.Equal(got[44:], []byte{1, 0, 2, 0, 3, 0}) {
			t.Fatalf("audio is not in script order: %x", got)
		}
		slices.Sort(bodies)
		if baseline == nil {
			baseline, baselineBodies = got, bodies
		} else if !bytes.Equal(got, baseline) || !slices.Equal(bodies, baselineBodies) {
			t.Fatalf("serial/concurrent output or requests differ")
		}
	}
}

func TestSynthesisCancellationPreservesOutput(t *testing.T) {
	for _, concurrency := range []int{1, 4} {
		for _, preCanceled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/pre=%t", concurrency, preCanceled), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var calls atomic.Int32
				p := provider{client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls.Add(1)
					cancel()
					<-req.Context().Done()
					return nil, req.Context().Err()
				})}}
				if preCanceled {
					cancel()
				}
				mode := options.MiniMaxNormalizationQuality
				out := preservedOutput(t)
				_, err := p.Synthesize(ctx, synthesisDocument(), providers.Options{Out: out, Voice: "voice", Model: "speech-2.8-hd", Endpoint: "https://example.invalid", Synthesis: options.MiniMax{TextNormalizationMode: &mode}, Concurrency: concurrency})
				if !errors.Is(err, context.Canceled) || preCanceled && calls.Load() != 0 || !preCanceled && calls.Load() == 0 {
					t.Fatalf("error = %v, calls = %d", err, calls.Load())
				}
				assertPreservedOutput(t, out)
			})
		}
	}
}

func requestJSON(t *testing.T, r request) string {
	t.Helper()
	body, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func synthesisDocument() *ir.Document {
	return &ir.Document{Utterances: []ir.Utterance{
		{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "one"}}},
		{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "two"}}},
		{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "three"}}},
	}}
}

func preservedOutput(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "out.wav")
	if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
		t.Fatal(err)
	}
	return out
}

func assertPreservedOutput(t *testing.T, out string) {
	t.Helper()
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "KEEP" {
		t.Fatalf("output = %q, error = %v", data, err)
	}
	files, err := os.ReadDir(filepath.Dir(out))
	if err != nil || len(files) != 1 || files[0].Name() != "out.wav" {
		t.Fatalf("unexpected output artifacts = %v, error = %v", files, err)
	}
}
