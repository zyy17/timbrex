package fish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

// Independently transcribed from TTSRequest, PronunciationDictionaryRef,
// PronunciationDictionaryInline and PronunciationItem, verified 2026-10-24:
// https://github.com/fishaudio/docs/blob/main/api-reference/openapi.json
// https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech#body-pronunciation-dictionary
func TestDictionaryCompleteWireContracts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		value  options.Fish
		fields string
	}{
		{"inline", dictionaryFixture(), `"features":["quality-guard"],"pronunciation_dictionary":[{"items":[{"key":"重庆","value":"tʂʰʊŋ tɕʰiŋ","case_sensitive":false},{"key":"word","value":"wɜːd"}]},{"items":[]}],`},
		{"references", options.Fish{PronunciationDictionary: &options.FishDictionaries{References: &[]options.FishDictionaryReference{{ID: "dictionary_1", Version: "v-1"}, {ID: "dictionary_2", Version: "v-2"}}}}, `"pronunciation_dictionary":[{"id":"dictionary_1","version":"v-1"},{"id":"dictionary_2","version":"v-2"}],`},
		{"case_true", options.Fish{PronunciationDictionary: &options.FishDictionaries{Inline: &[]options.FishDictionaryInline{{Items: []options.FishPronunciationItem{{Key: "Word", Value: "wɜːd", CaseSensitive: fishPointer(true)}}}}}}, `"pronunciation_dictionary":[{"items":[{"key":"Word","value":"wɜːd","case_sensitive":true}]}],`},
		{"empty", options.Fish{Features: &[]options.FishFeature{}, PronunciationDictionary: &options.FishDictionaries{References: &[]options.FishDictionaryReference{}}}, `"features":[],"pronunciation_dictionary":[],`},
		{"empty_inline", options.Fish{PronunciationDictionary: &options.FishDictionaries{Inline: &[]options.FishDictionaryInline{}}}, `"pronunciation_dictionary":[],`},
	} {
		for _, model := range []string{"s2-pro", "s2.1-pro", "s2.1-pro-free"} {
			t.Run(tc.name+"/"+model, func(t *testing.T) {
				var calls atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
					}
					want := `{` + tc.fields + `"text":"Hello","reference_id":"test-reference","format":"wav","sample_rate":24000,"normalize":false,"prosody":{"speed":1,"volume":0,"normalize_loudness":true}}`
					if string(body) != want {
						t.Errorf("body=%s; want %s", body, want)
					}
					if r.Method != http.MethodPost || r.URL.RequestURI() != "/v1/tts" || r.Header.Get("Authorization") != "Bearer FAKE-KEY" || r.Header.Get("Model") != model || r.Header.Get("Content-Type") != "application/json" {
						t.Errorf("wire route/header mismatch: %s %s", r.Method, r.URL.RequestURI())
					}
					_, _ = w.Write(fixtureWAV())
				}))
				defer srv.Close()
				doc := &ir.Document{Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{{Text: "Hello"}}}}}
				_, err := (provider{}).Synthesize(context.Background(), doc, providers.Options{Model: model, Voice: "test-reference", Endpoint: srv.URL + "/v1", APIKey: "FAKE-KEY", Out: filepath.Join(t.TempDir(), "out.wav"), Synthesis: tc.value})
				if err != nil || calls.Load() != 1 {
					t.Fatalf("calls=%d error=%v", calls.Load(), err)
				}
			})
		}
	}
}

