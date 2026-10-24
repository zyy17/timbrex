package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
)

// These expected JSON additions come from the documented wire contracts, not
// the adapters' private request types. Exercise YAML through ordered audio output.
func TestExtendedConfiguredSynthesisConcurrency(t *testing.T) {
	for _, tc := range []struct {
		id, name, body string
		fields         string
	}{
		{"elevenlabs", "settings_zero_false", `{voice_settings: {stability: 0, similarity_boost: 1, style: 0, use_speaker_boost: false}, pronunciation_dictionary_locators: [{pronunciation_dictionary_id: dictionary-one}, {pronunciation_dictionary_id: dictionary-two, version_id: revision-two}, {pronunciation_dictionary_id: dictionary-one}]}`, `{"voice_settings":{"stability":0,"similarity_boost":1,"style":0,"use_speaker_boost":false},"pronunciation_dictionary_locators":[{"pronunciation_dictionary_id":"dictionary-one"},{"pronunciation_dictionary_id":"dictionary-two","version_id":"revision-two"},{"pronunciation_dictionary_id":"dictionary-one"}]}`},
		{"elevenlabs", "empty_dictionaries", `{voice_settings: {}, pronunciation_dictionary_locators: []}`, `{"pronunciation_dictionary_locators":[]}`},
		{"fish", "bounds_zero_false", `{normalize: true, normalization_language: en-US, prosody: {normalize_loudness: false}, temperature: 0, top_p: 1, chunk_length: 100, min_chunk_length: 0, latency: balanced, condition_on_previous_chunks: false, early_stop_threshold: 0}`, `{"normalize":true,"language":"en-US","prosody":{"normalize_loudness":false},"temperature":0,"top_p":1,"chunk_length":100,"min_chunk_length":0,"latency":"balanced","condition_on_previous_chunks":false,"early_stop_threshold":0}`},
		{"fish", "bounds_max_true", `{normalize: false, normalization_language: '', prosody: {normalize_loudness: true}, temperature: 1, top_p: 0, chunk_length: 300, min_chunk_length: 100, latency: low, condition_on_previous_chunks: true, early_stop_threshold: 1}`, `{"normalize":false,"language":"","prosody":{"normalize_loudness":true},"temperature":1,"top_p":0,"chunk_length":300,"min_chunk_length":100,"latency":"low","condition_on_previous_chunks":true,"early_stop_threshold":1}`},
		{"qwen", "ordered_unicode_pairs", `{seed: 0, hot_fix: {pronunciation: [{text: 天气, replacement: tian1 qi4}, {text: 天气, replacement: tian2 qi4}], replace: [{text: 今天, replacement: 金天}, {text: 今, replacement: ''}]}}`, `{"input":{"seed":0,"hot_fix":{"pronunciation":[{"天气":"tian1 qi4"},{"天气":"tian2 qi4"}],"replace":[{"今天":"金天"},{"今":""}]}}}`},
		{"qwen", "max_empty_pairs", `{seed: 65535, hot_fix: {pronunciation: [], replace: []}}`, `{"input":{"seed":65535,"hot_fix":{"pronunciation":[],"replace":[]}}}`},
		{"minimax", "basic", `{text_normalization_mode: basic}`, `{"text_normalization_mode":"basic"}`},
		{"minimax", "quality", `{text_normalization_mode: quality}`, `{"text_normalization_mode":"quality"}`},
	} {
		t.Run(tc.id+"/"+tc.name, func(t *testing.T) {
			vendor := synthesisVendor(t, tc.id)
			baseline, baselineWAV := runConcurrentContract(t, vendor, 0, "")
			var reference map[string]concurrencyRequest
			for _, workers := range []int{0, 1, 2, 4, 8} {
				t.Run(fmt.Sprintf("workers_%d", workers), func(t *testing.T) {
					requests, wav := runConcurrentContract(t, vendor, workers, "providers: {"+tc.id+": {synthesis: "+tc.body+"}}")
					if !bytes.Equal(wav, baselineWAV) {
						t.Fatal("configured synthesis changed ordered fixture audio or pauses")
					}
					if workers == 0 {
						reference = requests
					} else if !reflect.DeepEqual(requests, reference) {
						t.Fatal("concurrency changed complete configured requests")
					}
					for key, request := range requests {
						if request.Method != http.MethodPost {
							if !reflect.DeepEqual(request, baseline[key]) {
								t.Fatal("synthesis options changed an audio download")
							}
							continue
						}
						metadata, baselineMetadata := request, baseline[key]
						metadata.Body, baselineMetadata.Body = "", ""
						metadata.Header = metadata.Header.Clone()
						baselineMetadata.Header = baselineMetadata.Header.Clone()
						metadata.Header.Del("Content-Length")
						baselineMetadata.Header.Del("Content-Length")
						if !reflect.DeepEqual(metadata, baselineMetadata) {
							t.Fatal("synthesis options changed POST routing or unrelated headers")
						}
						var expected, additions map[string]any
						if err := json.Unmarshal([]byte(baseline[key].Body), &expected); err != nil {
							t.Fatal(err)
						}
						if err := json.Unmarshal([]byte(tc.fields), &additions); err != nil {
							t.Fatal(err)
						}
						mergeExpectedFields(expected, additions)
						assertJSONRequest(t, []byte(request.Body), expected)
					}
				})
			}
		})
	}
}

