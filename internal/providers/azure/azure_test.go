package azure

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func TestPitchRespectsAzureFrequencyRange(t *testing.T) {
	for _, tc := range []struct {
		pitch string
		valid bool
	}{{"-12st", true}, {"+7st", true}, {"+7.0195st", true}, {"+7.02st", false}, {"+12st", false}, {"-12.01st", false}} {
		_, err := build(providers.Segment{Voice: "en-US-GuyNeural", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: []ir.Node{{Name: "pitch", Args: []ir.Arg{{Raw: tc.pitch, Kind: "semitone"}}}}})
		if (err == nil) != tc.valid {
			t.Errorf("pitch %s: valid=%v, err=%v", tc.pitch, tc.valid, err)
		}
	}
}

func TestSSMLEscapesTextAndAttributes(t *testing.T) {
	voice := `en-US-"<&Neural`
	style := `calm" malicious="yes`
	text := `Say <break/> & "hello".`
	seg := providers.Segment{Voice: voice, Lang: "en-US", Nodes: []ir.Node{{Kind: ir.Text, Text: text}}, Controls: []ir.Node{{Name: "style", Args: []ir.Arg{{Raw: style}}}}}
	ssml, err := build(seg)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Voice struct {
			Name    string `xml:"name,attr"`
			Express struct {
				Style    string `xml:"style,attr"`
				Text     string `xml:",chardata"`
				Children []any  `xml:",any"`
			} `xml:"express-as"`
		} `xml:"voice"`
	}
	if err := xml.Unmarshal([]byte(ssml), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Voice.Name != voice || doc.Voice.Express.Style != style || doc.Voice.Express.Text != text || len(doc.Voice.Express.Children) != 0 {
		t.Fatalf("XML changed content: %s", ssml)
	}
}

func TestRejectConflictingStylesAndUnsupportedTags(t *testing.T) {
	for _, controls := range [][]ir.Node{{{Name: "happy"}, {Name: "whisper"}}, {{Name: "role", Args: []ir.Arg{{Raw: "Boy"}}}}, {{Name: "act", Args: []ir.Arg{{Raw: "warmly"}}}}} {
		_, err := build(providers.Segment{Voice: "en-US-JennyNeural", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: controls})
		if err == nil {
			t.Fatalf("accepted %+v", controls)
		}
	}
	ssml, err := build(providers.Segment{Voice: "en-US-JennyNeural", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}, Controls: []ir.Node{{Name: "pitch", Args: []ir.Arg{{Raw: "+2st", Kind: "semitone"}}}, {Name: "volume", Args: []ir.Arg{{Raw: "0dB", Kind: "decibel"}}}}})
	if err != nil || !strings.Contains(ssml, `pitch="+2st"`) || !strings.Contains(ssml, `volume="+0.0000%"`) {
		t.Fatalf("%s: %v", ssml, err)
	}
}

func TestLanguageOverrideInsideVoice(t *testing.T) {
	for _, tc := range []struct{ voice, lang, root, target string }{
		{"en-US-AndrewMultilingualNeural", "en-GB", "en-US", "en-GB"},
		{"en-US-Ava:DragonHDLatestNeural", "de-DE", "en-US", "de-DE"},
		{"de-DE-KatjaNeural", "", "de-DE", ""},
		{"en-US-GuyNeural", "en-US", "en-US", ""},
		{"custom-voice", "fr-FR", "fr-FR", ""},
		{"my-custom-voice", "en-US", "en-US", ""},
		{"fake-custom-voice-long-id", "fr-FR", "fr-FR", ""},
	} {
		t.Run(tc.voice+tc.lang, func(t *testing.T) {
			ssml, err := build(providers.Segment{Voice: tc.voice, Lang: tc.lang, Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello & goodbye"}}, Controls: []ir.Node{{Name: "speed", Args: []ir.Arg{{Raw: "1.2", Kind: "number"}}}}})
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Lang  string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
				Voice struct {
					Lang struct {
						Locale  string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
						Prosody struct {
							Rate string `xml:"rate,attr"`
							Text string `xml:",chardata"`
						} `xml:"prosody"`
					} `xml:"lang"`
				} `xml:"voice"`
			}
			if err := xml.Unmarshal([]byte(ssml), &doc); err != nil {
				t.Fatal(err)
			}
			if doc.Lang != tc.root || doc.Voice.Lang.Locale != tc.target {
				t.Fatalf("language mapping = %s", ssml)
			}
			if tc.target != "" && (doc.Voice.Lang.Prosody.Rate != "1.2" || doc.Voice.Lang.Prosody.Text != "Hello & goodbye") {
				t.Fatalf("lost controlled speech: %s", ssml)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestResponseAtDurationLimitPreservesOutput(t *testing.T) {
	const cutoff = 10 * 60 * 24000 * 2
	for _, size := range []int{cutoff - 2, cutoff, cutoff + 2} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			calls := 0
			p := provider{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(make([]byte, size)))}, nil
			})}}
			out := filepath.Join(t.TempDir(), "out.wav")
			if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			doc := &ir.Document{Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}}}
			_, err := p.Synthesize(context.Background(), doc, providers.Options{Out: out, Voice: "en-US-GuyNeural", Endpoint: "https://example.invalid"})
			if size < cutoff {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "10-minute") {
				t.Errorf("error=%v", err)
			}
			data, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if size >= cutoff && string(data) != "KEEP" {
				t.Errorf("replaced output with %d bytes", len(data))
			}
			if size < cutoff && len(data) != 44+size {
				t.Errorf("WAV size=%d", len(data))
			}
			if calls != 1 {
				t.Errorf("requests=%d", calls)
			}
			partials, err := filepath.Glob(filepath.Join(filepath.Dir(out), ".timbrex-*.partial"))
			if err != nil || len(partials) != 0 {
				t.Errorf("temporary files=%v error=%v", partials, err)
			}
		})
	}
}
