package azure

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

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
