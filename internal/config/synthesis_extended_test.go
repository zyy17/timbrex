package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestExtendedSynthesisConfigurationRoundTrip(t *testing.T) {
	for _, tc := range []struct{ id, body string }{
		{"elevenlabs", `{voice_settings: {stability: 0, similarity_boost: 1, style: 0.25, use_speaker_boost: false}, pronunciation_dictionary_locators: [{pronunciation_dictionary_id: dictionary-one}, {pronunciation_dictionary_id: dictionary-two, version_id: revision-two}, {pronunciation_dictionary_id: dictionary-one}]}`},
		{"elevenlabs", `{voice_settings: {}, pronunciation_dictionary_locators: []}`},
		{"fish", `{normalize: false, normalization_language: en-US, prosody: {normalize_loudness: false}, temperature: 0, top_p: 1, chunk_length: 100, min_chunk_length: 0, latency: balanced, condition_on_previous_chunks: false, early_stop_threshold: 0}`},
		{"fish", `{normalize: true, normalization_language: '', prosody: {}, temperature: 1.0, top_p: 0.0, chunk_length: 300, min_chunk_length: 100, latency: low, condition_on_previous_chunks: true, early_stop_threshold: 1}`},
		{"qwen", `{seed: 0, hot_fix: {pronunciation: [{text: 天气, replacement: tian1 qi4}, {text: 天气, replacement: tian2 qi4}], replace: [{text: 今天, replacement: 金天}, {text: remove, replacement: ''}]}}`},
		{"qwen", `{seed: 65535, hot_fix: {pronunciation: [], replace: []}}`},
		{"qwen", `{hot_fix: {}}`},
		{"minimax", `{text_normalization_mode: basic}`},
		{"minimax", `{text_normalization_mode: quality}`},
	} {
		t.Run(tc.id+"/"+tc.body, func(t *testing.T) {
			c, err := loadTest(t, "providers: {"+tc.id+": {api_key: FAKE-SECRET-DO-NOT-LOG, synthesis: "+tc.body+"}}")
			if err != nil {
				t.Fatal(err)
			}
			value := c.Providers[tc.id].Synthesis
			if value == nil {
				t.Fatal("configuration did not retain typed options")
			}
			redacted, err := c.RedactedYAML()
			if err != nil || strings.Contains(string(redacted), "FAKE-SECRET-DO-NOT-LOG") {
				t.Fatalf("redaction failed: %v", err)
			}
			copy, err := loadTest(t, string(redacted))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(value, copy.Providers[tc.id].Synthesis) {
				t.Fatalf("round trip changed explicit values or presence:\n%+v\n%+v", value, copy.Providers[tc.id].Synthesis)
			}
			if c.Providers[tc.id].APIKey != "FAKE-SECRET-DO-NOT-LOG" {
				t.Fatal("redaction mutated the source configuration")
			}
		})
	}
}

