package options

import (
	"strings"
	"testing"
)

func TestMiniMaxNormalization(t *testing.T) {
	if err := (MiniMax{}).Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []MiniMaxTextNormalization{MiniMaxNormalizationBasic, MiniMaxNormalizationQuality} {
		if err := (MiniMax{TextNormalizationMode: &mode}).Validate(); err != nil {
			t.Fatalf("mode %q: %v", mode, err)
		}
	}
	for _, mode := range []MiniMaxTextNormalization{"", "off", "auto", "Quality", " basic", "quality ", "false"} {
		if err := (MiniMax{TextNormalizationMode: &mode}).Validate(); err == nil || !strings.Contains(err.Error(), "text_normalization_mode") {
			t.Fatalf("mode %q: %v", mode, err)
		}
	}
}
