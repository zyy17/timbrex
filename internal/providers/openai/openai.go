// Package openai implements the Audio Speech REST API.
package openai

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

func (provider) ID() string { return "openai" }

// https://developers.openai.com/api/reference/resources/audio/subresources/speech/methods/create
type request struct {
	Model        string  `json:"model"`
	Input        string  `json:"input"`
	Voice        string  `json:"voice"`
	Format       string  `json:"response_format"`
	Instructions string  `json:"instructions,omitempty"`
	Speed        float64 `json:"speed,omitempty"`
}

func (p provider) Synthesize(ctx context.Context, doc *ir.Document, opt providers.Options) (providers.Result, error) {
	if err := ctx.Err(); err != nil {
		return providers.Result{}, err
	}
	if err := providers.ValidateConcurrency(opt.Concurrency); err != nil {
		return providers.Result{}, err
	}
	if opt.Synthesis != nil {
		return providers.Result{}, fmt.Errorf("openai received an unsupported synthesis options type")
	}
	segments, requests, err := providers.Prepare(ctx, doc, p.ID(), opt.Voice, func(seg providers.Segment) (request, error) { return build(seg, opt.Model) })
	if err != nil {
		return providers.Result{}, err
	}
	endpoint := strings.TrimRight(opt.Endpoint, "/") + "/audio/speech"
	headers := map[string]string{
		"Authorization": "Bearer " + opt.APIKey,
		"Accept":        "audio/pcm, application/octet-stream",
	}
	return providers.Render(ctx, p.ID(), opt, segments, func(ctx context.Context, i int) ([]byte, error) {
		return providers.PostJSON(ctx, p.client, endpoint, headers, requests[i], providers.ResponsePCM)
	})
}

func build(seg providers.Segment, model string) (request, error) {
	r := request{Model: model, Voice: seg.Voice, Format: "pcm"}
	text, err := providers.Text(seg.Nodes, nil)
	if err != nil {
		return r, err
	}
	if utf8.RuneCountInString(text) > 4096 {
		return r, fmt.Errorf("speech segment exceeds 4096 characters; split it into paragraphs")
	}
	r.Input = text
	var directions []string
	for _, n := range seg.Controls {
		switch n.Name {
		case "speed", "fast", "slow":
			r.Speed, err = providers.Speed(n, 0.25, 4)
		default:
			var direction string
			direction, err = providers.Direction(n)
			directions = append(directions, direction)
		}
		if err != nil {
			return r, err
		}
	}
	if (len(directions) > 0 || seg.Lang != "") && (model == "tts-1" || model == "tts-1-hd") {
		return r, fmt.Errorf("%s does not support acting instructions; use gpt-4o-mini-tts", model)
	}
	if seg.Lang != "" {
		directions = append(directions, "Speak in "+seg.Lang)
	}
	r.Instructions = strings.Join(directions, ". ")
	if utf8.RuneCountInString(r.Instructions) > 4096 {
		return r, fmt.Errorf("acting instructions exceed 4096 characters")
	}
	return r, nil
}