func TestExtendedSynthesisStrictConfiguration(t *testing.T) {
	invalid := map[string][]string{
		"elevenlabs": {
			`{voice_settings: null}`, `{voice_settings: []}`, `{voice_settings: secret-value}`,
			`{voice_settings: {stability: null}}`, `{voice_settings: {stability: true}}`, `{voice_settings: {stability: '0'}}`, `{voice_settings: {stability: -.1}}`, `{voice_settings: {stability: 1.1}}`,
			`{voice_settings: {similarity_boost: .nan}}`, `{voice_settings: {similarity_boost: -1}}`, `{voice_settings: {similarity_boost: 2}}`,
			`{voice_settings: {style: .inf}}`, `{voice_settings: {style: -.inf}}`, `{voice_settings: {style: false}}`, `{voice_settings: {style: '1'}}`,
			`{voice_settings: {use_speaker_boost: null}}`, `{voice_settings: {use_speaker_boost: 'false'}}`, `{voice_settings: {use_speaker_boost: 0}}`,
			`{voice_settings: {speed: 1}}`, `{voice_settings.stability: 0}`, `{voice_settings: {secret-value: secret-value}}`,
			`{pronunciation_dictionary_locators: null}`, `{pronunciation_dictionary_locators: {}}`, `{pronunciation_dictionary_locators: [null]}`, `{pronunciation_dictionary_locators: [secret-value]}`,
			`{pronunciation_dictionary_locators: [{}]}`, `{pronunciation_dictionary_locators: [{id: secret-value}]}`, `{pronunciation_dictionary_locators: [{pronunciation_dictionary_id: null}]}`, `{pronunciation_dictionary_locators: [{pronunciation_dictionary_id: 1}]}`,
			`{pronunciation_dictionary_locators: [{pronunciation_dictionary_id: ''}]}`, `{pronunciation_dictionary_locators: [{pronunciation_dictionary_id: d, version_id: null}]}`, `{pronunciation_dictionary_locators: [{pronunciation_dictionary_id: d, version_id: 1}]}`, `{pronunciation_dictionary_locators: [{pronunciation_dictionary_id: d, version_id: ''}]}`,
			`{pronunciation_dictionary_locators: [{pronunciation_dictionary_id: d, secret-value: secret-value}]}`, `{pronunciation_dictionary_locators: [{pronunciation_dictionary_id: d, version.id: x}]}`,
			`{pronunciation_dictionary_locators: [{pronunciation_dictionary_id: a}, {pronunciation_dictionary_id: b}, {pronunciation_dictionary_id: c}, {pronunciation_dictionary_id: d}]}`,
		},
		"fish": {
			`null`, `[]`, `true`, `{seed: 0}`, `{normalize: null}`, `{normalize: 'false'}`, `{normalize: 0}`, `{normalization_language: null}`, `{normalization_language: 1}`, `{normalization_language: 中文}`, `{normalization_language: 'en US'}`, `{normalization_language: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa}`, `{language: en}`,
			`{prosody: null}`, `{prosody: []}`, `{prosody: {normalize_loudness: null}}`, `{prosody: {normalize_loudness: 'false'}}`, `{prosody: {speed: 1}}`, `{prosody.normalize_loudness: true}`,
			`{temperature: -.1}`, `{temperature: 1.1}`, `{temperature: .nan}`, `{temperature: .inf}`, `{temperature: '0'}`, `{temperature: false}`, `{temperature: null}`, `{top_p: -1}`, `{top_p: 2}`, `{top_p: -.inf}`,
			`{chunk_length: 99}`, `{chunk_length: 301}`, `{chunk_length: 100.0}`, `{chunk_length: 1e2}`, `{chunk_length: '100'}`, `{chunk_length: 18446744073709551615}`, `{chunk_length: 18446744073709551616}`, `{min_chunk_length: -1}`, `{min_chunk_length: 101}`, `{min_chunk_length: 0.0}`,
			`{latency: ''}`, `{latency: LOW}`, `{latency: true}`, `{condition_on_previous_chunks: 'false'}`, `{early_stop_threshold: -1}`, `{early_stop_threshold: 2}`, `{early_stop_threshold: .nan}`, `{features: [secret-value]}`, `{max_new_tokens: 10}`, `{repetition_penalty: 1.2}`, `{secret-value: secret-value}`,
		},
		"qwen": {
			`null`, `[]`, `{seed: -1}`, `{seed: 65536}`, `{seed: 1.0}`, `{seed: '0'}`, `{seed: true}`, `{seed: null}`, `{seed: .nan}`, `{seed: 18446744073709551615}`,
			`{hot_fix: null}`, `{hot_fix: []}`, `{hot_fix: {pronunciation: null}}`, `{hot_fix: {pronunciation: {}}}`, `{hot_fix: {replace: [null]}}`, `{hot_fix: {replace: [{}]}}`,
			`{hot_fix: {replace: [{text: x}]}}`, `{hot_fix: {replace: [{replacement: y}]}}`, `{hot_fix: {replace: [{text: null, replacement: y}]}}`, `{hot_fix: {replace: [{text: x, replacement: null}]}}`, `{hot_fix: {replace: [{text: 1, replacement: y}]}}`, `{hot_fix: {replace: [{text: x, replacement: false}]}}`, `{hot_fix: {replace: [{text: '', replacement: y}]}}`,
			`{hot_fix: {pronunciation: [{天气: tian1 qi4}]}}`, `{hot_fix: {replace: [{text: x, replacement: y, secret-value: secret-value}]}}`, `{hot_fix.replace: []}`, `{hot_fix: {replace.secret-value: []}}`, `{replace: []}`, `{input: {seed: 0}}`, `{secret-value: secret-value}`,
		},
		"minimax": {
			`null`, `[]`, `{text_normalization_mode: null}`, `{text_normalization_mode: true}`, `{text_normalization_mode: 0}`, `{text_normalization_mode: Quality}`, `{text_normalization_mode: ''}`, `{text_normalization_mode: secret-value}`, `{voice_setting: {text_normalization: true}}`, `{seed: 0}`, `{secret-value: secret-value}`,
		},
	}
	for id, values := range invalid {
		for _, body := range values {
			t.Run(id+"/"+body, func(t *testing.T) {
				_, err := loadTest(t, "providers: {"+id+": {synthesis: "+body+"}}", strings.ToUpper(id)+"_MODEL=override-model")
				if err == nil || strings.Contains(err.Error(), "secret-value") {
					t.Fatalf("invalid options accepted or leaked arbitrary input: %v", err)
				}
			})
		}
	}
}

func TestExtendedSynthesisAbsentEmptyAndLiteralKeys(t *testing.T) {
	for _, id := range []string{"elevenlabs", "fish", "qwen", "minimax"} {
		c, err := loadTest(t, "")
		if err != nil || c.Providers[id].Synthesis != nil {
			t.Fatalf("omission changed %s options: %v", id, err)
		}
		c, err = loadTest(t, "providers: {"+id+": {synthesis: {}}}")
		if err != nil || c.Providers[id].Synthesis == nil {
			t.Fatalf("empty %s options rejected: %v", id, err)
		}
		for _, body := range []string{
			"providers: {" + id + ".synthesis: {}}",
			"providers." + id + ".synthesis: {}",
			"providers: {" + id + ": {synthesis: {}, synthesis.secret-value: true}}",
		} {
			if _, err := loadTest(t, body); err == nil || strings.Contains(err.Error(), "secret-value") {
				t.Fatalf("accepted or leaked a dotted schema key for %s", id)
			}
		}
	}
}

func TestExtendedSynthesisExamples(t *testing.T) {
	for _, id := range []string{"elevenlabs", "fish", "qwen", "minimax"} {
		c, err := load("../../examples/"+id+"-synthesis.yaml", func() []string { return nil })
		if err != nil || c.Provider != id || c.Providers[id].Synthesis == nil {
			t.Fatalf("invalid %s example: %v", id, err)
		}
	}
}
