// Package minimax implements synchronous T2A v2.
package minimax

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func init() { providers.Register(provider{}) }

type provider struct{ client *http.Client }

func (provider) ID() string { return "minimax" }

// https://platform.minimax.io/docs/api-reference/speech-t2a-http
type request struct {
	Model    string        `json:"model"`
	Text     string        `json:"text"`
	Stream   bool          `json:"stream"`
	Output   string        `json:"output_format"`
	Language string        `json:"language_boost,omitempty"`
	Voice    voiceSettings `json:"voice_setting"`
	Audio    audioSettings `json:"audio_setting"`
}
type voiceSettings struct {
	ID      string  `json:"voice_id"`
	Speed   float64 `json:"speed"`
	Volume  float64 `json:"vol"`
	Pitch   int     `json:"pitch"`
	Emotion string  `json:"emotion,omitempty"`
}
type audioSettings struct {
	Rate    int    `json:"sample_rate"`
	Format  string `json:"format"`
	Channel int    `json:"channel"`
}

var emotions = map[string]string{"happy": "happy", "sad": "sad", "angry": "angry", "fear": "fearful", "disgust": "disgusted", "surprise": "surprised", "calm": "calm"}
var events = map[string]string{
	"laugh": "(laughs)", "chuckle": "(chuckle)", "cough": "(coughs)", "clear-throat": "(clear-throat)",
	"groan": "(groans)", "breath": "(breath)", "pant": "(pant)", "inhale": "(inhale)", "exhale": "(exhale)",
	"gasp": "(gasps)", "sniff": "(sniffs)", "sigh": "(sighs)", "snort": "(snorts)", "burp": "(burps)",
	"lip-smack": "(lip-smacking)", "hum": "(humming)", "hiss": "(hissing)", "emm": "(emm)", "sneeze": "(sneezes)",
}

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
			return providers.Result{}, fmt.Errorf("minimax: %w", err)
		}
	}
	endpoint := strings.TrimRight(opt.Endpoint, "/") + "/t2a_v2"
	headers := map[string]string{"Authorization": "Bearer " + opt.APIKey}
	return providers.Render(ctx, p.ID(), opt, segments, func(ctx context.Context, i int) ([]byte, error) {
		data, err := providers.PostJSON(ctx, p.client, endpoint, headers, requests[i])
		if err != nil {
			return nil, err
		}
		return decode(data)
	})
}

func build(seg providers.Segment, model string) (request, error) {
	r := request{Model: model, Output: "hex", Voice: voiceSettings{ID: seg.Voice, Speed: 1, Volume: 1}, Audio: audioSettings{Rate: providers.SampleRate, Format: "pcm", Channel: 1}}
	marks := []string{"<#"}
	for _, marker := range events {
		marks = append(marks, marker)
	}
	slices.Sort(marks)
	if err := providers.ReservedText(seg.Nodes, marks...); err != nil {
		return r, err
	}
	eventMap := events
	if model != "speech-2.8-hd" && model != "speech-2.8-turbo" {
		eventMap = nil
	}
	text, err := providers.Text(seg.Nodes, eventMap)
	if err != nil {
		return r, err
	}
	r.Text = text
	if seg.Lang != "" {
		lang := strings.ToLower(seg.Lang)
		languages := map[string]string{"zh": "Chinese", "en": "English", "ja": "Japanese", "ko": "Korean", "fr": "French", "de": "German", "es": "Spanish", "ru": "Russian", "pt": "Portuguese", "it": "Italian", "ar": "Arabic", "hi": "Hindi", "th": "Thai", "id": "Indonesian", "vi": "Vietnamese", "yue": "Chinese,Yue"}
		r.Language = languages[strings.Split(lang, "-")[0]]
		if lang == "zh-hk" || lang == "zh-mo" {
			r.Language = "Chinese,Yue"
		}
		if r.Language == "" {
			return r, fmt.Errorf("unsupported language hint %q", seg.Lang)
		}
	}
	if utf8.RuneCountInString(text) >= 10000 {
		return r, fmt.Errorf("speech segment must be shorter than 10000 characters")
	}
	for _, n := range seg.Controls {
		switch n.Name {
		case "speed", "fast", "slow":
			r.Voice.Speed, err = providers.Speed(n, 0.5, 2)
		case "pitch":
			var pitch float64
			pitch, err = providers.Number(n, "st", -12, 12)
			if err == nil && pitch != math.Trunc(pitch) {
				err = fmt.Errorf("MiniMax pitch needs whole semitones")
			}
			r.Voice.Pitch = int(pitch)
		case "volume":
			var db float64
			db, err = providers.Number(n, "dB", -60, 20)
			r.Voice.Volume = math.Pow(10, db/20)
		case "whisper":
			if model != "speech-2.6-hd" && model != "speech-2.6-turbo" {
				return r, fmt.Errorf("[whisper] requires speech-2.6-hd or speech-2.6-turbo")
			}
			if r.Voice.Emotion != "" {
				return r, fmt.Errorf("MiniMax cannot combine whisper with another emotion")
			}
			r.Voice.Emotion, err = "whisper", providers.NoArgs(n)
		default:
			emotion, ok := emotions[n.Name]
			if !ok {
				return r, fmt.Errorf("unsupported tag [%s]", n.Name)
			}
			if r.Voice.Emotion != "" {
				return r, fmt.Errorf("MiniMax cannot combine multiple emotions or delivery styles")
			}
			r.Voice.Emotion, err = emotion, providers.NoArgs(n)
		}
		if err != nil {
			return r, err
		}
	}
	return r, nil
}

func decode(data []byte) ([]byte, error) {
	var res struct {
		Data struct {
			Audio  string `json:"audio"`
			Status int    `json:"status"`
		} `json:"data"`
		Base *struct {
			Code int `json:"status_code"`
		} `json:"base_resp"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if res.Base == nil {
		return nil, fmt.Errorf("response is missing base_resp")
	}
	if res.Base.Code != 0 {
		return nil, fmt.Errorf("API status_code %d", res.Base.Code)
	}
	if res.Data.Status != 2 || res.Data.Audio == "" {
		return nil, fmt.Errorf("response has no completed audio")
	}
	pcm, err := hex.DecodeString(res.Data.Audio)
	if err != nil {
		return nil, fmt.Errorf("invalid hex audio: %w", err)
	}
	return pcm, nil
}
