package elevenlabs

import (
	"encoding/json"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

func TestSynthesisRequestFields(t *testing.T) {
	seed := int64(0)
	norm := options.ElevenNormalizationOff
	languageNorm := false
	value := options.ElevenLabs{Seed: &seed, ApplyTextNormalization: &norm, ApplyLanguageTextNormalization: &languageNorm}
	seg := providers.Segment{Lang: "en-US", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello 123"}}, Controls: []ir.Node{{Name: "speed", Args: []ir.Arg{{Kind: "number", Raw: "1.1"}}}}}
	r, err := buildWithSynthesis(seg, "eleven_v3", value)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"text":"Hello 123","model_id":"eleven_v3","language_code":"en","voice_settings":{"speed":1.1},"seed":0,"apply_text_normalization":"off","apply_language_text_normalization":false}`
	if string(body) != want {
		t.Fatalf("body = %s; want %s", body, want)
	}
}

func TestSynthesisOmissionPreservesCompleteRequest(t *testing.T) {
	for _, model := range []string{"eleven_v3", "eleven_multilingual_v2", "eleven_flash_v2"} {
		for _, seg := range []providers.Segment{
			{Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}},
			{Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: []ir.Node{{Name: "speed", Args: []ir.Arg{{Kind: "number", Raw: "1.1"}}}}},
		} {
			old, err := build(seg, model)
			if err != nil {
				t.Fatal(err)
			}
			got, err := buildWithSynthesis(seg, model, options.ElevenLabs{})
			if err != nil {
				t.Fatal(err)
			}
			before, err := json.Marshal(old)
			if err != nil {
				t.Fatal(err)
			}
			after, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatalf("omission changed body: %s != %s", before, after)
			}
		}
	}
}

func TestSynthesisRequestSnapshotsAndRange(t *testing.T) {
	seed := int64(4294967295)
	norm := options.ElevenNormalizationOn
	languageNorm := true
	value := options.ElevenLabs{Seed: &seed, ApplyTextNormalization: &norm, ApplyLanguageTextNormalization: &languageNorm}
	seg := providers.Segment{Nodes: []ir.Node{{Kind: ir.Text, Text: "今日は10月24日です。"}}}
	for _, model := range []string{"eleven_v3", "eleven_multilingual_v2", "eleven_flash_v2_5", "eleven_turbo_v2_5"} {
		r, err := buildWithSynthesis(seg, model, value)
		if err != nil {
			t.Fatal(err)
		}
		if r.Language != "" || *r.Seed != 4294967295 || *r.ApplyTextNormalization != "on" || !*r.ApplyLanguageTextNormalization {
			t.Fatalf("wrong options for %s: %+v", model, r)
		}
	}
	r, err := buildWithSynthesis(seg, "eleven_v3", value)
	if err != nil {
		t.Fatal(err)
	}
	seed, norm, languageNorm = 1, options.ElevenNormalizationOff, false
	if *r.Seed != 4294967295 || *r.ApplyTextNormalization != "on" || !*r.ApplyLanguageTextNormalization {
		t.Fatal("request shares mutable caller options")
	}
	*r.Seed, *r.ApplyTextNormalization, *r.ApplyLanguageTextNormalization = 2, options.ElevenNormalizationAuto, true
	if seed != 1 || norm != "off" || languageNorm {
		t.Fatal("builder mutated caller settings")
	}
	for _, badSeed := range []int64{-1, 4294967296} {
		if _, err := synthesisOptions(options.ElevenLabs{Seed: &badSeed}); err == nil {
			t.Fatalf("narrowed invalid seed %d", badSeed)
		}
	}
}
