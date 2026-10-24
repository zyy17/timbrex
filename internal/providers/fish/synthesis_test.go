package fish

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

func fishPointer[T any](v T) *T { return &v }

func synthesisFixture() options.Fish {
	return options.Fish{
		Normalize:                 fishPointer(true),
		NormalizationLanguage:     fishPointer("es-MX"),
		Prosody:                   &options.FishProsody{NormalizeLoudness: fishPointer(false)},
		Temperature:               fishPointer(0.0),
		TopP:                      fishPointer(0.0),
		ChunkLength:               fishPointer(int64(100)),
		MinChunkLength:            fishPointer(int64(0)),
		Latency:                   fishPointer(options.FishLatencyLow),
		ConditionOnPreviousChunks: fishPointer(false),
		EarlyStopThreshold:        fishPointer(0.0),
	}
}

// Exact full-body fixtures follow the official TTSRequest and ProsodyControl
// schemas, retaining this adapter's WAV/24000 output and DSL speed/volume.
// https://github.com/fishaudio/docs/blob/main/api-reference/openapi.json
func TestSynthesisRequestFields(t *testing.T) {
	for _, model := range []string{"s2-pro", "s2.1-pro", "s2.1-pro-free"} {
		t.Run(model, func(t *testing.T) {
			seg := providers.Segment{
				Voice: "test-reference", Nodes: []ir.Node{{Text: "Hello 123"}},
				Controls: []ir.Node{
					{Name: "speed", Args: []ir.Arg{{Kind: "number", Raw: "1.25"}}},
					{Name: "volume", Args: []ir.Arg{{Kind: "decibel", Raw: "-6dB"}}},
				},
			}
			r, err := buildWithSynthesis(seg, model, synthesisFixture())
			if err != nil {
				t.Fatal(err)
			}
			assertRequestJSON(t, r, `{"text":"Hello 123","reference_id":"test-reference","format":"wav","sample_rate":24000,"normalize":true,"prosody":{"speed":1.25,"volume":-6,"normalize_loudness":false},"language":"es-MX","temperature":0,"top_p":0,"chunk_length":100,"min_chunk_length":0,"latency":"low","condition_on_previous_chunks":false,"early_stop_threshold":0}`)
		})
	}
}

func TestSynthesisUpperBoundsAndExplicitEmptyLanguage(t *testing.T) {
	value := options.Fish{
		Normalize: fishPointer(false), NormalizationLanguage: fishPointer(""),
		Prosody:     &options.FishProsody{NormalizeLoudness: fishPointer(true)},
		Temperature: fishPointer(1.0), TopP: fishPointer(1.0), EarlyStopThreshold: fishPointer(1.0),
		ChunkLength: fishPointer(int64(300)), MinChunkLength: fishPointer(int64(100)),
		ConditionOnPreviousChunks: fishPointer(true), Latency: fishPointer(options.FishLatencyBalanced),
	}
	seg := providers.Segment{Voice: "test-reference", Nodes: []ir.Node{{Text: "Hello"}}}
	r, err := buildWithSynthesis(seg, "s2.1-pro", value)
	if err != nil {
		t.Fatal(err)
	}
	assertRequestJSON(t, r, `{"text":"Hello","reference_id":"test-reference","format":"wav","sample_rate":24000,"normalize":false,"prosody":{"speed":1,"volume":0,"normalize_loudness":true},"language":"","temperature":1,"top_p":1,"chunk_length":300,"min_chunk_length":100,"latency":"balanced","condition_on_previous_chunks":true,"early_stop_threshold":1}`)
	value.Latency = fishPointer(options.FishLatencyNormal)
	value.NormalizationLanguage = fishPointer("auto")
	r, err = buildWithSynthesis(seg, "s2.1-pro", value)
	if err != nil || *r.Language != "auto" || *r.Latency != "normal" {
		t.Fatalf("request=%+v error=%v", r, err)
	}
}

