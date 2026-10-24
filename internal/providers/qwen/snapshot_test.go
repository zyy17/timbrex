package qwen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

func TestSynthesisSnapshotSharedOnlyWithinPreparation(t *testing.T) {
	seed := int64(65535)
	pairs := []options.QwenTextPair{{Text: "天气", Replacement: "tian1 qi4"}}
	value := options.Qwen{Seed: &seed, HotFix: &options.QwenHotFix{Pronunciation: &pairs, Replace: &pairs}}
	first, err := synthesisOptions(value)
	if err != nil {
		t.Fatal(err)
	}
	second, err := synthesisOptions(value)
	if err != nil {
		t.Fatal(err)
	}
	_, requests, err := providers.Prepare(context.Background(), synthesisDocument(), "qwen", "voice", func(seg providers.Segment) (request, error) {
		return first.build(seg, "qwen-audio-3.1-tts-flash")
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range requests {
		if r.Input.Seed != first.seed || r.Input.HotFix != first.hotFix {
			t.Fatal("prepared requests do not share their invocation's immutable snapshot")
		}
	}
	seed = 0
	pairs[0].Replacement = "caller mutation"
	value.HotFix.Replace = nil
	*first.seed = 1
	(*first.hotFix.Pronunciation)[0]["天气"] = "first snapshot mutation"
	(*first.hotFix.Replace)[0]["天气"] = "first snapshot mutation"
	if *second.seed != 65535 || (*second.hotFix.Pronunciation)[0]["天气"] != "tian1 qi4" || (*second.hotFix.Replace)[0]["天气"] != "tian1 qi4" {
		t.Fatal("separate snapshots share mutable storage")
	}
	if seed != 0 || pairs[0].Replacement != "caller mutation" {
		t.Fatal("snapshot shares mutable caller storage")
	}
}

func TestSynthesisSeparateInvocationsSnapshotCallerValues(t *testing.T) {
	for _, concurrency := range []int{1, 4} {
		t.Run(fmt.Sprint(concurrency), func(t *testing.T) {
			seed := int64(1)
			pairs := []options.QwenTextPair{{Text: "word", Replacement: "first"}}
			value := options.Qwen{Seed: &seed, HotFix: &options.QwenHotFix{Pronunciation: &pairs, Replace: &pairs}}
			started, release := make(chan struct{}), make(chan struct{})
			var start sync.Once
			releaseFirst := sync.OnceFunc(func() { close(release) })
			defer releaseFirst()
			type invocationKey struct{}
			p := provider{client: &http.Client{Transport: synthesisRoundTrip(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					return testResponse("audio/pcm", []byte{1, 0}), nil
				}
				wantSeed, wantText := uint16(2), "second"
				if req.Context().Value(invocationKey{}) != nil {
					start.Do(func() { close(started) })
					select {
					case <-release:
					case <-req.Context().Done():
						return nil, req.Context().Err()
					}
					wantSeed, wantText = 1, "first"
				}
				var r request
				if err := json.NewDecoder(req.Body).Decode(&r); err != nil {
					return nil, err
				}
				if r.Input.Seed == nil || *r.Input.Seed != wantSeed || r.Input.HotFix == nil || r.Input.HotFix.Pronunciation == nil || r.Input.HotFix.Replace == nil || (*r.Input.HotFix.Pronunciation)[0]["word"] != wantText || (*r.Input.HotFix.Replace)[0]["word"] != wantText {
					t.Errorf("invocation snapshot = %+v; want seed %d and replacement %q", r.Input, wantSeed, wantText)
				}
				return testResponse("application/json", []byte(`{"output":{"finish_reason":"stop","audio":{"url":"https://audio.example.invalid/audio"}}}`)), nil
			})}}
			opt := providers.Options{Out: filepath.Join(t.TempDir(), "first.wav"), Voice: "voice", Model: "qwen-audio-3.1-tts-flash", Endpoint: "https://example.invalid/speech", Synthesis: value, Concurrency: concurrency}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := p.Synthesize(context.WithValue(ctx, invocationKey{}, true), synthesisDocument(), opt)
				done <- err
			}()
			select {
			case <-started:
			case err := <-done:
				t.Fatalf("first invocation stopped before HTTP: %v", err)
			}
			// The first invocation is fully prepared but still rendering. A new
			// invocation must snapshot the new values without changing the first.
			seed = 2
			pairs[0].Replacement = "second"
			secondOpt := opt
			secondOpt.Out = filepath.Join(t.TempDir(), "second.wav")
			_, secondErr := p.Synthesize(ctx, synthesisDocument(), secondOpt)
			releaseFirst()
			firstErr := <-done
			if firstErr != nil || secondErr != nil {
				t.Fatalf("first error = %v; second error = %v", firstErr, secondErr)
			}
		})
	}
}

func TestSynthesisHotFixAllocationsDoNotScaleWithSegments(t *testing.T) {
	value := benchmarkHotFix(1000)
	stop := errors.New("stop after preparation")
	p := provider{client: &http.Client{Transport: synthesisRoundTrip(func(*http.Request) (*http.Response, error) {
		return nil, stop
	})}}
	opt := providers.Options{Out: preservedOutput(t), Voice: "voice", Model: "qwen-audio-3.1-tts-flash", Endpoint: "https://example.invalid/speech", Synthesis: value}
	allocations := func(segments int) float64 {
		doc := benchmarkDocument(segments)
		return testing.AllocsPerRun(3, func() {
			if _, err := p.Synthesize(context.Background(), doc, opt); !errors.Is(err, stop) {
				t.Fatalf("expected first HTTP call after preparation, got %v", err)
			}
		})
	}
	one, hundred := allocations(1), allocations(100)
	// Allow generous per-segment planning overhead, but not the ~400,000
	// additional allocations from copying both 1,000-entry arrays 99 times.
	if hundred-one > 10000 {
		t.Fatalf("hot-fix allocations scale with segments: one=%g, hundred=%g", one, hundred)
	}
	t.Logf("allocations: one segment=%g, 100 segments=%g", one, hundred)
	assertPreservedOutput(t, opt.Out)
}

func BenchmarkSynthesisHotFixPreparation(b *testing.B) {
	for _, segments := range []int{1, 100} {
		b.Run(fmt.Sprintf("segments=%d/pairs=1000+1000", segments), func(b *testing.B) {
			doc, value := benchmarkDocument(segments), benchmarkHotFix(1000)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				snapshot, err := synthesisOptions(value)
				if err != nil {
					b.Fatal(err)
				}
				_, _, err = providers.Prepare(context.Background(), doc, "qwen", "voice", func(seg providers.Segment) (request, error) {
					return snapshot.build(seg, "qwen-audio-3.1-tts-flash")
				})
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func benchmarkHotFix(count int) options.Qwen {
	pairs := make([]options.QwenTextPair, count)
	for i := range pairs {
		pairs[i] = options.QwenTextPair{Text: fmt.Sprintf("word%d", i), Replacement: "replacement"}
	}
	return options.Qwen{HotFix: &options.QwenHotFix{Pronunciation: &pairs, Replace: &pairs}}
}

func benchmarkDocument(segments int) *ir.Document {
	doc := &ir.Document{Utterances: make([]ir.Utterance, segments)}
	for i := range doc.Utterances {
		doc.Utterances[i] = ir.Utterance{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "hello"}}}
	}
	return doc
}
