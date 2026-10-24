package azure

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

// These assertions cover SSML structure, not whether a voice supports each control.
func TestSSMLControlOrder(t *testing.T) {
	controls := []ir.Node{
		{Name: "emph"},
		{Name: "speed", Args: []ir.Arg{{Raw: "1.2", Kind: "number"}}},
		{Name: "pitch", Args: []ir.Arg{{Raw: "+2st", Kind: "semitone"}}},
		{Name: "volume", Args: []ir.Arg{{Raw: "-6dB", Kind: "decibel"}}},
		{Name: "style", Args: []ir.Arg{{Raw: "cheerful"}}},
		{Name: "role", Args: []ir.Arg{{Raw: "YoungAdultMale"}}},
	}
	want := `<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="https://www.w3.org/2001/mstts" xml:lang="en-US"><voice name="en-US-GuyNeural"><lang xml:lang="de-DE"><mstts:express-as style="cheerful" role="YoungAdultMale"><emphasis level="strong"><prosody rate="1.2" pitch="+2st" volume="-49.8813%">Hello &amp; goodbye.</prosody></emphasis></mstts:express-as></lang></voice></speak>`
	var permutations int
	var visit func(int)
	visit = func(start int) {
		if start == len(controls) {
			permutations++
			ssml, err := build(providers.Segment{Voice: "en-US-GuyNeural", Lang: "de-DE", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello & goodbye."}}, Controls: controls})
			if err != nil {
				t.Fatal(err)
			}
			if ssml != want {
				t.Fatalf("controls = %+v\nSSML = %s\nwant = %s", controls, ssml, want)
			}
			assertSSMLHierarchy(t, ssml, []string{"speak", "voice", "lang", "express-as", "emphasis", "prosody"})
			return
		}
		for i := start; i < len(controls); i++ {
			controls[start], controls[i] = controls[i], controls[start]
			visit(start + 1)
			controls[start], controls[i] = controls[i], controls[start]
		}
	}
	visit(0)
	if permutations != 720 {
		t.Fatalf("tested %d permutations", permutations)
	}
}

func TestSSMLProsodyAliases(t *testing.T) {
	for _, tc := range []struct {
		name, arg, kind, rate string
	}{
		{"fast", "", "", "1.2"},
		{"slow", "", "", "0.8"},
		{"speed", "120%", "percent", "1.2"},
	} {
		t.Run(tc.name+tc.arg, func(t *testing.T) {
			n := ir.Node{Name: tc.name}
			if tc.arg != "" {
				n.Args = []ir.Arg{{Raw: tc.arg, Kind: tc.kind}}
			}
			for _, controls := range [][]ir.Node{{{Name: "emph"}, n}, {n, {Name: "emph"}}} {
				ssml, err := build(providers.Segment{Voice: "en-US-GuyNeural", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello."}}, Controls: controls})
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(ssml, fmt.Sprintf(`<emphasis level="strong"><prosody rate="%s">Hello.</prosody></emphasis>`, tc.rate)) {
					t.Fatalf("SSML = %s", ssml)
				}
			}
		})
	}
}

func TestSSMLVoiceLocaleFormats(t *testing.T) {
	for _, tc := range []struct{ voice, lang, root, target string }{
		{"en-US-GuyNeural", "", "en-US", ""},
		{"fil-PH-AngeloNeural", "", "fil-PH", ""},
		{"sr-Latn-RS-NicholasNeural", "", "sr-Latn-RS", ""},
		{"sr-Latn-RS-SophieNeural", "sr-RS", "sr-Latn-RS", "sr-RS"},
		{"SR-lATN-rS-NicholasNeural", "", "sr-Latn-RS", ""},
		{"en-us-Ava:DragonHDLatestNeural", "", "en-US", ""},
		{"en-us-Ava:DragonHDLatestNeural", "de-DE", "en-US", "de-DE"},
		{"en-us-Ava:DragonHDLatestNeural", "EN-us", "en-US", ""},
		// A numeric region is a supported locale shape, not a catalog assertion.
		{"es-419-ExampleNeural", "", "es-419", ""},
		// Preserve prefix inference without imposing rules on custom voice names.
		{"en-US-My Voice", "", "en-US", ""},
		{"en-US-", "", "en-US", ""},
		{"en-US--Voice", "", "en-US", ""},
	} {
		t.Run(tc.voice+"/"+tc.lang, func(t *testing.T) {
			ssml, err := build(providers.Segment{Voice: tc.voice, Lang: tc.lang, Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello."}}})
			if err != nil {
				t.Fatal(err)
			}
			assertSSMLLocale(t, ssml, tc.voice, tc.root, tc.target)
		})
	}
}

func TestSSMLUnknownVoiceLocaleFallback(t *testing.T) {
	for _, voice := range []string{
		"custom-voice", "my-custom-voice", "fake-custom-voice-long-id",
		"x-US-Voice", "engl-US-Voice", "en-USA-Voice", "en-12-Voice",
		"en-Latin-US-Voice", "en-Latn-Voice", "en_US_Voice", "en-US",
		"prefix-en-US-GuyNeural", " en-US-GuyNeural", "éñ-US-Voice", "en-KS-Voice",
	} {
		t.Run(voice, func(t *testing.T) {
			seg := providers.Segment{Voice: voice, Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello."}}}
			if _, err := build(seg); err == nil || !strings.Contains(err.Error(), "set lang for voice") {
				t.Fatalf("missing explicit lang: %v", err)
			}
			seg.Lang = "fr-FR"
			ssml, err := build(seg)
			if err != nil {
				t.Fatal(err)
			}
			assertSSMLLocale(t, ssml, voice, "fr-FR", "")
		})
	}
}

func TestSSMLCombinedControlsEscapeAllContent(t *testing.T) {
	voice := `en-US-"<&Neural`
	lang := `de-DE" injected="yes`
	style := `cheerful" injected="yes<&`
	role := `YoungAdultMale" injected="yes<&`
	text := `Say </prosody><audio src="evil"/> & "hello".`
	ssml, err := build(providers.Segment{Voice: voice, Lang: lang, Nodes: []ir.Node{{Kind: ir.Text, Text: text}}, Controls: []ir.Node{
		{Name: "emph"},
		{Name: "speed", Args: []ir.Arg{{Raw: "1.2", Kind: "number"}}},
		{Name: "pitch", Args: []ir.Arg{{Raw: "+2st", Kind: "semitone"}}},
		{Name: "volume", Args: []ir.Arg{{Raw: "0dB", Kind: "decibel"}}},
		{Name: "style", Args: []ir.Arg{{Raw: style}}},
		{Name: "role", Args: []ir.Arg{{Raw: role}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	assertSSMLLocale(t, ssml, voice, "en-US", lang)
	nodes := assertSSMLHierarchy(t, ssml, []string{"speak", "voice", "lang", "express-as", "emphasis", "prosody"})
	for _, node := range nodes {
		for _, attr := range node.Attrs {
			if attr.Name.Local == "injected" {
				t.Fatalf("injected XML attribute: %s", ssml)
			}
		}
	}
	if !reflect.DeepEqual(nodes[3].Attrs, []xml.Attr{{Name: xml.Name{Local: "style"}, Value: style}, {Name: xml.Name{Local: "role"}, Value: role}}) || nodes[5].Text != text {
		t.Fatalf("changed attribute or text: %s", ssml)
	}
}

func TestSSMLCustomVoiceEscapesRootLanguage(t *testing.T) {
	voice := `custom-"<&voice`
	lang := `en-US" injected="yes<&`
	ssml, err := build(providers.Segment{Voice: voice, Lang: lang, Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello."}}})
	if err != nil {
		t.Fatal(err)
	}
	assertSSMLLocale(t, ssml, voice, lang, "")
	nodes := assertSSMLHierarchy(t, ssml, []string{"speak", "voice"})
	if len(nodes[0].Attrs) != 4 || len(nodes[1].Attrs) != 1 || nodes[1].Text != "Hello." {
		t.Fatalf("changed attributes or text: %s", ssml)
	}
}

type ssmlElement struct {
	XMLName  xml.Name
	Attrs    []xml.Attr    `xml:",any,attr"`
	Text     string        `xml:",chardata"`
	Children []ssmlElement `xml:",any"`
}

func assertSSMLHierarchy(t *testing.T, ssml string, want []string) []ssmlElement {
	t.Helper()
	var root ssmlElement
	if err := xml.Unmarshal([]byte(ssml), &root); err != nil {
		t.Fatal(err)
	}
	var nodes []ssmlElement
	var names []string
	for node := root; ; node = node.Children[0] {
		nodes = append(nodes, node)
		names = append(names, node.XMLName.Local)
		if len(node.Children) == 0 {
			break
		}
		if len(node.Children) != 1 {
			t.Fatalf("unexpected children: %s", ssml)
		}
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("hierarchy = %v, want %v: %s", names, want, ssml)
	}
	return nodes
}

func assertSSMLLocale(t *testing.T, ssml, voice, root, target string) {
	t.Helper()
	var doc struct {
		Lang  string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
		Voice struct {
			Name string `xml:"name,attr"`
			Lang struct {
				Locale string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
			} `xml:"lang"`
		} `xml:"voice"`
	}
	if err := xml.Unmarshal([]byte(ssml), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Lang != root || doc.Voice.Name != voice || doc.Voice.Lang.Locale != target {
		t.Fatalf("language or voice mapping = %s", ssml)
	}
}
