package config

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/providers/options"
)

func TestFishDictionariesConfigurationRoundTrip(t *testing.T) {
	for _, body := range []string{
		`{features: [], pronunciation_dictionary: []}`,
		`{features: [quality-guard], pronunciation_dictionary: [{id: first, version: v_1}, {id: second, version: v-2}]}`,
		`{pronunciation_dictionary: [{items: []}]}`,
		`{pronunciation_dictionary: [{items: [{key: 重庆, value: phonetic}, {key: word, value: 'wɜːd', case_sensitive: false}]}, {items: [{key: second, value: value, case_sensitive: true}]}]}`,
	} {
		t.Run(body, func(t *testing.T) {
			c, err := loadTest(t, "providers: {fish: {synthesis: "+body+"}}")
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := c.RedactedYAML()
			if err != nil {
				t.Fatal(err)
			}
			copy, err := loadTest(t, string(encoded))
			if err != nil || !reflect.DeepEqual(c.Providers["fish"].Synthesis, copy.Providers["fish"].Synthesis) {
				t.Fatalf("roundtrip failed: %v\n%s", err, encoded)
			}
		})
	}
}

func TestFishDictionariesStrictConfiguration(t *testing.T) {
	for _, body := range []string{
		`{features: null}`, `{features: {}}`, `{features: [null]}`, `{features: [true]}`, `{features: [0]}`, `{features: [secret-value]}`,
		`{pronunciation_dictionary: null}`, `{pronunciation_dictionary: {}}`, `{pronunciation_dictionary: [null]}`, `{pronunciation_dictionary: [secret-value]}`,
		`{pronunciation_dictionary: [{}]}`, `{pronunciation_dictionary: [{id: d}]}`, `{pronunciation_dictionary: [{version: v}]}`,
		`{pronunciation_dictionary: [{id: null, version: v}]}`, `{pronunciation_dictionary: [{id: 123, version: v}]}`, `{pronunciation_dictionary: [{id: d, version: null}]}`, `{pronunciation_dictionary: [{id: d, version: 1}]}`,
		`{pronunciation_dictionary: [{id: d, version: v, secret-value: x}]}`, `{pronunciation_dictionary: [{id: d, version.id: v}]}`, `{pronunciation_dictionary.items: []}`,
		`{pronunciation_dictionary: [{id: d, version: v}, {items: []}]}`, `{pronunciation_dictionary: [{items: []}, {id: d, version: v}]}`,
		`{pronunciation_dictionary: [{items: [], id: d, version: v}]}`, `{pronunciation_dictionary: [{items: [], secret-value: x}]}`,
		`{pronunciation_dictionary: [{items: null}]}`, `{pronunciation_dictionary: [{items: {}}]}`, `{pronunciation_dictionary: [{items: [null]}]}`, `{pronunciation_dictionary: [{items: [secret-value]}]}`,
		`{pronunciation_dictionary: [{items: [{}]}]}`, `{pronunciation_dictionary: [{items: [{key: a}]}]}`, `{pronunciation_dictionary: [{items: [{value: b}]}]}`,
		`{pronunciation_dictionary: [{items: [{key: null, value: b}]}]}`, `{pronunciation_dictionary: [{items: [{key: a, value: null}]}]}`,
		`{pronunciation_dictionary: [{items: [{key: 1, value: b}]}]}`, `{pronunciation_dictionary: [{items: [{key: a, value: true}]}]}`,
		`{pronunciation_dictionary: [{items: [{key: a, value: b, case_sensitive: null}]}]}`, `{pronunciation_dictionary: [{items: [{key: a, value: b, case_sensitive: 'false'}]}]}`,
		`{pronunciation_dictionary: [{items: [{key: a, value: b, case.sensitive: false}]}]}`, `{pronunciation_dictionary: [{items: [{key: a, value: b, secret-value: x}]}]}`,
		`{pronunciation_dictionary: [{items: [{key: a, key: b, value: b}]}]}`, `{features: [], features: [quality-guard]}`,
	} {
		t.Run(body, func(t *testing.T) {
			_, err := loadTest(t, "provider: stub\nproviders: {fish: {synthesis: "+body+"}}", "FISH_MODEL=s2-pro")
			if err == nil || strings.Contains(err.Error(), "secret-value") {
				t.Fatalf("invalid value accepted or leaked: %v", err)
			}
		})
	}
}

func TestFishDictionaryConfigurationLimits(t *testing.T) {
	for _, count := range []int{3, 4} {
		body := "providers: {fish: {synthesis: {pronunciation_dictionary: [" + strings.Repeat("{id: d, version: v},", count) + "]}}}"
		_, err := loadTest(t, body)
		if (err == nil) != (count == 3) {
			t.Errorf("dictionary count %d: %v", count, err)
		}
	}
	for _, count := range []int{5000, 5001} {
		body := "providers: {fish: {synthesis: {pronunciation_dictionary: [{items: [" + strings.Repeat("{key: k, value: v},", count) + "]}]}}}"
		_, err := loadTest(t, body)
		if (err == nil) != (count == 5000) {
			t.Errorf("item count %d: %v", count, err)
		}
	}
	for _, count := range []int{256, 257} {
		body := fmt.Sprintf("providers: {fish: {synthesis: {pronunciation_dictionary: [{items: [{key: '%s', value: v}]}]}}}", strings.Repeat("界", count))
		_, err := loadTest(t, body)
		if (err == nil) != (count == 256) {
			t.Errorf("key rune count %d: %v", count, err)
		}
	}
	value, err := options.Decode("fish", map[string]any{"pronunciation_dictionary": []any{map[string]any{"items": []any{map[string]any{"key": "\xff", "value": "valid"}}}}})
	if err == nil {
		t.Fatalf("accepted invalid UTF-8: %#v", value)
	}
	c, err := loadTest(t, "providers: {fish: {synthesis: {}}}")
	if err != nil {
		t.Fatal(err)
	}
	v := c.Providers["fish"].Synthesis.(options.Fish)
	if v.Features != nil || v.PronunciationDictionary != nil {
		t.Fatal("omission became an explicit empty list")
	}
}