func TestSynthesisOmissionPreservesCompleteRequest(t *testing.T) {
	const want = `{"text":"Hello","reference_id":"test-reference","format":"wav","sample_rate":24000,"normalize":false,"prosody":{"speed":1,"volume":0,"normalize_loudness":true}}`
	for _, model := range []string{"s2-pro", "s2.1-pro", "s2.1-pro-free"} {
		for _, input := range []options.Value{nil, options.Fish{}, options.Fish{Prosody: &options.FishProsody{}}} {
			value, err := synthesisOptions(input)
			if err != nil {
				t.Fatal(err)
			}
			r, err := value.build(providers.Segment{Voice: "test-reference", Nodes: []ir.Node{{Text: "Hello"}}}, model)
			if err != nil {
				t.Fatal(err)
			}
			assertRequestJSON(t, r, want)
		}
	}
}

func TestSynthesisNormalizationDoesNotEnableSpokenLanguage(t *testing.T) {
	value := synthesisFixture()
	for _, normalize := range []bool{true, false} {
		value.Normalize = &normalize
		seg := providers.Segment{Voice: "test-reference", Lang: "es-MX", Nodes: []ir.Node{{Text: "Hello"}}}
		if _, err := buildWithSynthesis(seg, "s2.1-pro", value); err == nil || !strings.Contains(err.Error(), "spoken-language") {
			t.Fatalf("accepted DSL language with normalization language: %v", err)
		}
	}
}

func TestSynthesisEntryRejectsInvalidOptions(t *testing.T) {
	for _, value := range []options.Value{options.ElevenLabs{}, &options.Fish{}, (*options.Fish)(nil)} {
		if _, err := synthesisOptions(value); err == nil {
			t.Fatal("accepted unsupported synthesis variant")
		}
	}
	invalid := options.Fish{Temperature: fishPointer(math.NaN())}
	if _, err := synthesisOptions(invalid); err == nil {
		t.Fatal("accepted invalid synthesis options")
	}
}

func TestSynthesisPreparedRequestsOwnAllValues(t *testing.T) {
	value := synthesisFixture()
	seg := providers.Segment{Voice: "test-reference", Nodes: []ir.Node{{Text: "Hello"}}}
	first, err := buildWithSynthesis(seg, "s2.1-pro", value)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildWithSynthesis(seg, "s2.1-pro", value)
	if err != nil {
		t.Fatal(err)
	}
	want := requestJSON(t, first)
	mutateSynthesisFixture(value)
	assertRequestJSON(t, first, want)
	assertRequestJSON(t, second, want)
	*first.Language, *first.Temperature, *first.TopP = "auto", 0.5, 0.5
	*first.ChunkLength, *first.MinChunkLength, *first.Latency = 200, 50, options.FishLatencyNormal
	*first.ConditionOnPreviousChunks, *first.EarlyStopThreshold = true, 0.5
	first.Normalize, first.Prosody.NormalizeLoudness = false, true
	assertRequestJSON(t, second, want)
	if *value.NormalizationLanguage != "zh-CN" || *value.Temperature != 1 || *value.TopP != 1 || *value.ChunkLength != 300 || *value.MinChunkLength != 100 || *value.Latency != options.FishLatencyBalanced || !*value.ConditionOnPreviousChunks || *value.EarlyStopThreshold != 1 || *value.Normalize || !*value.Prosody.NormalizeLoudness {
		t.Fatal("request mutation changed caller options")
	}
}

func mutateSynthesisFixture(value options.Fish) {
	*value.Normalize, *value.Prosody.NormalizeLoudness = false, true
	*value.NormalizationLanguage = "zh-CN"
	*value.Temperature, *value.TopP, *value.EarlyStopThreshold = 1, 1, 1
	*value.ChunkLength, *value.MinChunkLength = 300, 100
	*value.Latency = options.FishLatencyBalanced
	*value.ConditionOnPreviousChunks = true
}

