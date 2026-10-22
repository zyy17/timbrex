// Package elevenlabs implements ElevenLabs Create Speech.
package elevenlabs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
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

func (p provider) Synthesize(ctx context.Context, doc *ir.Document, opt providers.Options) (providers.Result, error) {
	key, err := providers.RequireEnv("ELEVENLABS_API_KEY")
	if err != nil {
		return providers.Result{}, err
	}
	model := providers.First(opt.Model, doc.Model, os.Getenv("ELEVENLABS_MODEL"), "eleven_v3")
	voice := providers.First(opt.Voice, doc.Voice, os.Getenv("ELEVENLABS_VOICE_ID"))
	segments, err := providers.Plan(doc, p.ID(), voice)
	if err != nil {
		return providers.Result{}, err
	}
	requests := make([]request, len(segments))
	for i, seg := range segments {
		if seg.Pause > 0 {
			continue
		}
		if seg.Voice == "" {
			return providers.Result{}, fmt.Errorf("elevenlabs: set --voice, ELEVENLABS_VOICE_ID, or cast voices.elevenlabs")
		}
		requests[i], err = build(seg, model)
		if err != nil {
			return providers.Result{}, fmt.Errorf("elevenlabs: %w", err)
		}
	}
	base := providers.First(os.Getenv("ELEVENLABS_BASE_URL"), "https://api.elevenlabs.io/v1")
	return providers.Render(ctx, p.ID(), opt, segments, func(ctx context.Context, i int) ([]byte, error) {
		endpoint := strings.TrimRight(base, "/") + "/text-to-speech/" + url.PathEscape(segments[i].Voice) + "?output_format=pcm_24000"
		return providers.PostJSON(ctx, p.client, endpoint, map[string]string{"xi-api-key": key, "Accept": "audio/*, application/octet-stream"}, requests[i])
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
	var prefix strings.Builder
	for _, n := range seg.Controls {
		if n.Name == "speed" || n.Name == "fast" || n.Name == "slow" {
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
