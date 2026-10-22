// Package qwen implements the Qwen-Audio-TTS HTTP API.
package qwen

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func init() { providers.Register(provider{}) }

type provider struct{ client *http.Client }

func (provider) ID() string { return "qwen" }

// https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api
type request struct {
	Model string `json:"model"`
	Input input  `json:"input"`
}
type input struct {
	Text        string   `json:"text"`
	Voice       string   `json:"voice"`
	Format      string   `json:"format"`
	SampleRate  int      `json:"sample_rate"`
	Instruction string   `json:"instruction,omitempty"`
	Rate        float64  `json:"rate,omitempty"`
	Pitch       float64  `json:"pitch,omitempty"`
	Volume      *int     `json:"volume,omitempty"`
	Languages   []string `json:"language_hints,omitempty"`
}

var events = map[string]string{"laugh": "[laughing]", "sigh": "[sighing]", "gasp": "[gasp]", "clear-throat": "[clears throat]", "giggle": "[giggles]", "cough": "[cough]", "snort": "[snorts]"}

func (p provider) Synthesize(ctx context.Context, doc *ir.Document, opt providers.Options) (providers.Result, error) {
	segments, err := providers.Plan(doc, p.ID(), opt.Voice)
	if err != nil {
		return providers.Result{}, err
	}
	requests := make([]request, len(segments))
	for i, seg := range segments {
		if seg.Pause > 0 {
			continue
		}
		requests[i], err = build(seg, opt.Model)
		if err != nil {
			return providers.Result{}, fmt.Errorf("qwen: %w", err)
		}
	}
	headers := map[string]string{"Authorization": "Bearer " + opt.APIKey}
	return providers.Render(ctx, p.ID(), opt, segments, func(ctx context.Context, i int) ([]byte, error) {
		data, err := providers.PostJSON(ctx, p.client, opt.Endpoint, headers, requests[i])
		if err != nil {
			return nil, err
		}
		return p.download(ctx, data)
	})
}

func build(seg providers.Segment, model string) (request, error) {
	r := request{Model: model, Input: input{Voice: seg.Voice, Format: "pcm", SampleRate: providers.SampleRate}}
	switch model {
	case "qwen-audio-3.1-tts-flash", "qwen-audio-3.1-tts-plus", "qwen-audio-3.0-tts-flash", "qwen-audio-3.0-tts-plus":
	default:
		return r, fmt.Errorf("model %q is not supported by the Qwen-Audio-TTS adapter", model)
	}
	if err := providers.ReservedText(seg.Nodes, "[", "]"); err != nil {
		return r, err
	}
	text, err := providers.Text(seg.Nodes, events)
	if err != nil {
		return r, err
	}
	r.Input.Text = text
	var directions []string
	for _, n := range seg.Controls {
		switch n.Name {
		case "speed", "fast", "slow":
			r.Input.Rate, err = providers.Speed(n, 0.5, 2)
		case "pitch":
			var pitch float64
			pitch, err = providers.Number(n, "st", -12, 12)
			r.Input.Pitch = math.Pow(2, pitch/12)
		case "volume":
			var db float64
			db, err = providers.Number(n, "dB", -60, 6.020599913279624)
			volume := int(math.Round(50 * math.Pow(10, db/20)))
			r.Input.Volume = &volume
		default:
			var direction string
			direction, err = providers.Direction(n)
			directions = append(directions, direction)
		}
		if err != nil {
			return r, err
		}
	}
	r.Input.Instruction = strings.Join(directions, ". ")
	if seg.Lang != "" {
		lang := strings.ToLower(strings.Split(seg.Lang, "-")[0])
		switch lang {
		case "zh", "en", "fr", "de", "ja", "ko", "ru", "pt", "th", "id", "vi", "es", "it", "ms", "fil", "ar":
			r.Input.Languages = []string{lang}
		default:
			return r, fmt.Errorf("unsupported language hint %q", seg.Lang)
		}
	}
	return r, nil
}

func (p provider) download(ctx context.Context, data []byte) ([]byte, error) {
	var res struct {
		Code   string `json:"code"`
		Output struct {
			FinishReason string `json:"finish_reason"`
			Audio        struct {
				URL string `json:"url"`
			} `json:"audio"`
		} `json:"output"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if res.Code != "" {
		return nil, fmt.Errorf("API error %s", res.Code)
	}
	if res.Output.FinishReason != "stop" {
		return nil, fmt.Errorf("response has no completed audio")
	}
	u, err := url.Parse(res.Output.Audio.URL)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, fmt.Errorf("response has no valid audio URL")
	}
	// The audio URL is signed; it must never receive the DashScope API key.
	return providers.Request(ctx, p.client, http.MethodGet, u.String(), "", map[string]string{"Accept": "audio/pcm, application/octet-stream"}, nil)
}
