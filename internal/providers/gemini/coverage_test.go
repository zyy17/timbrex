package gemini

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/vocab"
)

var testModels = []string{"gemini-3.8-flash-tts", "gemini-3.8-flash-lite-tts"}

// Independent fixtures from the GenerateContent TTS vocal-burst table, checked
// 2026-10-24. Keep DSL names distinct from the vendor's exact tag spellings.
var vocalEvents = []struct{ name, tag string }{
	{"laugh", "<laugh>"}, {"sigh", "<sigh>"}, {"cough", "<cough>"},
	{"breath", "<breath>"}, {"gasp", "<gasp>"}, {"exhale", "<exhales>"},
	{"chuckle", "<chuckle>"}, {"giggle", "<giggle>"}, {"cackle", "<cackle>"},
	{"snort", "<snort>"}, {"cry", "<cry>"}, {"sob", "<sob>"},
	{"whimper", "<whimper>"}, {"pant", "<pant>"}, {"groan", "<groan>"},
	{"grunt", "<grunt>"}, {"moan", "<moan>"}, {"sneeze", "<sneeze>"},
	{"yawn", "<yawn>"}, {"hiss", "<hiss>"}, {"clear-throat", "<throat-clearing>"},
}

func TestBuildDocumentedVocalEvents(t *testing.T) {
	for _, model := range testModels {
		for _, event := range vocalEvents {
			t.Run(model+"/"+event.name, func(t *testing.T) {
				spec, ok := vocab.Lookup(event.name)
				if !ok || spec.Kind != vocab.Event {
					t.Fatal("fixture must refer to a registered DSL event")
				}
				seg := providers.Segment{Voice: "Kore", Nodes: []ir.Node{
					{Text: "Before \"&\\\n"}, {Kind: ir.Event, Name: event.name}, {Text: " 后面。"},
				}}
				r, err := build(seg, model)
				if err != nil {
					t.Fatal(err)
				}
				part := r.Contents[0].Parts[0]
				if part.Text != "Before \"&\\\n"+event.tag+" 后面。" || part.SpeechMetadata.Style != "" {
					t.Fatalf("incorrect vocal event: %+v", part)
				}
				for _, invalid := range []ir.Node{
					{Note: "quietly"},
					{Args: []ir.Arg{{Raw: "0.5", Kind: "number"}}},
					{Args: []ir.Arg{{Raw: "quietly", Kind: "quoted"}}},
					{Args: []ir.Arg{{Key: "intensity", Raw: "0.5", Kind: "number"}}},
				} {
					invalid.Kind, invalid.Name = ir.Event, event.name
					seg.Nodes[1] = invalid
					if _, err := build(seg, model); err == nil || !strings.Contains(err.Error(), "arguments") {
						t.Fatalf("accepted event arguments: %+v, err=%v", invalid, err)
					}
				}
			})
		}
	}
}

func TestBuildModelsAndLanguages(t *testing.T) {
	// SpeechConfig.languageCode's exact API allowlist, intersected with each
	// model's documented languages. Thai is absent from Flash-Lite's table.
	codes := strings.Fields("de-DE en-AU en-GB en-IN en-US es-US fr-FR hi-IN pt-BR ar-XA es-ES fr-CA id-ID it-IT ja-JP tr-TR vi-VN bn-IN gu-IN kn-IN ml-IN mr-IN ta-IN te-IN nl-NL ko-KR cmn-CN pl-PL ru-RU th-TH")
	for _, model := range testModels {
		t.Run(model, func(t *testing.T) {
			base := providers.Segment{Voice: "Kore", Nodes: []ir.Node{{Text: "Hello"}}}
			for _, voice := range voices {
				seg := base
				seg.Voice = voice
				if _, err := build(seg, model); err != nil {
					t.Fatalf("voice %s: %v", voice, err)
				}
			}
			for _, code := range codes {
				seg := base
				seg.Lang = code
				r, err := build(seg, model)
				if code == "th-TH" && model == "gemini-3.8-flash-lite-tts" {
					if err == nil || !strings.Contains(err.Error(), "language") {
						t.Fatalf("Lite accepted Thai: %v", err)
					}
					continue
				}
				if err != nil || r.GenerationConfig.SpeechConfig.LanguageCode != code {
					t.Fatalf("language %s: %+v, %v", code, r, err)
				}
			}
			for _, code := range []string{"en", "zh-CN", "sv-SE", "EN-US", "en-US ", "auto"} {
				seg := base
				seg.Lang = code
				if _, err := build(seg, model); err == nil {
					t.Fatalf("accepted undocumented languageCode %q", code)
				}
			}
			base.Nodes = []ir.Node{{Text: "Καλημέρα"}}
			r, err := build(base, model)
			if err != nil {
				t.Fatal(err)
			}
			body, err := json.Marshal(r)
			if err != nil || strings.Contains(string(body), "languageCode") {
				t.Fatalf("autodetection must omit languageCode: %s, %v", body, err)
			}
		})
	}
	for _, model := range []string{"", "gemini-2.5-flash-preview-tts", "gemini-2.5-pro-preview-tts", "gemini-3.1-flash-tts-preview", "gemini-3.8-flash", "gemini-3.8-flash-tts-preview", "gemini-3.8-flash-lite-tts-preview", "../gemini-3.8-flash-tts"} {
		if _, err := build(providers.Segment{Voice: "Kore", Nodes: []ir.Node{{Text: "Hello"}}}, model); err == nil {
			t.Fatalf("accepted unsupported model %q", model)
		}
	}
}

