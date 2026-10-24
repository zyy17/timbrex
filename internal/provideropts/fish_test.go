package provideropts

import (
	"math"
	"strings"
	"testing"
)

func TestFishFiniteRanges(t *testing.T) {
	for _, field := range []struct {
		name string
		set  func(*float64) Fish
	}{
		{"temperature", func(v *float64) Fish { return Fish{Temperature: v} }},
		{"top_p", func(v *float64) Fish { return Fish{TopP: v} }},
		{"early_stop_threshold", func(v *float64) Fish { return Fish{EarlyStopThreshold: v} }},
	} {
		t.Run(field.name, func(t *testing.T) {
			for _, number := range []float64{math.Inf(-1), -math.MaxFloat64, -0.001, 0, 0.5, 1, 1.001, math.MaxFloat64, math.Inf(1), math.NaN()} {
				err := field.set(&number).Validate()
				valid := !math.IsNaN(number) && number >= 0 && number <= 1
				if (err == nil) != valid {
					t.Errorf("%v: %v", number, err)
				}
				if err != nil && !strings.Contains(err.Error(), field.name) {
					t.Errorf("error does not identify field: %v", err)
				}
			}
		})
	}
}

func TestFishChunkRanges(t *testing.T) {
	for _, number := range []int64{math.MinInt64, -1, 0, 99, 100, 101, 299, 300, 301, math.MaxInt64} {
		if err := (Fish{ChunkLength: &number}).Validate(); (err == nil) != (number >= 100 && number <= 300) {
			t.Errorf("chunk_length %d: %v", number, err)
		}
		if err := (Fish{MinChunkLength: &number}).Validate(); (err == nil) != (number >= 0 && number <= 100) {
			t.Errorf("min_chunk_length %d: %v", number, err)
		}
	}
}

func TestFishNormalizationLanguage(t *testing.T) {
	for _, tc := range []struct {
		language string
		valid    bool
	}{
		{"", true}, {"auto", true}, {"AUTO", true}, {"en", true}, {"es-MX", true}, {"zh_CN", true},
		{"x-1", true}, {strings.Repeat("a", 35), true}, {"unknown", true},
		{strings.Repeat("a", 36), false}, {"en US", false}, {"en\n", false}, {" en", false},
		{"中文", false}, {"en.US", false}, {"en/US", false}, {"en\x00US", false},
	} {
		if err := (Fish{NormalizationLanguage: &tc.language}).Validate(); (err == nil) != tc.valid {
			t.Errorf("language %q: %v", tc.language, err)
		}
	}
}

func TestFishLatencyAndPresence(t *testing.T) {
	for _, latency := range []FishLatency{FishLatencyLow, FishLatencyNormal, FishLatencyBalanced, "", "LOW", "low ", "auto"} {
		valid := latency == FishLatencyLow || latency == FishLatencyNormal || latency == FishLatencyBalanced
		if err := (Fish{Latency: &latency}).Validate(); (err == nil) != valid {
			t.Errorf("latency %q: %v", latency, err)
		}
	}
	if err := (Fish{}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Fish{Prosody: &FishProsody{}}).Validate(); err != nil {
		t.Fatal(err)
	}
	for _, value := range []bool{false, true} {
		if err := (Fish{Normalize: &value, ConditionOnPreviousChunks: &value, Prosody: &FishProsody{NormalizeLoudness: &value}}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFishValueRegistration(t *testing.T) {
	for _, id := range []string{"fish", "elevenlabs", "openai", "stub"} {
		if err := Validate(id, Fish{}); (err == nil) != (id == "fish") {
			t.Errorf("%s: %v", id, err)
		}
		for _, value := range []Value{(*Fish)(nil), &Fish{}, embeddedValue{Value: Fish{}}} {
			if err := Validate(id, value); err == nil {
				t.Errorf("%s accepted a pointer or embedded Fish variant", id)
			}
		}
	}
}