func TestSynthesisOptionsSerialAndConcurrent(t *testing.T) {
	for _, concurrency := range []int{0, 1, 3} {
		t.Run(fmt.Sprint(concurrency), func(t *testing.T) {
			value := synthesisFixture()
			var mutate sync.Once
			var calls atomic.Int32
			later := make(chan struct{}, 2)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					return
				}
				if r.URL.Path != "/tts" || r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer FAKE-KEY" || r.Header.Get("Model") != "s2.1-pro" || r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("unexpected request: %s %s %v", r.Method, r.URL, r.Header)
				}
				// The first POST can only happen after whole-script preparation.
				// Mutating caller pointers here must not race with any request.
				mutate.Do(func() { mutateSynthesisFixture(value) })
				var got request
				if err := json.Unmarshal(body, &got); err != nil {
					t.Error(err)
					return
				}
				want := `{"text":"TEXT","reference_id":"test-reference","format":"wav","sample_rate":24000,"normalize":true,"prosody":{"speed":1,"volume":0,"normalize_loudness":false},"language":"es-MX","temperature":0,"top_p":0,"chunk_length":100,"min_chunk_length":0,"latency":"low","condition_on_previous_chunks":false,"early_stop_threshold":0}`
				if string(body) != strings.Replace(want, "TEXT", got.Text, 1) {
					t.Errorf("body=%s", body)
				}
				var sample byte
				switch got.Text {
				case "first":
					sample = 1
					if concurrency > 1 {
						for range 2 {
							select {
							case <-later:
							case <-r.Context().Done():
								return
							}
						}
					}
				case "second":
					sample = 2
				case "third":
					sample = 3
				default:
					t.Errorf("unexpected text %q", got.Text)
				}
				wav := fixtureWAV()
				wav[44], wav[46] = sample, sample
				_, _ = w.Write(wav)
				if sample != 1 {
					later <- struct{}{}
				}
			}))
			defer srv.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			out := filepath.Join(t.TempDir(), "out.wav")
			result, err := (provider{}).Synthesize(ctx, synthesisDocument(), providers.Options{Model: "s2.1-pro", Voice: "test-reference", APIKey: "FAKE-KEY", Endpoint: srv.URL, Out: out, Concurrency: concurrency, Synthesis: value})
			if err != nil || calls.Load() != 3 || result.Path != out || result.Provider != "fish" {
				t.Fatalf("result=%+v calls=%d error=%v", result, calls.Load(), err)
			}
			wav, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			pcm, err := providers.DecodeWAV(wav)
			if err != nil || !bytes.Equal(pcm, []byte{1, 0, 1, 0, 2, 0, 2, 0, 3, 0, 3, 0}) {
				t.Fatalf("PCM=%v error=%v", pcm, err)
			}
		})
	}
}

