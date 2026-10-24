package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

type embeddedOptions struct{ options.Value }

func TestAdaptersRejectInvalidSynthesisOptionsBeforeSideEffects(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	badSeed := int64(4294967296)
	badNorm := options.ElevenTextNormalization("invalid")
	badTemperature := math.NaN()
	badQwenSeed := int64(65536)
	badMode := options.MiniMaxTextNormalization("invalid")
	badDeployment := "\x00"
	for _, id := range providers.IDs() {
		t.Run(id, func(t *testing.T) {
			p, err := providers.Get(id)
			if err != nil {
				t.Fatal(err)
			}
			variants := []options.Value{
				(*options.ElevenLabs)(nil), &options.ElevenLabs{},
				(*options.Fish)(nil), &options.Fish{},
				(*options.Qwen)(nil), &options.Qwen{},
				(*options.MiniMax)(nil), &options.MiniMax{},
				(*options.Azure)(nil), &options.Azure{},
				embeddedOptions{}, (*embeddedOptions)(nil),
				options.ElevenLabs{Seed: &badSeed},
				options.ElevenLabs{ApplyTextNormalization: &badNorm},
				options.Fish{Temperature: &badTemperature},
				options.Qwen{Seed: &badQwenSeed},
				options.MiniMax{TextNormalizationMode: &badMode},
				options.Azure{DeploymentID: &badDeployment},
				options.Fish{PronunciationDictionary: &options.FishDictionaries{}},
			}
			for _, variant := range []struct {
				owner string
				value options.Value
			}{
				{"elevenlabs", options.ElevenLabs{}}, {"fish", options.Fish{}},
				{"qwen", options.Qwen{}}, {"minimax", options.MiniMax{}}, {"azure", options.Azure{}},
			} {
				if id != variant.owner {
					variants = append(variants, variant.value)
				}
				variants = append(variants, embeddedOptions{Value: variant.value})
			}
			out := filepath.Join(t.TempDir(), "out.wav")
			if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			for _, doc := range []*ir.Document{
				nil,
				{},
				{Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}}},
			} {
				for _, value := range variants {
					opt := providers.Options{Out: out, Endpoint: srv.URL, Voice: "test-voice", Synthesis: value}
					_, err := p.Synthesize(context.Background(), doc, opt)
					if err == nil || !strings.Contains(err.Error(), "synthesis options") && !strings.Contains(err.Error(), "seed") && !strings.Contains(err.Error(), "apply_text_normalization") && !strings.Contains(err.Error(), "temperature") && !strings.Contains(err.Error(), "text_normalization_mode") && !strings.Contains(err.Error(), "deployment_id") && !strings.Contains(err.Error(), "pronunciation_dictionary") {
						t.Fatalf("invalid options did not fail at the adapter boundary: %v", err)
					}
					data, err := os.ReadFile(out)
					if err != nil || string(data) != "KEEP" {
						t.Fatalf("existing output changed: %q, %v", data, err)
					}
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					if _, err := p.Synthesize(ctx, doc, opt); !errors.Is(err, context.Canceled) {
						t.Fatalf("cancellation must still precede validation: %v", err)
					}
				}
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid options made %d requests", calls.Load())
	}
}

func TestAdapterOptionValidationOrdering(t *testing.T) {
	for _, id := range providers.IDs() {
		t.Run(id, func(t *testing.T) {
			p, err := providers.Get(id)
			if err != nil {
				t.Fatal(err)
			}
			opt := providers.Options{Concurrency: -1, Synthesis: &options.ElevenLabs{}}
			if _, err := p.Synthesize(context.Background(), nil, opt); err == nil || !strings.Contains(err.Error(), "concurrency") {
				t.Fatalf("concurrency must precede option and document validation: %v", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := p.Synthesize(ctx, nil, opt); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation must precede concurrency validation: %v", err)
			}
		})
	}
}

func TestElevenLabsConfiguredSynthesisConcurrency(t *testing.T) {
	vendor := vendors[1]
	baseline, baselineWAV := runConcurrentContract(t, vendor, 0, "")
	for _, tc := range []struct {
		name, config string
		fields       map[string]any
	}{
		{"empty", "{}", nil},
		{"zero_false", "{seed: 0, apply_text_normalization: off, apply_language_text_normalization: false}", map[string]any{"seed": float64(0), "apply_text_normalization": "off", "apply_language_text_normalization": false}},
		{"max_true", "{seed: 4294967295, apply_text_normalization: on, apply_language_text_normalization: true}", map[string]any{"seed": float64(4294967295), "apply_text_normalization": "on", "apply_language_text_normalization": true}},
		{"auto", "{apply_text_normalization: auto}", map[string]any{"apply_text_normalization": "auto"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var reference map[string]concurrencyRequest
			for _, workers := range []int{0, 1, 2, 4, 8} {
				t.Run(fmt.Sprintf("workers_%d", workers), func(t *testing.T) {
					config := "providers: {elevenlabs: {synthesis: " + tc.config + "}}"
					requests, wav := runConcurrentContract(t, vendor, workers, config)
					if !bytes.Equal(wav, baselineWAV) {
						t.Fatal("synthesis options changed fixture output order or pauses")
					}
					if workers == 0 {
						reference = requests
					} else if !reflect.DeepEqual(requests, reference) {
						t.Fatal("concurrency changed complete configured requests")
					}
					for key, request := range requests {
						var expected map[string]any
						if err := json.Unmarshal([]byte(baseline[key].Body), &expected); err != nil {
							t.Fatal(err)
						}
						for field, value := range tc.fields {
							expected[field] = value
						}
						assertJSONRequest(t, []byte(request.Body), expected)
					}
					if tc.fields == nil && !reflect.DeepEqual(requests, baseline) {
						t.Fatal("empty options changed the existing complete request contract")
					}
				})
			}
		})
	}
}

func TestConfiguredSynthesisPreservesWholeScriptPreflight(t *testing.T) {
	clearProviderDefaults(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("ELEVENLABS_API_KEY", "test-key")
	t.Setenv("ELEVENLABS_BASE_URL", srv.URL)
	for _, workers := range []int{1, 4} {
		for _, tc := range []struct{ model, later string }{
			{"eleven_multilingual_v2", "[lang ja|今日は10月24日です。]"},
			{"eleven_flash_v2", "[lang ja|今日は10月24日です。]"},
			{"eleven_v4", "[speed 1.1|Later]"},
		} {
			path, out := concurrencyFiles(t, "---\nprovider: elevenlabs\nvoice: test-voice\nmodel: "+tc.model+"\n---\nFirst\n\n"+tc.later+"\n")
			configPath := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(configPath, []byte("providers: {elevenlabs: {model: eleven_v3, synthesis: {seed: 0, apply_text_normalization: on, apply_language_text_normalization: true}}}"), 0o600); err != nil {
				t.Fatal(err)
			}
			err := run(context.Background(), []string{"--config", configPath, "synth", path, "-o", out, "--concurrency", fmt.Sprint(workers)}, nil, &bytes.Buffer{}, &bytes.Buffer{})
			if err == nil || calls.Load() != 0 {
				t.Fatalf("invalid later segment: error=%v, requests=%d", err, calls.Load())
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != "KEEP" {
				t.Fatalf("output changed: %q, %v", data, err)
			}
		}
	}
}
