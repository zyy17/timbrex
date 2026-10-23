package providers_test

import (
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func TestSpeedUsesParsedArgumentTypes(t *testing.T) {
	for _, arg := range []string{"1.2", "120%"} {
		segs, err := providers.Plan(document(t, "[speed "+arg+"|Hi]"), "openai", "coral")
		if err != nil {
			t.Fatal(err)
		}
		speed, err := providers.Speed(segs[0].Controls[0], 0.25, 4)
		if err != nil || speed != 1.2 {
			t.Fatalf("%q: speed=%g, error=%v", arg, speed, err)
		}
	}
}

func TestDirection(t *testing.T) {
	for _, tt := range []struct {
		name    string
		node    ir.Node
		want    string
		wantErr string
	}{
		{"free acting", ir.Node{Name: "act", Note: "quietly, then brighten"}, "quietly, then brighten", ""},
		{"explicit acting", ir.Node{Name: "act", Args: []ir.Arg{{Raw: "with warmth", Kind: "quoted"}}}, "with warmth", ""},
		{"emotion", ir.Node{Name: "happy"}, "happy", ""},
		{"minimum intensity", ir.Node{Name: "sad", Args: []ir.Arg{{Raw: "0", Kind: "number"}}}, "sad (intensity 0 of 1)", ""},
		{"fractional intensity", ir.Node{Name: "happy", Args: []ir.Arg{{Raw: "0.25", Kind: "number"}}}, "happy (intensity 0.25 of 1)", ""},
		{"maximum intensity", ir.Node{Name: "angry", Args: []ir.Arg{{Raw: "1", Kind: "number"}}}, "angry (intensity 1 of 1)", ""},
		{"laugh span", ir.Node{Name: "laugh"}, "speak while laughing", ""},
		{"cry span with note", ir.Node{Name: "cry", Note: "softly"}, "speak while crying softly", ""},
		{"sigh span trims note", ir.Node{Name: "sigh", Note: "gently  "}, "speak while sighing gently", ""},
		{"whisper", ir.Node{Name: "whisper"}, "whisper", ""},
		{"shout", ir.Node{Name: "shout"}, "shout", ""},
		{"emphasis", ir.Node{Name: "emph"}, "emphasize the spoken words", ""},
		{"style", ir.Node{Name: "style", Args: []ir.Arg{{Raw: "newscast", Kind: "word"}}}, "style: newscast", ""},
		{"role", ir.Node{Name: "role", Args: []ir.Arg{{Raw: "narrator", Kind: "word"}}}, "role: narrator", ""},
		{"accent", ir.Node{Name: "accent", Args: []ir.Arg{{Raw: "British", Kind: "word"}}}, "accent: British", ""},
		{"dialect", ir.Node{Name: "dialect", Args: []ir.Arg{{Raw: "Cantonese", Kind: "word"}}}, "dialect: Cantonese", ""},
		{"custom emotion", ir.Node{Name: "emotion", Args: []ir.Arg{{Raw: "wistful", Kind: "word"}}}, "emotion: wistful", ""},
		{"missing acting", ir.Node{Name: "act"}, "", "[act] needs one positional value"},
		{"acting note and value", ir.Node{Name: "act", Note: "softly", Args: []ir.Arg{{Raw: "warmly", Kind: "word"}}}, "", "[act] needs one positional value"},
		{"named acting value", ir.Node{Name: "act", Args: []ir.Arg{{Key: "style", Raw: "softly", Kind: "word"}}}, "", "[act] needs one positional value"},
		{"empty acting value", ir.Node{Name: "act", Args: []ir.Arg{{Kind: "quoted"}}}, "", "[act] needs one positional value"},
		{"emotion note", ir.Node{Name: "happy", Note: "softly"}, "", "[happy] does not support arguments here"},
		{"quoted intensity", ir.Node{Name: "happy", Args: []ir.Arg{{Raw: "0.5", Kind: "quoted"}}}, "", "[happy] needs an unquoted number value"},
		{"negative intensity", ir.Node{Name: "happy", Args: []ir.Arg{{Raw: "-0.1", Kind: "number"}}}, "", "[happy] needs a value from 0 to 1"},
		{"excessive intensity", ir.Node{Name: "happy", Args: []ir.Arg{{Raw: "1.1", Kind: "number"}}}, "", "[happy] needs a value from 0 to 1"},
		{"nonfinite intensity", ir.Node{Name: "happy", Args: []ir.Arg{{Raw: "NaN", Kind: "number"}}}, "", "[happy] needs a value from 0 to 1"},
		{"multiple intensity values", ir.Node{Name: "happy", Args: []ir.Arg{{Raw: "0.1", Kind: "number"}, {Raw: "0.2", Kind: "number"}}}, "", "[happy] needs one positional value"},
		{"numeric laugh span", ir.Node{Name: "laugh", Args: []ir.Arg{{Raw: "2", Kind: "number"}}}, "", "[laugh] span does not support numeric arguments"},
		{"whisper argument", ir.Node{Name: "whisper", Args: []ir.Arg{{Raw: "softly", Kind: "word"}}}, "", "[whisper] does not support arguments here"},
		{"shout note", ir.Node{Name: "shout", Note: "loudly"}, "", "[shout] does not support arguments here"},
		{"emphasis argument", ir.Node{Name: "emph", Args: []ir.Arg{{Raw: "2", Kind: "number"}}}, "", "[emph] does not support arguments here"},
		{"missing style", ir.Node{Name: "style"}, "", "[style] needs one positional value"},
		{"unknown tag", ir.Node{Name: "missing"}, "", "unsupported tag [missing]"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := providers.Direction(tt.node)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return // Direction's result is unused when it returns an error.
			}
			if err != nil || got != tt.want {
				t.Fatalf("direction = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}
