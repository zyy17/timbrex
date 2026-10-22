// Package fish implements single-speaker Fish Audio S2.1-Pro requests.
package fish

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func init() { providers.Register(provider{}) }

type provider struct{ client *http.Client }

func (provider) ID() string { return "fish" }

// https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech
// https://docs.fish.audio/developer-guide/core-features/emotions
type request struct {
	Text        string  `json:"text"`
	ReferenceID string  `json:"reference_id"`
	Format      string  `json:"format"`
	SampleRate  int     `json:"sample_rate"`
	Normalize   bool    `json:"normalize"`
	Prosody     prosody `json:"prosody"`
}
type prosody struct {
	Speed             float64 `json:"speed"`
	Volume            float64 `json:"volume"`
	NormalizeLoudness bool    `json:"normalize_loudness"`
}

var cues = map[string]string{
	"happy": "happy", "sad": "sad", "angry": "angry", "fear": "scared", "surprise": "surprised", "disgust": "disgusted", "calm": "calm", "excited": "excited",
	"whisper": "whispering", "shout": "shouting", "emph": "emphasis",
}
var events = map[string]string{
	"laugh": "[laughing]", "chuckle": "[chuckling]", "sigh": "[sighing]", "groan": "[groaning]", "pant": "[panting]", "gasp": "[gasping]", "yawn": "[yawning]", "snore": "[snoring]", "clear-throat": "[clear throat]",
}

func (p provider) Synthesize(ctx context.Context, doc *ir.Document, opt providers.Options) (providers.Result, error) {
	segments, requests, err := providers.Prepare(doc, p.ID(), opt.Voice, func(seg providers.Segment) (request, error) { return build(seg, opt.Model) })
	if err != nil {
		return providers.Result{}, err
	}
	headers := map[string]string{"Authorization": "Bearer " + opt.APIKey, "model": opt.Model}
	endpoint := strings.TrimRight(opt.Endpoint, "/") + "/tts"
	return providers.Render(ctx, p.ID(), opt, segments, func(ctx context.Context, i int) ([]byte, error) {
		wav, err := providers.PostJSON(ctx, p.client, endpoint, headers, requests[i])
		if err != nil {
			return nil, err
		}
		return providers.DecodeWAV(wav)
	})
}
func build(seg providers.Segment, model string) (request, error) {
	r := request{ReferenceID: seg.Voice, Format: "wav", SampleRate: providers.SampleRate, Normalize: false, Prosody: prosody{Speed: 1, NormalizeLoudness: true}}
	// Unknown model headers silently fall back at the service; never allow that.
	if model != "s2.1-pro" {
		return r, fmt.Errorf("unsupported Fish TTS model")
	}
	if strings.TrimSpace(seg.Voice) == "" {
		return r, fmt.Errorf("fish requires an explicit voice reference ID")
	}
	if seg.Lang != "" {
		return r, fmt.Errorf("fish has no spoken-language override; its language field only controls text normalization")
	}
	if err := providers.ReservedText(seg.Nodes, "[", "]", "<|"); err != nil {
		return r, err
	}
	text, err := providers.Text(seg.Nodes, events)
	if err != nil {
		return r, err
	}
	var prefix strings.Builder
	for _, n := range seg.Controls {
		switch n.Name {
		case "speed", "fast", "slow":
			r.Prosody.Speed, err = providers.Speed(n, 0.5, 2)
		case "volume":
			r.Prosody.Volume, err = providers.Number(n, "dB", -60, 20)
		default:
			cue, ok := cues[n.Name]
			if !ok {
				return r, fmt.Errorf("unsupported tag [%s]", n.Name)
			}
			err = providers.NoArgs(n)
			prefix.WriteString("[" + cue + "] ")
		}
		if err != nil {
			return r, err
		}
	}
	r.Text = prefix.String() + text
	if utf8.RuneCountInString(r.Text) > 4096 {
		return r, fmt.Errorf("fish text exceeds the adapter's 4096-character limit")
	}
	return r, nil
}