func TestBuildCommonRuneLimit(t *testing.T) {
	for _, model := range testModels {
		for _, tc := range []struct {
			name, text, style, event string
			invalid                  bool
		}{
			{name: "ASCII at bound", text: strings.Repeat("a", 4096)},
			{name: "ASCII over bound", text: strings.Repeat("a", 4097), invalid: true},
			{name: "multibyte at bound", text: strings.Repeat("中", 4096)},
			{name: "multibyte over bound", text: strings.Repeat("中", 4097), invalid: true},
			{name: "combining at bound", text: strings.Repeat("e\u0301", 2048)},
			{name: "combining over bound", text: strings.Repeat("e\u0301", 2048) + "a", invalid: true},
			{name: "style at bound", text: "a", style: strings.Repeat("中", 4095)},
			{name: "style over bound", text: "a", style: strings.Repeat("中", 4096), invalid: true},
			{name: "event at bound", text: strings.Repeat("a", 4087), event: "exhale"},
			{name: "event over bound", text: strings.Repeat("a", 4088), event: "exhale", invalid: true},
		} {
			t.Run(model+"/"+tc.name, func(t *testing.T) {
				seg := providers.Segment{Voice: "Kore", Nodes: []ir.Node{{Text: tc.text}}}
				if tc.style != "" {
					seg.Controls = []ir.Node{{Name: "act", Note: tc.style}}
				}
				if tc.event != "" {
					seg.Nodes = append(seg.Nodes, ir.Node{Kind: ir.Event, Name: tc.event})
				}
				_, err := build(seg, model)
				if (err != nil) != tc.invalid || (err != nil && !strings.Contains(err.Error(), "4096")) {
					t.Fatalf("invalid=%v, err=%v", tc.invalid, err)
				}
			})
		}
	}
}

func TestBuildPreservesStrictRestrictions(t *testing.T) {
	for _, model := range testModels {
		t.Run(model, func(t *testing.T) {
			for _, name := range []string{"inhale", "quick-breath", "huff", "scoff", "sniffle", "sniff", "wail", "hiccup", "burp", "snore", "swallow", "gulp", "lip-smack", "tongue-click", "kiss", "shush", "whistle", "hum", "hmm", "uh", "um", "emm", "sfx", "scream"} {
				_, err := build(providers.Segment{Voice: "Kore", Nodes: []ir.Node{{Kind: ir.Event, Name: name}}}, model)
				if err == nil || !strings.Contains(err.Error(), "unsupported event") {
					t.Fatalf("accepted unsupported event %s: %v", name, err)
				}
			}
			for _, text := range []string{"<laugh>", "<", ">", `\<chuckle\>`, "before <throat-clearing> after"} {
				_, err := build(providers.Segment{Voice: "Kore", Nodes: []ir.Node{{Text: text}}}, model)
				if err == nil || !strings.Contains(err.Error(), "literal") {
					t.Fatalf("accepted reserved markup %q: %v", text, err)
				}
			}
			for _, name := range []string{"speed", "pitch", "volume"} {
				_, err := build(providers.Segment{Voice: "Kore", Nodes: []ir.Node{{Text: "Hello"}}, Controls: []ir.Node{{Name: name, Args: []ir.Arg{{Raw: "1", Kind: "number"}}}}}, model)
				if err == nil || !strings.Contains(err.Error(), "no numeric") {
					t.Fatalf("accepted numeric control %s: %v", name, err)
				}
			}
			for _, voice := range []string{"kore", "voice_custom", "voicekey_custom", "unknown", "Kore ", ""} {
				if _, err := build(providers.Segment{Voice: voice, Nodes: []ir.Node{{Text: "Hello"}}}, model); err == nil {
					t.Fatalf("accepted unsupported voice %q", voice)
				}
			}
		})
	}
}

func TestSynthesizeRejectsUnsupportedModelsBeforeSilence(t *testing.T) {
	for _, model := range []string{"", "gemini-2.5-flash-preview-tts", "gemini-3.1-flash-tts-preview", "gemini-3.8-flash-lite-tts-preview", "unknown"} {
		t.Run(model, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out.wav")
			if err := os.WriteFile(out, []byte("KEEP"), 0600); err != nil {
				t.Fatal(err)
			}
			doc := &ir.Document{Utterances: []ir.Utterance{{Nodes: []ir.Node{{Kind: ir.Event, Name: "pause", Args: []ir.Arg{{Raw: "1ms", Kind: "duration"}}}}}}}
			_, err := (provider{}).Synthesize(context.Background(), doc, providers.Options{Model: model, Voice: "Kore", Out: out})
			if err == nil || !strings.Contains(err.Error(), "unsupported Gemini TTS model") {
				t.Fatalf("accepted unsupported model for silence-only script: %v", err)
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != "KEEP" {
				t.Fatalf("output changed: %q, %v", data, err)
			}
		})
	}
}