func mergeExpectedFields(target, additions map[string]any) {
	for key, value := range additions {
		object, nested := value.(map[string]any)
		existing, exists := target[key].(map[string]any)
		if nested && exists {
			mergeExpectedFields(existing, object)
		} else {
			target[key] = value
		}
	}
}

func synthesisVendor(t *testing.T, id string) vendorTest {
	t.Helper()
	for _, vendor := range concurrencyVendors() {
		if vendor.id == id {
			return vendor
		}
	}
	t.Fatalf("missing vendor fixture %s", id)
	return vendorTest{}
}

func TestExtendedEmptySynthesisPreservesDefaults(t *testing.T) {
	for _, tc := range []struct{ id, body string }{
		{"elevenlabs", `{voice_settings: {}}`}, {"fish", `{prosody: {}}`},
		{"qwen", `{hot_fix: {}}`}, {"minimax", `{}`},
	} {
		t.Run(tc.id, func(t *testing.T) {
			vendor := synthesisVendor(t, tc.id)
			baseline, wav := runConcurrentContract(t, vendor, 0, "")
			for _, body := range []string{"{}", tc.body} {
				requests, audio := runConcurrentContract(t, vendor, 4, "providers: {"+tc.id+": {synthesis: "+body+"}}")
				if !reflect.DeepEqual(requests, baseline) || !bytes.Equal(audio, wav) {
					t.Fatal("empty synthesis settings changed the complete baseline")
				}
			}
		})
	}
}

func TestExtendedConfiguredPreflightPreservesOutput(t *testing.T) {
	for _, tc := range []struct{ id, body, model, last string }{
		{"elevenlabs", `{voice_settings: {stability: 0}, pronunciation_dictionary_locators: [{pronunciation_dictionary_id: example}]}`, "eleven_v3", `[pitch 1st|Last]`},
		{"fish", `{normalization_language: en, prosody: {normalize_loudness: false}}`, "s2.1-pro", `[lang en|Last]`},
		{"qwen", `{seed: 0, hot_fix: {replace: [{text: Last, replacement: final}]}}`, "qwen-audio-3.1-tts-flash", `[lang xx|Last]`},
		{"minimax", `{text_normalization_mode: quality}`, "speech-2.8-hd", `[lang xx|Last]`},
		{"minimax", `{text_normalization_mode: quality}`, "speech-02-hd", `Last`},
	} {
		t.Run(tc.id+"/"+tc.model, func(t *testing.T) {
			clearProviderDefaults(t)
			vendor := synthesisVendor(t, tc.id)
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
			defer srv.Close()
			t.Setenv(vendor.keyEnv, "test-key")
			t.Setenv(vendor.endpointEnv, srv.URL)
			for _, workers := range []int{1, 4} {
				path, out := concurrencyFiles(t, "---\nprovider: "+tc.id+"\nvoice: test-voice\nmodel: "+tc.model+"\n---\nFirst\n\n"+tc.last+"\n")
				configPath := filepath.Join(t.TempDir(), "config.yaml")
				if err := os.WriteFile(configPath, []byte("providers: {"+tc.id+": {synthesis: "+tc.body+"}}"), 0o600); err != nil {
					t.Fatal(err)
				}
				err := run(context.Background(), []string{"--config", configPath, "synth", path, "-o", out, "--concurrency", fmt.Sprint(workers)}, nil, &bytes.Buffer{}, &bytes.Buffer{})
				if err == nil || calls.Load() != 0 {
					t.Fatalf("invalid late segment/model: error=%v, requests=%d", err, calls.Load())
				}
				assertConcurrentOutputPreserved(t, out)
			}
		})
	}
}
