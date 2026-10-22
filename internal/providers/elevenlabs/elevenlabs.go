// Package elevenlabs implements ElevenLabs Create Speech.
package elevenlabs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func init() { providers.Register(provider{}) }

type provider struct{ client *http.Client }

func (provider) ID() string { return "elevenlabs" }

// https://elevenlabs.io/docs/api-reference/text-to-speech/convert
type request struct {
	Text     string    `json:"text"`
	Model    string    `json:"model_id"`
	Language string    `json:"language_code,omitempty"`
	Settings *settings `json:"voice_settings,omitempty"`
}
type settings struct {
	Speed float64 `json:"speed"`
}

var events = map[string]string{
	"laugh": "[laughs]", "sigh": "[sighs]", "gasp": "[gasps]",
	"clear-throat": "[clears throat]", "giggle": "[giggles]", "cough": "[coughs]",
}

var (
	breakTag   = regexp.MustCompile(`<break[\t\n\r />]`)
	phonemeTag = regexp.MustCompile(`</?phoneme[\t\n\r />]`)
)

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
		if seg.Voice == "" {
			return providers.Result{}, fmt.Errorf("elevenlabs: set --voice, providers.elevenlabs.voice, or cast voices.elevenlabs")
		}
		requests[i], err = build(seg, opt.Model)
		if err != nil {
			return providers.Result{}, fmt.Errorf("elevenlabs: %w", err)
		}
	}
	base := strings.TrimRight(opt.Endpoint, "/") + "/text-to-speech/"
	headers := map[string]string{
		"xi-api-key": opt.APIKey,
		"Accept":     "audio/pcm, application/octet-stream",
	}
	return providers.Render(ctx, p.ID(), opt, segments, func(ctx context.Context, i int) ([]byte, error) {
		endpoint := base + url.PathEscape(segments[i].Voice) + "?output_format=pcm_24000"
		return providers.PostJSON(ctx, p.client, endpoint, headers, requests[i])
	})
}

func build(seg providers.Segment, model string) (request, error) {
	r := request{Model: model}
	if err := providers.ReservedText(seg.Nodes, "[", "]"); err != nil {
		return r, err
	}
	inline := model == "eleven_v3" || model == "eleven_v4" || model == "eleven_v4_turbo"
	eventMap := events
	if !inline {
		eventMap = nil
	}
	text, err := providers.Text(seg.Nodes, eventMap)
	if err != nil {
		return r, err
	}
	if !inline && breakTag.MatchString(text) ||
		(model == "eleven_flash_v2" || model == "eleven_turbo_v2") && phonemeTag.MatchString(text) {
		return r, fmt.Errorf("literal SSML conflicts with this provider's native markup; use a Unicode alternative")
	}
	var prefix strings.Builder
	for _, n := range seg.Controls {
		if n.Name == "speed" || n.Name == "fast" || n.Name == "slow" {
			if model == "eleven_v4" || model == "eleven_v4_turbo" {
				return r, fmt.Errorf("[%s] is not supported by %s", n.Name, model)
			}
			speed, err := providers.Speed(n, 0.7, 1.2)
			if err != nil {
				return r, err
			}
			r.Settings = &settings{Speed: speed}
			continue
		}
		if !inline {
			return r, fmt.Errorf("[%s] requires an Eleven v3/v4 model", n.Name)
		}
		direction, err := providers.Direction(n)
		if err != nil {
			return r, err
		}
		if strings.ContainsAny(direction, "[]") {
			return r, fmt.Errorf("acting text cannot contain brackets for ElevenLabs")
		}
		switch n.Name {
		case "whisper":
			direction = "whispers"
		case "shout":
			direction = "shouts"
		}
		prefix.WriteString("[" + direction + "]")
	}
	r.Text = prefix.String() + text
	if utf8.RuneCountInString(r.Text) > 5000 {
		return r, fmt.Errorf("speech segment exceeds 5000 characters; split it into paragraphs")
	}
	if seg.Lang != "" {
		if model == "eleven_multilingual_v2" {
			return r, fmt.Errorf("eleven_multilingual_v2 does not support a language hint; omit lang")
		}
		r.Language = strings.ToLower(strings.Split(seg.Lang, "-")[0])
	}
	return r, nil
}
