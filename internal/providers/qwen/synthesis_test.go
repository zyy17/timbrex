package qwen

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
	seed := int64(0)
	pronunciation := []options.QwenTextPair{{Text: "天气", Replacement: "tian1 qi4"}, {Text: "天气", Replacement: "tian1 qi4"}}
	replace := []options.QwenTextPair{{Text: "今天", Replacement: "金天"}, {Text: "今天", Replacement: "今\"天\n"}}
	value := options.Qwen{Seed: &seed, HotFix: &options.QwenHotFix{Pronunciation: &pronunciation, Replace: &replace}}
	seg := providers.Segment{Voice: "longanhuan_v3.1", Lang: "zh-CN", Nodes: []ir.Node{{Kind: ir.Text, Text: "天气，今天。"}}, Controls: []ir.Node{
		{Name: "speed", Args: []ir.Arg{{Kind: "number", Raw: "1.1"}}},
		{Name: "pitch", Args: []ir.Arg{{Kind: "semitone", Raw: "+12st"}}},
		{Name: "volume", Args: []ir.Arg{{Kind: "decibel", Raw: "0dB"}}},
		{Name: "act", Args: []ir.Arg{{Raw: "warmly"}}},
	}}
	r, err := buildWithSynthesis(seg, "qwen-audio-3.1-tts-flash", value)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"model":"qwen-audio-3.1-tts-flash","input":{"text":"天气，今天。","voice":"longanhuan_v3.1","format":"pcm","sample_rate":24000,"instruction":"warmly","rate":1.1,"pitch":2,"volume":50,"language_hints":["zh"],"seed":0,"hot_fix":{"pronunciation":[{"天气":"tian1 qi4"},{"天气":"tian1 qi4"}],"replace":[{"今天":"金天"},{"今天":"今\"天\n"}]}}}`
	if got := requestJSON(t, r); got != want {
		t.Fatalf("body = %s; want %s", got, want)
	}
}

func TestSynthesisOmissionAndExplicitEmptyArrays(t *testing.T) {
	seg := providers.Segment{Voice: "voice", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}
	var nilPairs []options.QwenTextPair
	emptyPairs := []options.QwenTextPair{}
	for _, model := range []string{"qwen-audio-3.1-tts-flash", "qwen-audio-3.1-tts-plus", "qwen-audio-3.0-tts-flash", "qwen-audio-3.0-tts-plus"} {
		base, err := build(seg, model)
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range []options.Qwen{{}, {HotFix: &options.QwenHotFix{}}} {
			r, err := buildWithSynthesis(seg, model, value)
			if err != nil || requestJSON(t, r) != requestJSON(t, base) {
				t.Fatalf("omission changed %s: %+v, %v", model, r, err)
			}
		}
		for _, tc := range []struct {
			fix  options.QwenHotFix
			want string
		}{
			{options.QwenHotFix{Pronunciation: &emptyPairs}, `"hot_fix":{"pronunciation":[]}`},
			{options.QwenHotFix{Replace: &emptyPairs}, `"hot_fix":{"replace":[]}`},
			{options.QwenHotFix{Pronunciation: &nilPairs, Replace: &emptyPairs}, `"hot_fix":{"pronunciation":[],"replace":[]}`},
		} {
			r, err := buildWithSynthesis(seg, model, options.Qwen{HotFix: &tc.fix})
			if err != nil || !strings.Contains(requestJSON(t, r), tc.want) {
				t.Fatalf("explicit empty array: %+v, %v; want %s", r, err, tc.want)
			}
		}
	}
}

func TestSynthesisSnapshotsNestedValues(t *testing.T) {
	seed := int64(65535)
	pairs := []options.QwenTextPair{{Text: " 天气\"\n", Replacement: ""}, {Text: "天气", Replacement: "tian1 qi4"}}
	value := options.Qwen{Seed: &seed, HotFix: &options.QwenHotFix{Pronunciation: &pairs, Replace: &pairs}}
	seg := providers.Segment{Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}
	first, err := buildWithSynthesis(seg, "qwen-audio-3.1-tts-flash", value)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildWithSynthesis(seg, "qwen-audio-3.1-tts-flash", value)
	if err != nil {
		t.Fatal(err)
	}
	want := requestJSON(t, first)
	seed = 0
	pairs[0].Text, pairs[0].Replacement = "mutated", "caller"
	pairs = append(pairs, options.QwenTextPair{Text: "extra", Replacement: "new"})
	value.HotFix.Pronunciation = nil
	if requestJSON(t, first) != want || requestJSON(t, second) != want {
		t.Fatal("request shares nested caller options")
	}
	(*first.Input.HotFix.Pronunciation)[0]["new"] = "changed"
	(*first.Input.HotFix.Replace)[1]["天气"] = "changed"
	*first.Input.Seed = 123
	if requestJSON(t, second) != want || seed != 0 || pairs[1].Replacement != "tian1 qi4" {
		t.Fatal("requests or caller share mutable storage")
	}
}

func TestSynthesisInvalidOptionsAndLastSegmentPreflight(t *testing.T) {
	badSeed := int64(65536)
	negativeSeed := int64(-1)
	pairs := []options.QwenTextPair{{Text: "", Replacement: "bad"}}
	invalidText := []options.QwenTextPair{{Text: "PRIVATE_SOURCE_\xff", Replacement: "value"}}
	invalidReplacement := []options.QwenTextPair{{Text: "source", Replacement: "PRIVATE_VALUE_\xff"}}
	goodPairs := []options.QwenTextPair{{Text: "天气", Replacement: "tian1 qi4"}}
	valid := options.Qwen{HotFix: &options.QwenHotFix{Pronunciation: &goodPairs}}
	for _, tc := range []struct {
		name  string
		value options.Value
		model string
		bad   bool
	}{
		{"seed overflow", options.Qwen{Seed: &badSeed}, "qwen-audio-3.1-tts-flash", false},
		{"negative seed", options.Qwen{Seed: &negativeSeed}, "qwen-audio-3.1-tts-flash", false},
		{"empty source", options.Qwen{HotFix: &options.QwenHotFix{Replace: &pairs}}, "qwen-audio-3.1-tts-flash", false},
		{"invalid UTF-8 source", options.Qwen{HotFix: &options.QwenHotFix{Pronunciation: &invalidText}}, "qwen-audio-3.1-tts-flash", false},
		{"invalid UTF-8 replacement", options.Qwen{HotFix: &options.QwenHotFix{Replace: &invalidReplacement}}, "qwen-audio-3.1-tts-flash", false},
		{"wrong options", options.ElevenLabs{}, "qwen-audio-3.1-tts-flash", false},
		{"pointer options", &options.Qwen{}, "qwen-audio-3.1-tts-flash", false},
		{"unknown model", valid, "qwen3-tts-flash", false},
		{"invalid last segment", valid, "qwen-audio-3.1-tts-flash", true},
	} {
		for _, concurrency := range []int{1, 4} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, concurrency), func(t *testing.T) {
				var calls atomic.Int32
				p := provider{client: &http.Client{Transport: synthesisRoundTrip(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return nil, errors.New("unexpected HTTP call")
				})}}
				out := preservedOutput(t)
				doc := synthesisDocument()
				if tc.bad {
					doc.Utterances[2].Nodes = append(doc.Utterances[2].Nodes, ir.Node{Kind: ir.State, Name: "unknown"}, ir.Node{Kind: ir.Text, Text: "bad"})
				}
				opt := providers.Options{Out: out, Voice: "voice", Model: tc.model, Endpoint: "https://example.invalid/speech", Synthesis: tc.value, Concurrency: concurrency}
				_, err := p.Synthesize(context.Background(), doc, opt)
				if err == nil || strings.Contains(err.Error(), "PRIVATE_") || calls.Load() != 0 {
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
		seed := int64(65535)
		pairs := []options.QwenTextPair{{Text: "天气", Replacement: "tian1 qi4"}, {Text: "天气", Replacement: "new"}}
		value := options.Qwen{Seed: &seed, HotFix: &options.QwenHotFix{Pronunciation: &pairs, Replace: &pairs}}
		var once sync.Once
		var mu sync.Mutex
		var bodies []string
		thirdStarted := make(chan struct{})
		p := provider{client: &http.Client{Transport: synthesisRoundTrip(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodGet {
				if req.Header.Get("Authorization") != "" {
					t.Error("synthesis credential leaked to audio download")
				}
				sample := map[string]byte{"/one": 1, "/two": 2, "/three": 3}[req.URL.Path]
				return testResponse("audio/pcm", []byte{sample, 0}), nil
			}
			if req.Method != http.MethodPost || req.URL.Path != "/speech" || req.Header.Get("Authorization") != "Bearer FAKE" || req.Header.Get("Content-Type") != "application/json" {
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
			// All requests must be detached before the first network call.
			once.Do(func() { seed = 1; pairs[0].Replacement = "mutated"; value.HotFix.Replace = nil })
			if r.Input.Seed == nil || *r.Input.Seed != 65535 || r.Input.HotFix == nil || (*r.Input.HotFix.Pronunciation)[0]["天气"] != "tian1 qi4" || r.Input.HotFix.Replace == nil {
				t.Errorf("request was not snapshotted: %s", body)
			}
			mu.Lock()
			bodies = append(bodies, string(body))
			mu.Unlock()
			if concurrency > 1 {
				switch r.Input.Text {
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
			return testResponse("application/json", []byte(fmt.Sprintf(`{"output":{"finish_reason":"stop","audio":{"url":"https://audio.example.invalid/%s"}}}`, r.Input.Text))), nil
		})}}
		out := filepath.Join(t.TempDir(), "out.wav")
		_, err := p.Synthesize(context.Background(), synthesisDocument(), providers.Options{Out: out, Voice: "voice", Model: "qwen-audio-3.1-tts-flash", Endpoint: "https://example.invalid/speech", APIKey: "FAKE", Synthesis: value, Concurrency: concurrency})
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
				p := provider{client: &http.Client{Transport: synthesisRoundTrip(func(req *http.Request) (*http.Response, error) {
					calls.Add(1)
					cancel()
					<-req.Context().Done()
					return nil, req.Context().Err()
				})}}
				if preCanceled {
					cancel()
				}
				seed := int64(0)
				out := preservedOutput(t)
				_, err := p.Synthesize(ctx, synthesisDocument(), providers.Options{Out: out, Voice: "voice", Model: "qwen-audio-3.1-tts-flash", Endpoint: "https://example.invalid/speech", Synthesis: options.Qwen{Seed: &seed}, Concurrency: concurrency})
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

type synthesisRoundTrip func(*http.Request) (*http.Response, error)

func (f synthesisRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func testResponse(contentType string, body []byte) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(bytes.NewReader(body))}
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
