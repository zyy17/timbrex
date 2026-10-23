package elevenlabs

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func TestModelSpecificTags(t *testing.T) {
	seg := providers.Segment{Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}, {Kind: ir.Event, Name: "laugh"}}, Controls: []ir.Node{{Name: "whisper"}}}
	r, err := build(seg, "eleven_v3")
	if err != nil || r.Text != "[whispers]Hello[laughs]" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := build(seg, "eleven_multilingual_v2"); err == nil {
		t.Fatal("accepted unsupported nonverbal")
	}
	seg.Nodes = seg.Nodes[:1]
	if _, err := build(seg, "eleven_multilingual_v2"); err == nil {
		t.Fatal("accepted unsupported direction")
	}
	seg.Controls = nil
	if _, err := build(seg, "eleven_multilingual_v2"); err != nil {
		t.Fatal(err)
	}
	seg.Lang = "en"
	if _, err := build(seg, "eleven_multilingual_v2"); err == nil {
		t.Fatal("silently ignored language hint")
	}
}

func TestLanguageHints(t *testing.T) {
	for _, tc := range []struct {
		model, lang, want string
		wantErr           bool
	}{
		{"eleven_flash_v2", "", "", false},
		{"eleven_flash_v2", "en", "en", false},
		{"eleven_flash_v2", "en-US", "en", false},
		{"eleven_turbo_v2", "EN-gb", "en", false},
		{"eleven_flash_v2", "fr-FR", "", true},
		{"eleven_turbo_v2", "fr-FR", "", true},
		{"eleven_multilingual_v2", "en-US", "", true},
		{"eleven_v3", "-en", "", true},
		{"eleven_v4", "fr-FR", "fr", false},
		{"eleven_v3", "fil", "fil", false},
		{"eleven_v3", "yue", "yue", false},
	} {
		t.Run(tc.model+"/"+tc.lang, func(t *testing.T) {
			seg := providers.Segment{Lang: tc.lang, Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}
			r, err := build(seg, tc.model)
			if (err != nil) != tc.wantErr || !tc.wantErr && r.Language != tc.want {
				t.Fatalf("language=%q, error=%v", r.Language, err)
			}
		})
	}
}

func TestRejectNativeMarkupInLiteralTextAndInstructions(t *testing.T) {
	for _, seg := range []providers.Segment{
		{Nodes: []ir.Node{{Kind: ir.Text, Text: "[laughs]"}}},
		{Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: []ir.Node{{Name: "act", Args: []ir.Arg{{Raw: "[laughs]"}}}}},
	} {
		if _, err := build(seg, "eleven_v3"); err == nil {
			t.Fatalf("accepted %+v", seg)
		}
	}
}

func TestV4RejectsUnsupportedSpeedControls(t *testing.T) {
	for _, model := range []string{"eleven_v4", "eleven_v4_turbo"} {
		for _, control := range []ir.Node{
			{Name: "speed", Args: []ir.Arg{{Raw: "1.1", Kind: "number"}}},
			{Name: "speed", Args: []ir.Arg{{Raw: "90%", Kind: "percent"}}},
			{Name: "fast"},
			{Name: "slow"},
		} {
			seg := providers.Segment{Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: []ir.Node{control}}
			if _, err := build(seg, model); err == nil {
				t.Errorf("%s accepted unsupported [%s]", model, control.Name)
			}
			if _, err := build(seg, "eleven_multilingual_v2"); err != nil {
				t.Errorf("multilingual v2 rejected supported [%s]: %v", control.Name, err)
			}
		}
	}
}

func TestRejectLiteralSSMLForOlderModels(t *testing.T) {
	for _, model := range []string{"eleven_multilingual_v2", "eleven_flash_v2", "eleven_turbo_v2"} {
		for _, text := range []string{
			`Hello <break time="1.5s" /> again`,
			"Hello <break\ntime='1s'/> again",
			`Hello <break/> again`,
			`<phoneme alphabet="ipa" ph="təˈmeɪtoʊ">tomato</phoneme>`,
		} {
			if model == "eleven_multilingual_v2" && strings.HasPrefix(text, "<phoneme") {
				continue
			}
			seg := providers.Segment{Nodes: []ir.Node{{Kind: ir.Text, Text: text}}}
			if _, err := build(seg, model); err == nil {
				t.Errorf("%s accepted native markup in literal text: %s", model, text)
			}
		}
		for _, text := range []string{"1 < 2 and 3 > 2", "<breakfast> is not a break tag", "<phonemes> is not a phoneme tag"} {
			if _, err := build(providers.Segment{Nodes: []ir.Node{{Kind: ir.Text, Text: text}}}, model); err != nil {
				t.Errorf("%s rejected plain text %q: %v", model, text, err)
			}
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestInvalidLaterSegmentFailsBeforeAnyRequest(t *testing.T) {
	for _, tc := range []struct {
		model string
		nodes []ir.Node
	}{
		{"eleven_v4", []ir.Node{{Kind: ir.State, Name: "fast"}, {Kind: ir.Text, Text: "Later"}}},
		{"eleven_multilingual_v2", []ir.Node{{Kind: ir.Text, Text: `Later <break time="1s" />`}}},
		{"eleven_flash_v2", []ir.Node{{Kind: ir.SpanStart, Name: "lang", Args: []ir.Arg{{Raw: "fr-FR", Kind: "word"}}}, {Kind: ir.Text, Text: "Bonjour"}, {Kind: ir.SpanEnd, Name: "lang"}}},
		{"eleven_turbo_v2", []ir.Node{{Kind: ir.SpanStart, Name: "lang", Args: []ir.Arg{{Raw: "fr-FR", Kind: "word"}}}, {Kind: ir.Text, Text: "Bonjour"}, {Kind: ir.SpanEnd, Name: "lang"}}},
		{"eleven_v3", []ir.Node{{Kind: ir.SpanStart, Name: "lang", Args: []ir.Arg{{Raw: "-en", Kind: "quoted"}}}, {Kind: ir.Text, Text: "Later"}, {Kind: ir.SpanEnd, Name: "lang"}}},
	} {
		t.Run(tc.model, func(t *testing.T) {
			calls := 0
			p := provider{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return nil, fmt.Errorf("unexpected request")
			})}}
			out := filepath.Join(t.TempDir(), "out.wav")
			if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			doc := &ir.Document{Utterances: []ir.Utterance{
				{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "First"}}},
				{Speaker: "narrator", Nodes: tc.nodes},
			}}
			_, err := p.Synthesize(context.Background(), doc, providers.Options{Out: out, Voice: "test-voice", Model: tc.model, Endpoint: "https://example.invalid"})
			if err == nil || calls != 0 {
				t.Fatalf("error=%v calls=%d", err, calls)
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != "KEEP" {
				t.Fatalf("output=%q error=%v", data, err)
			}
		})
	}
}
