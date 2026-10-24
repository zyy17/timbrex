// Package azure implements the Azure Speech REST API with SSML.
package azure

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func init() { providers.Register(provider{}) }

type provider struct{ client *http.Client }

func (provider) ID() string { return "azure" }

// Infer only Azure's language[-Script]-region prefix, not arbitrary BCP-47
// subtags. The remaining name and voice availability are not validated here.
var voiceLocale = regexp.MustCompile(`^([A-Za-z]{2,3})(?:-([A-Za-z]{4}))?-([A-Za-z]{2}|[0-9]{3})-`)

var styles = map[string]string{
	"happy": "cheerful", "sad": "sad", "angry": "angry", "fear": "fearful",
	"excited": "excited", "calm": "calm", "tender": "gentle", "serious": "serious",
	"whisper": "whispering", "shout": "shouting", "crying": "crying", "neutral": "neutral",
}

func (p provider) Synthesize(ctx context.Context, doc *ir.Document, opt providers.Options) (providers.Result, error) {
	if err := ctx.Err(); err != nil {
		return providers.Result{}, err
	}
	if err := providers.ValidateConcurrency(opt.Concurrency); err != nil {
		return providers.Result{}, err
	}
	if opt.Model != "" {
		return providers.Result{}, fmt.Errorf("azure selects its model through the voice name; omit model")
	}
	segments, requests, err := providers.Prepare(ctx, doc, p.ID(), opt.Voice, build)
	if err != nil {
		return providers.Result{}, err
	}
	// https://learn.microsoft.com/azure/ai-services/speech-service/rest-text-to-speech
	headers := map[string]string{
		"Accept":                    "audio/pcm, application/octet-stream",
		"Ocp-Apim-Subscription-Key": opt.APIKey,
		"X-Microsoft-OutputFormat":  "raw-24khz-16bit-mono-pcm",
	}
	return providers.Render(ctx, p.ID(), opt, segments, func(ctx context.Context, i int) ([]byte, error) {
		pcm, err := providers.Request(ctx, p.client, http.MethodPost, opt.Endpoint, "application/ssml+xml", headers, []byte(requests[i]), providers.ResponsePCM)
		if err != nil {
			return nil, err
		}
		if err := azurePCMComplete(len(pcm)); err != nil {
			return nil, err
		}
		return pcm, nil
	})
}

// Azure REST returns HTTP 200 and truncates audio at ten minutes.
// https://learn.microsoft.com/azure/ai-services/speech-service/rest-text-to-speech
const maxAzurePCMBytes = 10 * 60 * providers.SampleRate * 2

func azurePCMComplete(n int) error {
	if n >= maxAzurePCMBytes {
		return fmt.Errorf("audio reaches Azure's 10-minute limit and may be truncated; split the segment")
	}
	return nil
}

func build(seg providers.Segment) (string, error) {
	text, err := providers.Text(seg.Nodes, nil)
	if err != nil {
		return "", err
	}
	text = escape(text)
	style, role := "", ""
	rate, pitch, volume := "", "", ""
	emphasis := false
	for _, n := range seg.Controls {
		switch n.Name {
		case "speed", "fast", "slow":
			var speed float64
			speed, err = providers.Speed(n, 0.5, 2)
			rate = fmt.Sprintf(` rate="%g"`, speed)
		case "pitch":
			var semitones float64
			semitones, err = providers.Number(n, "st", -12, 12*math.Log2(1.5))
			pitch = fmt.Sprintf(` pitch="%+gst"`, semitones)
		case "volume":
			var db float64
			db, err = providers.Number(n, "dB", -60, 20)
			volume = fmt.Sprintf(` volume="%+.4f%%"`, (math.Pow(10, db/20)-1)*100)
		case "emph":
			err = providers.NoArgs(n)
			emphasis = true
		case "style":
			if style != "" {
				return "", fmt.Errorf("azure accepts one style per segment; use separate spans")
			}
			style, err = providers.Value(n)
		case "role":
			role, err = providers.Value(n)
		default:
			if style != "" {
				return "", fmt.Errorf("azure accepts one style per segment; use separate spans")
			}
			var ok bool
			style, ok = styles[n.Name]
			if !ok {
				return "", fmt.Errorf("unsupported tag [%s]", n.Name)
			}
			err = providers.NoArgs(n)
		}
		if err != nil {
			return "", err
		}
	}
	if role != "" && style == "" {
		return "", fmt.Errorf("[role] requires a style supported by the selected voice")
	}
	// Azure permits prosody inside emphasis, but not emphasis inside prosody.
	// Compose the planner's resolved controls independently of their source order.
	// https://learn.microsoft.com/azure/ai-services/speech-service/speech-synthesis-markup-structure
	if attrs := rate + pitch + volume; attrs != "" {
		text = `<prosody` + attrs + `>` + text + `</prosody>`
	}
	if emphasis {
		text = `<emphasis level="strong">` + text + `</emphasis>`
	}
	if style != "" {
		attrs := ` style="` + escape(style) + `"`
		if role != "" {
			attrs += ` role="` + escape(role) + `"`
		}
		text = `<mstts:express-as` + attrs + `>` + text + `</mstts:express-as>`
	}
	lang := seg.Lang
	if match := voiceLocale.FindStringSubmatch(seg.Voice); match != nil {
		lang = strings.ToLower(match[1])
		if script := match[2]; script != "" {
			lang += "-" + strings.ToUpper(script[:1]) + strings.ToLower(script[1:])
		}
		lang += "-" + strings.ToUpper(match[3])
	} else if lang == "" {
		return "", fmt.Errorf("set lang for voice %q", seg.Voice)
	}
	if seg.Lang != "" && !strings.EqualFold(seg.Lang, lang) {
		text = `<lang xml:lang="` + escape(seg.Lang) + `">` + text + `</lang>`
	}
	ssml := `<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="https://www.w3.org/2001/mstts" xml:lang="` + escape(lang) + `"><voice name="` + escape(seg.Voice) + `">` + text + `</voice></speak>`
	if len(ssml) > 64000 {
		return "", fmt.Errorf("SSML segment exceeds 64000 bytes; split it into paragraphs")
	}
	return ssml, nil
}

func escape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