func TestDictionarySnapshotIsolation(t *testing.T) {
	value := dictionaryFixture()
	seg := providers.Segment{Voice: "voice", Nodes: []ir.Node{{Text: "hello"}}}
	first, err := synthesisOptions(value)
	if err != nil {
		t.Fatal(err)
	}
	second, err := synthesisOptions(value)
	if err != nil {
		t.Fatal(err)
	}
	_, prepared, err := providers.Prepare(context.Background(), synthesisDocument(), "fish", "voice", func(seg providers.Segment) (request, error) { return first.build(seg, "s2-pro") })
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range prepared {
		if r.PronunciationDictionary != first.dictionary || r.Features != first.value.Features {
			t.Fatal("prepared requests do not share one private snapshot")
		}
	}
	independent, err := buildWithSynthesis(seg, "s2-pro", value)
	if err != nil {
		t.Fatal(err)
	}
	want := requestJSON(t, independent)
	(*value.PronunciationDictionary.Inline)[0].Items[0].Value = "caller mutation"
	*(*value.PronunciationDictionary.Inline)[0].Items[0].CaseSensitive = true
	(*value.Features)[0] = "caller mutation"
	first.dictionary.inline[0].Items[0].Value = "first snapshot mutation"
	*first.dictionary.inline[0].Items[0].CaseSensitive = true
	(*first.value.Features)[0] = "first snapshot mutation"
	secondRequest, err := second.build(seg, "s2-pro")
	if err != nil {
		t.Fatal(err)
	}
	assertRequestJSON(t, secondRequest, want)
	assertRequestJSON(t, independent, want)
	if (*value.PronunciationDictionary.Inline)[0].Items[0].Value != "caller mutation" {
		t.Fatal("snapshot changed caller storage")
	}

	refs := []options.FishDictionaryReference{{ID: "first", Version: "v1"}}
	refValue := options.Fish{PronunciationDictionary: &options.FishDictionaries{References: &refs}}
	refRequest, err := buildWithSynthesis(seg, "s2-pro", refValue)
	if err != nil {
		t.Fatal(err)
	}
	refs[0].ID = "mutated"
	if refRequest.PronunciationDictionary.references[0].ID != "first" {
		t.Fatal("reference list shares caller storage")
	}
}

func TestDictionarySnapshotsAcrossConcurrentInvocations(t *testing.T) {
	for _, concurrency := range []int{1, 8} {
		t.Run(fmt.Sprint(concurrency), func(t *testing.T) {
			value := dictionaryFixture()
			started, release := make(chan struct{}), make(chan struct{})
			var start sync.Once
			releaseFirst := sync.OnceFunc(func() { close(release) })
			defer releaseFirst()
			type invocationKey struct{}
			p := provider{client: &http.Client{Transport: dictionaryRoundTrip(func(req *http.Request) (*http.Response, error) {
				wantValue, wantFlag := "second", false
				if req.Context().Value(invocationKey{}) != nil {
					start.Do(func() { close(started) })
					select {
					case <-release:
					case <-req.Context().Done():
						return nil, req.Context().Err()
					}
					wantValue, wantFlag = "tʂʰʊŋ tɕʰiŋ", true
				}
				var body struct {
					Features     []string                       `json:"features"`
					Dictionaries []options.FishDictionaryInline `json:"pronunciation_dictionary"`
				}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					return nil, err
				}
				if len(body.Dictionaries) != 2 || body.Dictionaries[0].Items[0].Value != wantValue || (len(body.Features) == 1) != wantFlag {
					t.Errorf("invocation data leaked: %+v", body)
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(fixtureWAV())))}, nil
			})}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			opt := providers.Options{Out: filepath.Join(t.TempDir(), "first.wav"), Voice: "voice", Model: "s2-pro", Endpoint: "https://example.invalid/v1", Synthesis: value, Concurrency: concurrency}
			done := make(chan error, 1)
			go func() {
				_, err := p.Synthesize(context.WithValue(ctx, invocationKey{}, true), synthesisDocument(), opt)
				done <- err
			}()
			select {
			case <-started:
			case err := <-done:
				t.Fatalf("first invocation stopped: %v", err)
			}
			(*value.PronunciationDictionary.Inline)[0].Items[0].Value = "second"
			*value.Features = []options.FishFeature{}
			opt.Out = filepath.Join(t.TempDir(), "second.wav")
			_, secondErr := p.Synthesize(ctx, synthesisDocument(), opt)
			releaseFirst()
			firstErr := <-done
			if firstErr != nil || secondErr != nil {
				t.Fatalf("first=%v second=%v", firstErr, secondErr)
			}
		})
	}
}

