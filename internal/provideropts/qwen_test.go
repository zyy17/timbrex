package provideropts

import (
	"strings"
	"testing"
)

func TestQwenSeed(t *testing.T) {
	for _, seed := range []int64{0, 1, 65535} {
		if err := (Qwen{Seed: &seed}).Validate(); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
	}
	for _, seed := range []int64{-1, 65536, 1<<63 - 1} {
		if err := (Qwen{Seed: &seed}).Validate(); err == nil || !strings.Contains(err.Error(), "seed") {
			t.Fatalf("seed %d: %v", seed, err)
		}
	}
}

func TestQwenHotFixLocalPolicy(t *testing.T) {
	var nilPairs []QwenTextPair
	empty := []QwenTextPair{}
	pairs := []QwenTextPair{{Text: "天气", Replacement: "tian1 qi4"}, {Text: "天气", Replacement: ""}, {Text: " \x00\t", Replacement: "\n\r\x00"}}
	for _, value := range []Qwen{{}, {HotFix: &QwenHotFix{}}, {HotFix: &QwenHotFix{Pronunciation: &empty, Replace: &nilPairs}}, {HotFix: &QwenHotFix{Pronunciation: &pairs, Replace: &pairs}}} {
		if err := value.Validate(); err != nil {
			t.Fatalf("valid options: %v", err)
		}
	}
	bad := []QwenTextPair{{Text: "valid", Replacement: "value"}, {Text: "", Replacement: "value"}}
	for _, tc := range []struct {
		fix  QwenHotFix
		want string
	}{
		{QwenHotFix{Pronunciation: &bad}, "hot_fix.pronunciation[1].text"},
		{QwenHotFix{Replace: &bad}, "hot_fix.replace[1].text"},
	} {
		if err := (Qwen{HotFix: &tc.fix}).Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("invalid pair: %v; want %s", err, tc.want)
		}
	}
}

func TestQwenHotFixRejectsInvalidUTF8WithoutLeakingValues(t *testing.T) {
	for _, tc := range []struct {
		pair QwenTextPair
		key  string
	}{
		{QwenTextPair{Text: "PRIVATE_SOURCE_\xff", Replacement: "PRIVATE_VALUE"}, "text"},
		{QwenTextPair{Text: "PRIVATE_SOURCE", Replacement: "PRIVATE_VALUE_\xff"}, "replacement"},
	} {
		pairs := []QwenTextPair{tc.pair}
		for _, field := range []struct {
			fix  QwenHotFix
			name string
		}{
			{QwenHotFix{Pronunciation: &pairs}, "pronunciation"},
			{QwenHotFix{Replace: &pairs}, "replace"},
		} {
			err := (Qwen{HotFix: &field.fix}).Validate()
			want := "hot_fix." + field.name + "[0]." + tc.key + " must be valid UTF-8"
			if err == nil || err.Error() != want || strings.Contains(err.Error(), "PRIVATE_") {
				t.Fatalf("error = %v; want %s", err, want)
			}
		}
	}
}
