package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

func optionPointer[T any](value T) *T { return &value }

func nestedOptions() options.ElevenLabs {
	return options.ElevenLabs{
		Seed:                           optionPointer(int64(0)),
		ApplyTextNormalization:         optionPointer(options.ElevenNormalizationOff),
		ApplyLanguageTextNormalization: optionPointer(false),
		VoiceSettings: &options.ElevenVoiceSettings{
			Stability: optionPointer(0.0), SimilarityBoost: optionPointer(1.0),
			Style: optionPointer(0.0), UseSpeakerBoost: optionPointer(false),
		},
		PronunciationDictionaryLocators: &[]options.ElevenPronunciationDictionaryLocator{
			{PronunciationDictionaryID: "dict-b", VersionID: optionPointer("version-2")},
			{PronunciationDictionaryID: "dict-a"},
			{PronunciationDictionaryID: "dict-b", VersionID: optionPointer("version-2")},
		},
	}
}

func TestSynthesisNestedRequestWireFixture(t *testing.T) {
	seg := providers.Segment{
		Lang: "en-US", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello 123"}},
		Controls: []ir.Node{{Name: "speed", Args: []ir.Arg{{Kind: "number", Raw: "1.1"}}}},
	}
	r, err := buildWithSynthesis(seg, "eleven_flash_v2_5", nestedOptions())
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	// Independent wire fixture: zero, false, omitted versions, duplicates and
	// caller dictionary order must all survive alongside the DSL speed.
	const want = `{"text":"Hello 123","model_id":"eleven_flash_v2_5","language_code":"en","voice_settings":{"speed":1.1,"stability":0,"similarity_boost":1,"style":0,"use_speaker_boost":false},"seed":0,"apply_text_normalization":"off","apply_language_text_normalization":false,"pronunciation_dictionary_locators":[{"pronunciation_dictionary_id":"dict-b","version_id":"version-2"},{"pronunciation_dictionary_id":"dict-a"},{"pronunciation_dictionary_id":"dict-b","version_id":"version-2"}]}`
	if string(body) != want {
		t.Fatalf("body = %s; want %s", body, want)
	}
}