func TestSynthesisPreflightPreservesOutput(t *testing.T) {
	for _, concurrency := range []int{0, 3} {
		for _, tc := range []struct {
			name   string
			change func(*ir.Document, *providers.Options)
		}{
			{"temperature", func(_ *ir.Document, opt *providers.Options) {
				opt.Synthesis = options.Fish{Temperature: fishPointer(math.NaN())}
			}},
			{"top_p", func(_ *ir.Document, opt *providers.Options) {
				opt.Synthesis = options.Fish{TopP: fishPointer(math.Inf(1))}
			}},
			{"early_stop", func(_ *ir.Document, opt *providers.Options) {
				opt.Synthesis = options.Fish{EarlyStopThreshold: fishPointer(-0.1)}
			}},
			{"chunk", func(_ *ir.Document, opt *providers.Options) {
				opt.Synthesis = options.Fish{ChunkLength: fishPointer(int64(99))}
			}},
			{"min_chunk", func(_ *ir.Document, opt *providers.Options) {
				opt.Synthesis = options.Fish{MinChunkLength: fishPointer(int64(101))}
			}},
			{"latency", func(_ *ir.Document, opt *providers.Options) {
				opt.Synthesis = options.Fish{Latency: fishPointer(options.FishLatency(""))}
			}},
			{"normalization_language", func(_ *ir.Document, opt *providers.Options) {
				opt.Synthesis = options.Fish{NormalizationLanguage: fishPointer("en.US")}
			}},
			{"wrong_vendor", func(_ *ir.Document, opt *providers.Options) { opt.Synthesis = options.ElevenLabs{} }},
			{"pointer_variant", func(_ *ir.Document, opt *providers.Options) { opt.Synthesis = &options.Fish{} }},
			{"nil_pointer", func(_ *ir.Document, opt *providers.Options) { opt.Synthesis = (*options.Fish)(nil) }},
			{"model", func(_ *ir.Document, opt *providers.Options) { opt.Model = "s1" }},
			{"later_tag", func(doc *ir.Document, _ *providers.Options) {
				doc.Utterances[2].Nodes = []ir.Node{{Kind: ir.State, Name: "pitch", Args: []ir.Arg{{Kind: "semitone", Raw: "2st"}}}, {Text: "third"}}
			}},
			{"later_lang", func(doc *ir.Document, _ *providers.Options) {
				doc.Utterances[2].Nodes = []ir.Node{{Kind: ir.State, Name: "lang", Args: []ir.Arg{{Kind: "word", Raw: "en"}}}, {Text: "third"}}
			}},
		} {
			t.Run(fmt.Sprintf("%d/%s", concurrency, tc.name), func(t *testing.T) {
				var calls atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); _, _ = w.Write(fixtureWAV()) }))
				defer srv.Close()
				out := preservedSynthesisOutput(t)
				doc := synthesisDocument()
				opt := providers.Options{Model: "s2.1-pro", Voice: "test-reference", Endpoint: srv.URL, Out: out, Concurrency: concurrency, Synthesis: synthesisFixture()}
				tc.change(doc, &opt)
				if _, err := (provider{}).Synthesize(context.Background(), doc, opt); err == nil || calls.Load() != 0 {
					t.Fatalf("error=%v calls=%d", err, calls.Load())
				}
				assertSynthesisOutputPreserved(t, out)
			})
		}
	}
}

func TestSynthesisCancellationWithOptions(t *testing.T) {
	for _, precanceled := range []bool{true, false} {
		t.Run(fmt.Sprint(precanceled), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				if _, err := io.Copy(io.Discard, r.Body); err != nil {
					t.Error(err)
				}
				if calls.Add(1) == 2 {
					cancel()
				}
				select {
				case <-r.Context().Done():
				case <-time.After(2 * time.Second):
					t.Error("request did not observe cancellation")
				}
			}))
			defer srv.Close()
			if precanceled {
				cancel()
			}
			out := preservedSynthesisOutput(t)
			_, err := (provider{}).Synthesize(ctx, synthesisDocument(), providers.Options{Model: "s2.1-pro", Voice: "test-reference", Endpoint: srv.URL, Out: out, Concurrency: 2, Synthesis: synthesisFixture()})
			wantCalls := int32(2)
			if precanceled {
				wantCalls = 0
			}
			if !errors.Is(err, context.Canceled) || calls.Load() != wantCalls {
				t.Fatalf("error=%v calls=%d", err, calls.Load())
			}
			assertSynthesisOutputPreserved(t, out)
		})
	}
}

func synthesisDocument() *ir.Document {
	return &ir.Document{Utterances: []ir.Utterance{
		{Speaker: "narrator", Nodes: []ir.Node{{Text: "first"}}},
		{Speaker: "narrator", Nodes: []ir.Node{{Text: "second"}}},
		{Speaker: "narrator", Nodes: []ir.Node{{Text: "third"}}},
	}}
}

func requestJSON(t *testing.T, r request) string {
	t.Helper()
	body, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func assertRequestJSON(t *testing.T, r request, want string) {
	t.Helper()
	if got := requestJSON(t, r); got != want {
		t.Fatalf("body=%s; want %s", got, want)
	}
}

func preservedSynthesisOutput(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "out.wav")
	if err := os.WriteFile(out, []byte("KEEP"), 0600); err != nil {
		t.Fatal(err)
	}
	return out
}

func assertSynthesisOutputPreserved(t *testing.T, out string) {
	t.Helper()
	got, err := os.ReadFile(out)
	if err != nil || string(got) != "KEEP" {
		t.Fatalf("output=%q error=%v", got, err)
	}
	entries, err := os.ReadDir(filepath.Dir(out))
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary output leaked: %v, %v", entries, err)
	}
}
