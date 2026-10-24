// Package fish implements single-speaker Fish Audio S2 requests.
package fish

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

func init() { providers.Register(provider{}) }

type provider struct{ client *http.Client }

func (provider) ID() string { return "fish" }

// https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech
// https://docs.fish.audio/developer-guide/core-features/emotions
// https://docs.fish.audio/developer-guide/models-pricing/models-overview
type request struct {
	Features                  *[]options.FishFeature   `json:"features,omitempty"`
	PronunciationDictionary   *pronunciationDictionary `json:"pronunciation_dictionary,omitempty"`
	Text                      string                   `json:"text"`
	ReferenceID               string                   `json:"reference_id"`
	Format                    string                   `json:"format"`
	SampleRate                int                      `json:"sample_rate"`
	Normalize                 bool                     `json:"normalize"`
	Prosody                   prosody                  `json:"prosody"`
	Language                  *string                  `json:"language,omitempty"`
	Temperature               *float64                 `json:"temperature,omitempty"`
	TopP                      *float64                 `json:"top_p,omitempty"`
	ChunkLength               *int64                   `json:"chunk_length,omitempty"`
	MinChunkLength            *int64                   `json:"min_chunk_length,omitempty"`
	Latency                   *options.FishLatency     `json:"latency,omitempty"`
	ConditionOnPreviousChunks *bool                    `json:"condition_on_previous_chunks,omitempty"`
	EarlyStopThreshold        *float64                 `json:"early_stop_threshold,omitempty"`
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

var models = []string{"s2.1-pro", "s2.1-pro-free", "s2-pro"}

func (p provider) Synthesize(ctx context.Context, doc *ir.Document, opt providers.Options) (providers.Result, error) {
	if err := ctx.Err(); err != nil {
		return providers.Result{}, err
	}
	if err := providers.ValidateConcurrency(opt.Concurrency); err != nil {
		return providers.Result{}, err
	}
	synthesis, err := synthesisOptions(opt.Synthesis)
	if err != nil {
		return providers.Result{}, err
	}
	if !slices.Contains(models, opt.Model) {
		return providers.Result{}, fmt.Errorf("unsupported Fish TTS model")
	}
	segments, requests, err := providers.Prepare(ctx, doc, p.ID(), opt.Voice, func(seg providers.Segment) (request, error) {
		return synthesis.build(seg, opt.Model)
	})
	if err != nil {
		return providers.Result{}, err
	}
	if err := providers.ValidateBaseEndpoint(opt.Endpoint); err != nil {
		return providers.Result{}, err
	}
	headers := map[string]string{"Authorization": "Bearer " + opt.APIKey, "model": opt.Model}
	endpoint := strings.TrimRight(opt.Endpoint, "/") + "/tts"
	return providers.Render(ctx, p.ID(), opt, segments, func(ctx context.Context, i int) ([]byte, error) {
		wav, err := providers.PostJSON(ctx, p.client, endpoint, headers, requests[i], providers.ResponseBody)
		if err != nil {
			return nil, err
		}
		return providers.DecodeWAV(wav)
	})
}

func build(seg providers.Segment, model string) (request, error) {
	r := request{ReferenceID: seg.Voice, Format: "wav", SampleRate: providers.SampleRate, Normalize: false, Prosody: prosody{Speed: 1, NormalizeLoudness: true}}
	// Unknown model headers silently fall back at the service; never allow that.
	if !slices.Contains(models, model) {
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
		case "act":
			direction, err := providers.Direction(n)
			if err != nil {
				return r, err
			}
			if strings.TrimSpace(direction) == "" {
				return r, fmt.Errorf("[act] needs nonempty acting text")
			}
			if strings.ContainsAny(direction, "[]") || strings.Contains(direction, "<|") {
				return r, fmt.Errorf("acting text cannot contain brackets or native token prefixes for Fish")
			}
			prefix.WriteString("[" + direction + "] ")
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