func TestSynthesisNestedOmissionAndExplicitEmpty(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value options.ElevenLabs
		want  string
	}{
		{"omitted", options.ElevenLabs{}, `{"text":"Hello","model_id":"eleven_multilingual_v2"}`},
		{"empty settings", options.ElevenLabs{VoiceSettings: &options.ElevenVoiceSettings{}}, `{"text":"Hello","model_id":"eleven_multilingual_v2"}`},
		{"stability zero", options.ElevenLabs{VoiceSettings: &options.ElevenVoiceSettings{Stability: optionPointer(0.0)}}, `{"text":"Hello","model_id":"eleven_multilingual_v2","voice_settings":{"stability":0}}`},
		{"similarity zero", options.ElevenLabs{VoiceSettings: &options.ElevenVoiceSettings{SimilarityBoost: optionPointer(0.0)}}, `{"text":"Hello","model_id":"eleven_multilingual_v2","voice_settings":{"similarity_boost":0}}`},
		{"style zero", options.ElevenLabs{VoiceSettings: &options.ElevenVoiceSettings{Style: optionPointer(0.0)}}, `{"text":"Hello","model_id":"eleven_multilingual_v2","voice_settings":{"style":0}}`},
		{"boost false", options.ElevenLabs{VoiceSettings: &options.ElevenVoiceSettings{UseSpeakerBoost: optionPointer(false)}}, `{"text":"Hello","model_id":"eleven_multilingual_v2","voice_settings":{"use_speaker_boost":false}}`},
		{"boost true", options.ElevenLabs{VoiceSettings: &options.ElevenVoiceSettings{UseSpeakerBoost: optionPointer(true)}}, `{"text":"Hello","model_id":"eleven_multilingual_v2","voice_settings":{"use_speaker_boost":true}}`},
		{"empty locators", options.ElevenLabs{PronunciationDictionaryLocators: &[]options.ElevenPronunciationDictionaryLocator{}}, `{"text":"Hello","model_id":"eleven_multilingual_v2","pronunciation_dictionary_locators":[]}`},
		{"nil slice locators", options.ElevenLabs{PronunciationDictionaryLocators: new([]options.ElevenPronunciationDictionaryLocator)}, `{"text":"Hello","model_id":"eleven_multilingual_v2","pronunciation_dictionary_locators":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := buildWithSynthesis(providers.Segment{Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}, "eleven_multilingual_v2", tc.value)
			if err != nil {
				t.Fatal(err)
			}
			body, err := json.Marshal(r)
			if err != nil || string(body) != tc.want {
				t.Fatalf("body=%s error=%v, want %s", body, err, tc.want)
			}
		})
	}
}

func TestSynthesisEmptyVoiceSettingsPreservesDSLSpeed(t *testing.T) {
	seg := providers.Segment{
		Nodes:    []ir.Node{{Kind: ir.Text, Text: "Hello"}},
		Controls: []ir.Node{{Name: "speed", Args: []ir.Arg{{Kind: "number", Raw: "1.1"}}}},
	}
	r, err := buildWithSynthesis(seg, "eleven_multilingual_v2", options.ElevenLabs{VoiceSettings: &options.ElevenVoiceSettings{}})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(r)
	const want = `{"text":"Hello","model_id":"eleven_multilingual_v2","voice_settings":{"speed":1.1}}`
	if err != nil || string(body) != want {
		t.Fatalf("body=%s error=%v, want %s", body, err, want)
	}
}

func TestSynthesisNestedNoInventedModelGates(t *testing.T) {
	// The common Create Speech schema does not declare these model gates.
	// Playground controls alone do not establish an API rejection constraint.
	for _, model := range []string{"eleven_multilingual_v2", "eleven_flash_v2", "eleven_flash_v2_5", "eleven_turbo_v2_5", "eleven_v3", "eleven_v4", "eleven_v4_turbo"} {
		r, err := buildWithSynthesis(providers.Segment{Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}, model, nestedOptions())
		if err != nil || r.Settings == nil || r.PronunciationDictionaryLocators == nil {
			t.Errorf("%s: request=%+v error=%v", model, r, err)
		}
	}
}

func TestSynthesisNestedSnapshotsAreIndependent(t *testing.T) {
	value := nestedOptions()
	seg := providers.Segment{Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}
	first, err := buildWithSynthesis(seg, "eleven_multilingual_v2", value)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildWithSynthesis(seg, "eleven_multilingual_v2", value)
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	*value.VoiceSettings.Stability, *value.VoiceSettings.SimilarityBoost = 0.5, 0.5
	*value.VoiceSettings.Style, *value.VoiceSettings.UseSpeakerBoost = 0.5, true
	(*value.PronunciationDictionaryLocators)[0].PronunciationDictionaryID = "changed"
	*(*value.PronunciationDictionaryLocators)[0].VersionID = "changed-version"
	*value.PronunciationDictionaryLocators = append(*value.PronunciationDictionaryLocators, options.ElevenPronunciationDictionaryLocator{})
	for _, r := range []request{first, second} {
		body, err := json.Marshal(r)
		if err != nil || !bytes.Equal(body, want) {
			t.Fatalf("caller mutation changed snapshot: %s error=%v", body, err)
		}
	}
	*first.Settings.Stability, *first.Settings.SimilarityBoost = 0.8, 0.8
	*first.Settings.Style, *first.Settings.UseSpeakerBoost = 0.8, true
	(*first.PronunciationDictionaryLocators)[0].PronunciationDictionaryID = "first-only"
	*(*first.PronunciationDictionaryLocators)[0].VersionID = "first-only"
	body, err := json.Marshal(second)
	if err != nil || !bytes.Equal(body, want) {
		t.Fatalf("requests share nested storage: %s error=%v", body, err)
	}
	if *value.VoiceSettings.Stability != 0.5 || (*value.PronunciationDictionaryLocators)[0].PronunciationDictionaryID != "changed" || *(*value.PronunciationDictionaryLocators)[0].VersionID != "changed-version" {
		t.Fatal("request mutation changed caller values")
	}
}

func synthesisDocument() *ir.Document {
	return &ir.Document{Utterances: []ir.Utterance{
		{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "First"}}},
		{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Second"}}},
	}}
}

func TestSynthesisNestedSerialConcurrentWireAndOutput(t *testing.T) {
	var serial []byte
	for _, concurrency := range []int{0, 1, 2, 8} {
		t.Run(fmt.Sprint(concurrency), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var mu sync.Mutex
				bodies := make(map[string]string)
				secondFinished := make(chan struct{})
				p := provider{client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if r.Method != http.MethodPost || r.URL.String() != "https://example.invalid/v1/text-to-speech/test-voice?output_format=pcm_24000" || r.Header.Get("xi-api-key") != "test-key" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "audio/pcm, application/octet-stream" {
						return nil, fmt.Errorf("unexpected request contract: %s %s headers=%v", r.Method, r.URL, r.Header)
					}
					body, err := io.ReadAll(r.Body)
					if err != nil {
						return nil, err
					}
					var text struct{ Text string }
					if err := json.Unmarshal(body, &text); err != nil {
						return nil, err
					}
					mu.Lock()
					bodies[text.Text] = string(body)
					mu.Unlock()
					pcm := []byte{1, 0}
					if text.Text == "Second" {
						pcm[0] = 2
						close(secondFinished)
					} else if concurrency > 1 {
						<-secondFinished
					}
					return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"audio/pcm"}}, Body: io.NopCloser(bytes.NewReader(pcm))}, nil
				})}}
				out := filepath.Join(t.TempDir(), "out.wav")
				result, err := p.Synthesize(context.Background(), synthesisDocument(), providers.Options{Out: out, Voice: "test-voice", Model: "eleven_multilingual_v2", Endpoint: "https://example.invalid/v1", APIKey: "test-key", Concurrency: concurrency, Synthesis: nestedOptions()})
				if err != nil || result.Path != out || result.Provider != "elevenlabs" {
					t.Fatalf("result=%+v error=%v", result, err)
				}
				if len(bodies) != 2 {
					t.Fatalf("requests=%v", bodies)
				}
				for _, text := range []string{"First", "Second"} {
					want := `{"text":"` + text + `","model_id":"eleven_multilingual_v2","voice_settings":{"stability":0,"similarity_boost":1,"style":0,"use_speaker_boost":false},"seed":0,"apply_text_normalization":"off","apply_language_text_normalization":false,"pronunciation_dictionary_locators":[{"pronunciation_dictionary_id":"dict-b","version_id":"version-2"},{"pronunciation_dictionary_id":"dict-a"},{"pronunciation_dictionary_id":"dict-b","version_id":"version-2"}]}`
					if bodies[text] != want {
						t.Fatalf("wire body=%s, want %s", bodies[text], want)
					}
				}
				audio, err := os.ReadFile(out)
				if err != nil || len(audio) != 48 || string(audio[:4]) != "RIFF" || !bytes.Equal(audio[44:], []byte{1, 0, 2, 0}) {
					t.Fatalf("audio=%v error=%v", audio, err)
				}
				if concurrency == 0 {
					serial = audio
				} else if !bytes.Equal(audio, serial) {
					t.Fatal("serial and concurrent output differ")
				}
			})
		})
	}
}

func TestSynthesisNestedInvalidPreflightPreservesOutput(t *testing.T) {
	for _, value := range []options.ElevenLabs{
		{VoiceSettings: &options.ElevenVoiceSettings{Stability: optionPointer(-0.1)}},
		{VoiceSettings: &options.ElevenVoiceSettings{SimilarityBoost: optionPointer(1.1)}},
		{VoiceSettings: &options.ElevenVoiceSettings{Style: optionPointer(math.NaN())}},
		{VoiceSettings: &options.ElevenVoiceSettings{Stability: optionPointer(math.Inf(1))}},
		{VoiceSettings: &options.ElevenVoiceSettings{SimilarityBoost: optionPointer(math.Inf(-1))}},
		{PronunciationDictionaryLocators: &[]options.ElevenPronunciationDictionaryLocator{{PronunciationDictionaryID: "valid"}, {PronunciationDictionaryID: ""}}},
		{PronunciationDictionaryLocators: &[]options.ElevenPronunciationDictionaryLocator{{PronunciationDictionaryID: "valid", VersionID: optionPointer("")}}},
		{PronunciationDictionaryLocators: &[]options.ElevenPronunciationDictionaryLocator{{PronunciationDictionaryID: "a"}, {PronunciationDictionaryID: "b"}, {PronunciationDictionaryID: "c"}, {PronunciationDictionaryID: "d"}}},
	} {
		for _, concurrency := range []int{0, 2} {
			var calls atomic.Int32
			p := provider{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("unexpected POST")
			})}}
			out := filepath.Join(t.TempDir(), "out.wav")
			if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := synthesisOptions(value); err == nil {
				t.Fatal("synthesis entry accepted invalid options")
			}
			_, err := p.Synthesize(context.Background(), synthesisDocument(), providers.Options{Out: out, Voice: "test-voice", Model: "eleven_multilingual_v2", Endpoint: "https://example.invalid", Concurrency: concurrency, Synthesis: value})
			if err == nil || calls.Load() != 0 {
				t.Fatalf("error=%v calls=%d", err, calls.Load())
			}
			assertNestedOutputPreserved(t, out)
		}
	}
}

func assertNestedOutputPreserved(t *testing.T, out string) {
	t.Helper()
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "KEEP" {
		t.Fatalf("output=%q error=%v", data, err)
	}
	entries, err := os.ReadDir(filepath.Dir(out))
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary output leaked: entries=%v error=%v", entries, err)
	}
}

func TestSynthesisNestedCancellationPreservesOutput(t *testing.T) {
	for _, preCanceled := range []bool{false, true} {
		t.Run(fmt.Sprint(preCanceled), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if preCanceled {
					cancel()
				}
				var calls atomic.Int32
				started := make(chan struct{}, 2)
				p := provider{client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					started <- struct{}{}
					<-r.Context().Done()
					return nil, r.Context().Err()
				})}}
				out := filepath.Join(t.TempDir(), "out.wav")
				if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
					t.Fatal(err)
				}
				done := make(chan error, 1)
				go func() {
					_, err := p.Synthesize(ctx, synthesisDocument(), providers.Options{Out: out, Voice: "test-voice", Model: "eleven_multilingual_v2", Endpoint: "https://example.invalid", Concurrency: 2, Synthesis: nestedOptions()})
					done <- err
				}()
				if !preCanceled {
					<-started
					<-started
					cancel()
				}
				if err := <-done; !errors.Is(err, context.Canceled) {
					t.Fatalf("error=%v", err)
				}
				if preCanceled && calls.Load() != 0 || !preCanceled && calls.Load() != 2 {
					t.Fatalf("calls=%d", calls.Load())
				}
				assertNestedOutputPreserved(t, out)
			})
		})
	}
}

func TestSynthesisNestedSnapshotBeforeConcurrentRender(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		value := nestedOptions()
		doc := synthesisDocument()
		doc.Utterances = append(doc.Utterances, synthesisDocument().Utterances...)
		started, release := make(chan struct{}, 2), make(chan struct{})
		var calls atomic.Int32
		p := provider{client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			started <- struct{}{}
			<-release
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			if !strings.Contains(string(body), `"voice_settings":{"stability":0,"similarity_boost":1,"style":0,"use_speaker_boost":false}`) || !strings.Contains(string(body), `"pronunciation_dictionary_id":"dict-b","version_id":"version-2"`) || strings.Contains(string(body), "mutated") {
				return nil, fmt.Errorf("caller mutation reached rendered body: %s", body)
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader([]byte{1, 0}))}, nil
		})}}
		done := make(chan error, 1)
		out := filepath.Join(t.TempDir(), "out.wav")
		go func() {
			_, err := p.Synthesize(context.Background(), doc, providers.Options{Out: out, Voice: "test-voice", Model: "eleven_multilingual_v2", Endpoint: "https://example.invalid", Concurrency: 2, Synthesis: value})
			done <- err
		}()
		<-started
		<-started
		// The first two requests entered transport; the remaining two are not
		// marshaled yet. They must already own their nested preflight snapshots.
		*value.VoiceSettings.Stability, *value.VoiceSettings.SimilarityBoost = 0.7, 0.7
		*value.VoiceSettings.Style, *value.VoiceSettings.UseSpeakerBoost = 0.7, true
		(*value.PronunciationDictionaryLocators)[0].PronunciationDictionaryID = "mutated"
		*(*value.PronunciationDictionaryLocators)[0].VersionID = "mutated"
		*value.PronunciationDictionaryLocators = nil
		close(release)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if calls.Load() != 4 {
			t.Fatalf("calls=%d, want 4", calls.Load())
		}
	})
}

func TestSynthesisNestedLaterSegmentPreflight(t *testing.T) {
	var calls atomic.Int32
	p := provider{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("unexpected POST")
	})}}
	doc := synthesisDocument()
	doc.Utterances[1].Nodes = []ir.Node{{Kind: ir.State, Name: "fast"}, {Kind: ir.Text, Text: "Second"}}
	out := filepath.Join(t.TempDir(), "out.wav")
	if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := p.Synthesize(context.Background(), doc, providers.Options{Out: out, Voice: "test-voice", Model: "eleven_v4", Endpoint: "https://example.invalid", Concurrency: 2, Synthesis: nestedOptions()})
	if err == nil || !strings.Contains(err.Error(), "[fast]") || calls.Load() != 0 {
		t.Fatalf("error=%v calls=%d", err, calls.Load())
	}
	assertNestedOutputPreserved(t, out)
}