func TestDictionaryPreflightPreservesOutput(t *testing.T) {
	for _, concurrency := range []int{1, 8} {
		for _, badOptions := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%t", concurrency, badOptions), func(t *testing.T) {
				var calls atomic.Int32
				p := provider{client: &http.Client{Transport: dictionaryRoundTrip(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return nil, errors.New("unexpected request")
				})}}
				value, doc := dictionaryFixture(), synthesisDocument()
				if badOptions {
					(*value.PronunciationDictionary.Inline)[1].Items = []options.FishPronunciationItem{{Key: "late", Value: "<|phoneme_end|>"}}
				} else {
					doc.Utterances[2].Nodes = []ir.Node{{Kind: ir.State, Name: "pitch", Args: []ir.Arg{{Kind: "semitone", Raw: "2st"}}}, {Text: "late"}}
				}
				out := preservedSynthesisOutput(t)
				_, err := p.Synthesize(context.Background(), doc, providers.Options{Out: out, Voice: "voice", Model: "s2-pro", Endpoint: "https://example.invalid/v1", Synthesis: value, Concurrency: concurrency})
				if err == nil || calls.Load() != 0 {
					t.Fatalf("calls=%d error=%v", calls.Load(), err)
				}
				assertSynthesisOutputPreserved(t, out)
			})
		}
	}
}

func TestDictionaryPreparationBoundedAllocation(t *testing.T) {
	value := largeDictionary(5000)
	stop := errors.New("stop after preparation")
	p := provider{client: &http.Client{Transport: dictionaryRoundTrip(func(*http.Request) (*http.Response, error) { return nil, stop })}}
	opt := providers.Options{Out: preservedSynthesisOutput(t), Voice: "voice", Model: "s2-pro", Endpoint: "https://example.invalid/v1", Synthesis: value}
	measure := func(segments int) uint64 {
		doc := dictionaryDocument(segments)
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		_, err := p.Synthesize(context.Background(), doc, opt)
		runtime.ReadMemStats(&after)
		if !errors.Is(err, stop) {
			t.Fatalf("expected first HTTP after preflight: %v", err)
		}
		return after.TotalAlloc - before.TotalAlloc
	}
	one, hundred := measure(1), measure(100)
	// A per-segment copy retains >24 MB for 100 × 5,000 items. This budget
	// permits planning/encoding overhead, but no dictionary-sized segment copies.
	if hundred > one+2<<20 {
		t.Fatalf("dictionary allocation scales with segments: one=%d hundred=%d", one, hundred)
	}
	t.Logf("allocated bytes: one=%d hundred=%d", one, hundred)
	assertSynthesisOutputPreserved(t, opt.Out)
}

func BenchmarkDictionaryPreparation(b *testing.B) {
	for _, segments := range []int{1, 100} {
		b.Run(fmt.Sprint(segments), func(b *testing.B) {
			value, doc := largeDictionary(5000), dictionaryDocument(segments)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				snapshot, err := synthesisOptions(value)
				if err != nil {
					b.Fatal(err)
				}
				_, _, err = providers.Prepare(context.Background(), doc, "fish", "voice", func(seg providers.Segment) (request, error) { return snapshot.build(seg, "s2-pro") })
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func dictionaryFixture() options.Fish {
	return options.Fish{Features: &[]options.FishFeature{options.FishQualityGuard}, PronunciationDictionary: &options.FishDictionaries{Inline: &[]options.FishDictionaryInline{
		{Items: []options.FishPronunciationItem{{Key: "重庆", Value: "tʂʰʊŋ tɕʰiŋ", CaseSensitive: fishPointer(false)}, {Key: "word", Value: "wɜːd"}}},
		{Items: []options.FishPronunciationItem{}},
	}}}
}

func largeDictionary(count int) options.Fish {
	items := make([]options.FishPronunciationItem, count)
	for i := range items {
		items[i] = options.FishPronunciationItem{Key: fmt.Sprintf("word%d", i), Value: "phonetic"}
	}
	return options.Fish{PronunciationDictionary: &options.FishDictionaries{Inline: &[]options.FishDictionaryInline{{Items: items}}}}
}

func dictionaryDocument(segments int) *ir.Document {
	doc := &ir.Document{Utterances: make([]ir.Utterance, segments)}
	for i := range doc.Utterances {
		doc.Utterances[i] = ir.Utterance{Speaker: "narrator", Nodes: []ir.Node{{Text: "hello"}}}
	}
	return doc
}

type dictionaryRoundTrip func(*http.Request) (*http.Response, error)

func (f dictionaryRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
