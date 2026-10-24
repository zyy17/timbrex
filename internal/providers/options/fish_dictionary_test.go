package options

import (
	"strings"
	"testing"
)

func fishInline(items []FishPronunciationItem) Fish {
	inline := []FishDictionaryInline{{Items: items}}
	return Fish{PronunciationDictionary: &FishDictionaries{Inline: &inline}}
}

func TestFishDictionaryBoundaries(t *testing.T) {
	for _, count := range []int{0, 1, 3, 4} {
		refs := make([]FishDictionaryReference, count)
		inline := make([]FishDictionaryInline, count)
		for i := range refs {
			refs[i] = FishDictionaryReference{ID: "dictionary_1", Version: "version-2"}
			inline[i].Items = []FishPronunciationItem{}
		}
		for _, dictionaries := range []FishDictionaries{{References: &refs}, {Inline: &inline}} {
			if err := (Fish{PronunciationDictionary: &dictionaries}).Validate(); (err == nil) != (count <= 3) {
				t.Errorf("dictionary count %d: %v", count, err)
			}
		}
	}
	for _, count := range []int{0, 1, 5000, 5001} {
		items := make([]FishPronunciationItem, count)
		for i := range items {
			items[i] = FishPronunciationItem{Key: "重复", Value: "phonetic"}
		}
		if err := fishInline(items).Validate(); (err == nil) != (count <= 5000) {
			t.Errorf("item count %d: %v", count, err)
		}
	}
	refs, inline := []FishDictionaryReference{}, []FishDictionaryInline{}
	for _, value := range []FishDictionaries{{}, {References: &refs, Inline: &inline}} {
		if err := value.Validate(); err == nil {
			t.Fatal("accepted absent or mixed union branches")
		}
	}
	if err := fishInline(nil).Validate(); err == nil {
		t.Fatal("accepted missing required items")
	}
}

func TestFishDictionaryIdentifiers(t *testing.T) {
	for _, text := range []string{"", "x", "_-Ab09", strings.Repeat("x", 128), strings.Repeat("x", 129), "dictionary.id", "v 1", "字", "x\n", "x\x00", "\xff"} {
		valid := text == "x" || text == "_-Ab09" || text == strings.Repeat("x", 128)
		for _, ref := range []FishDictionaryReference{{ID: text, Version: "version"}, {ID: "dictionary", Version: text}} {
			refs := []FishDictionaryReference{ref}
			if err := (FishDictionaries{References: &refs}).Validate(); (err == nil) != valid {
				t.Errorf("identifier %q: %v", text, err)
			}
		}
	}
}

func TestFishDictionaryUnicodeAndMarkers(t *testing.T) {
	for _, tc := range []struct {
		key, value string
		valid      bool
	}{
		{"word", "wɜːd", true}, {"重庆", "tʂʰʊŋ tɕʰiŋ", true}, {"a", "a\u0301", true},
		{" ", " ", true}, {"x", "a\n\x00b", true},
		{strings.Repeat("界", 256), strings.Repeat("ə", 1024), true},
		{strings.Repeat("界", 257), "value", false}, {"key", strings.Repeat("ə", 1025), false},
		{"", "value", false}, {"key", "", false}, {"\xff", "value", false}, {"key", "\xff", false},
		{"<|phoneme_start|>", "value", false}, {"key", "a<|phoneme_end|>b", false},
		{"<|phoneme_end|>", "value", false}, {"key", "<|phoneme_start|>", false},
	} {
		if err := fishInline([]FishPronunciationItem{{Key: tc.key, Value: tc.value}}).Validate(); (err == nil) != tc.valid {
			t.Errorf("key %q value %q: %v", tc.key, tc.value, err)
		}
	}
}

func TestFishQualityGuardKnownOnly(t *testing.T) {
	for _, features := range [][]FishFeature{{}, {FishQualityGuard}, {FishQualityGuard, FishQualityGuard}, {"unknown"}, {""}, {"QUALITY-GUARD"}} {
		valid := len(features) == 0 || features[0] == FishQualityGuard
		if err := (Fish{Features: &features}).Validate(); (err == nil) != valid {
			t.Errorf("features %v: %v", features, err)
		}
	}
}
