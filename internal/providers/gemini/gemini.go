// Package gemini implements unary speech generation through the Google Gemini API.
package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func init() { providers.Register(provider{}) }

type provider struct{ client *http.Client }

func (provider) ID() string { return "gemini" }

// https://ai.google.dev/api/generate-content
// https://ai.google.dev/gemini-api/docs/generate-content/speech-generation
// https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash-tts
// https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash-lite-tts
type request struct {
	Contents         []content        `json:"contents"`
	GenerationConfig generationConfig `json:"generationConfig"`
}

type content struct {
	Role  string `json:"role"`
	Parts []part `json:"parts"`
}

type part struct {
	Text           string   `json:"text"`
	SpeechMetadata metadata `json:"speechMetadata"`
}

type metadata struct {
	Style string `json:"style,omitempty"`
}

type generationConfig struct {
	ResponseModalities []string       `json:"responseModalities"`
	ResponseFormat     responseFormat `json:"responseFormat"`
	SpeechConfig       speechConfig   `json:"speechConfig"`
}

type responseFormat struct {
	Audio audioFormat `json:"audio"`
}

type audioFormat struct {
	MIMEType   string `json:"mimeType"`
	SampleRate int    `json:"sampleRate"`
	Delivery   string `json:"delivery"`
}

type speechConfig struct {
	VoiceConfig  voiceConfig `json:"voiceConfig"`
	LanguageCode string      `json:"languageCode,omitempty"`
}

type voiceConfig struct {
	PrebuiltVoiceConfig prebuiltVoice `json:"prebuiltVoiceConfig"`
}

type prebuiltVoice struct {
	VoiceName string `json:"voiceName"`
}

var models = []string{"gemini-3.8-flash-tts", "gemini-3.8-flash-lite-tts"}

var voices = strings.Fields("Zephyr Puck Charon Kore Fenrir Leda Orus Aoede Callirrhoe Autonoe Enceladus Iapetus Umbriel Algieba Despina Erinome Algenib Rasalgethi Laomedeia Achernar Alnilam Schedar Gacrux Pulcherrima Achird Zubenelgenubi Vindemiatrix Sadachbia Sadaltager Sulafat")

var languages = strings.Fields("de-DE en-AU en-GB en-IN en-US es-US fr-FR hi-IN pt-BR ar-XA es-ES fr-CA id-ID it-IT ja-JP tr-TR vi-VN bn-IN gu-IN kn-IN ml-IN mr-IN ta-IN te-IN nl-NL ko-KR cmn-CN pl-PL ru-RU th-TH")

// Only registered DSL events with documented native counterparts belong here.
// https://ai.google.dev/gemini-api/docs/generate-content/speech-generation#vocal-bursts-and-non-speech-sounds
var events = map[string]string{
	"laugh": "<laugh>", "sigh": "<sigh>", "cough": "<cough>", "breath": "<breath>", "gasp": "<gasp>",
	"exhale": "<exhales>", "chuckle": "<chuckle>", "giggle": "<giggle>", "cackle": "<cackle>", "snort": "<snort>",
	"cry": "<cry>", "sob": "<sob>", "whimper": "<whimper>", "pant": "<pant>", "groan": "<groan>",
	"grunt": "<grunt>", "moan": "<moan>", "sneeze": "<sneeze>", "yawn": "<yawn>", "hiss": "<hiss>",
	"clear-throat": "<throat-clearing>",
}

func (p provider) Synthesize(ctx context.Context, doc *ir.Document, opt providers.Options) (providers.Result, error) {
	if err := ctx.Err(); err != nil {
		return providers.Result{}, err
	}
	if err := providers.ValidateConcurrency(opt.Concurrency); err != nil {
		return providers.Result{}, err
	}
	if !slices.Contains(models, opt.Model) {
		return providers.Result{}, fmt.Errorf("unsupported Gemini TTS model")
	}
	segments, requests, err := providers.Prepare(ctx, doc, p.ID(), opt.Voice, func(seg providers.Segment) (request, error) { return build(seg, opt.Model) })
	if err != nil {
		return providers.Result{}, err
	}
	endpoint := strings.TrimRight(opt.Endpoint, "/") + "/models/" + opt.Model + ":generateContent"
	return providers.Render(ctx, p.ID(), opt, segments, func(ctx context.Context, i int) ([]byte, error) {
		data, err := providers.PostJSON(ctx, p.client, endpoint, map[string]string{"x-goog-api-key": opt.APIKey}, requests[i], providers.ResponseBody)
		if err != nil {
			return nil, err
		}
		return decode(data)
	})
}

func build(seg providers.Segment, model string) (request, error) {
	var r request
	if !slices.Contains(models, model) {
		return r, fmt.Errorf("unsupported Gemini TTS model")
	}
	if !slices.Contains(voices, seg.Voice) {
		return r, fmt.Errorf("gemini TTS requires a supported prebuilt voice")
	}
	if seg.Lang != "" && !slices.Contains(languages, seg.Lang) {
		return r, fmt.Errorf("unsupported Gemini speech language code")
	}
	// The API's generic languageCode list is broader than Flash-Lite's TTS
	// language inventory: Thai is documented only for Flash.
	if model == "gemini-3.8-flash-lite-tts" && seg.Lang == "th-TH" {
		return r, fmt.Errorf("gemini-3.8-flash-lite-tts does not support the th-TH language code")
	}
	if err := providers.ReservedText(seg.Nodes, "<", ">"); err != nil {
		return r, err
	}
	text, err := providers.Text(seg.Nodes, events)
	if err != nil {
		return r, err
	}
	var directions []string
	for _, n := range seg.Controls {
		var direction string
		switch n.Name {
		case "speed", "pitch", "volume":
			return r, fmt.Errorf("[%s] has no numeric Gemini TTS mapping", n.Name)
		case "fast", "slow":
			err = providers.NoArgs(n)
			direction = "speak faster"
			if n.Name == "slow" {
				direction = "speak slower"
			}
		default:
			direction, err = providers.Direction(n)
		}
		if err != nil {
			return r, err
		}
		directions = append(directions, direction)
	}
	style := strings.Join(directions, ". ")
	// Local resource bound, not an estimate of the service's token budget.
	if utf8.RuneCountInString(text)+utf8.RuneCountInString(style) > 4096 {
		return r, fmt.Errorf("gemini text and style exceed the adapter's 4096-character limit")
	}
	r.Contents = []content{{Role: "user", Parts: []part{{Text: text, SpeechMetadata: metadata{Style: style}}}}}
	r.GenerationConfig = generationConfig{
		ResponseModalities: []string{"AUDIO"},
		ResponseFormat:     responseFormat{Audio: audioFormat{MIMEType: "AUDIO_WAV", SampleRate: providers.SampleRate, Delivery: "INLINE"}},
		SpeechConfig:       speechConfig{VoiceConfig: voiceConfig{PrebuiltVoiceConfig: prebuiltVoice{VoiceName: seg.Voice}}, LanguageCode: seg.Lang},
	}
	return r, nil
}

func decode(data []byte) ([]byte, error) {
	var res struct {
		Error          json.RawMessage `json:"error"`
		PromptFeedback struct {
			BlockReason string `json:"blockReason"`
		} `json:"promptFeedback"`
		Candidates []struct {
			FinishReason  string `json:"finishReason"`
			SafetyRatings []struct {
				Blocked bool `json:"blocked"`
			} `json:"safetyRatings"`
			Content struct {
				Parts []map[string]json.RawMessage `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if len(data) > providers.MaxResponseBytes {
		return nil, fmt.Errorf("gemini response exceeds size limit")
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("invalid Gemini response JSON")
	}
	if len(res.Error) != 0 && string(res.Error) != "null" {
		return nil, fmt.Errorf("gemini API error")
	}
	if res.PromptFeedback.BlockReason != "" && res.PromptFeedback.BlockReason != "BLOCK_REASON_UNSPECIFIED" {
		return nil, fmt.Errorf("gemini prompt was blocked")
	}
	if len(res.Candidates) != 1 || res.Candidates[0].FinishReason != "STOP" {
		return nil, fmt.Errorf("gemini response needs one completed audio candidate")
	}
	candidate := res.Candidates[0]
	for _, rating := range candidate.SafetyRatings {
		if rating.Blocked {
			return nil, fmt.Errorf("gemini output was blocked")
		}
	}
	if len(candidate.Content.Parts) != 1 {
		return nil, fmt.Errorf("gemini response needs one inline audio part")
	}
	var thought bool
	if value, ok := candidate.Content.Parts[0]["thought"]; ok {
		if err := json.Unmarshal(value, &thought); err != nil || thought {
			return nil, fmt.Errorf("gemini response contains thought audio")
		}
	}
	// Part metadata may accompany its data union. Reject other payloads while
	// retaining the documented metadata fields on an inline audio response.
	for field := range candidate.Content.Parts[0] {
		switch field {
		case "inlineData", "thought", "thoughtSignature", "partMetadata", "mediaResolution", "mediaProcessing", "audioTranscription", "speechMetadata", "videoMetadata":
		default:
			return nil, fmt.Errorf("gemini response contains an unsupported audio part field")
		}
	}
	var audio struct {
		MIMEType string `json:"mimeType"`
		Data     string `json:"data"`
	}
	if err := json.Unmarshal(candidate.Content.Parts[0]["inlineData"], &audio); err != nil || audio.MIMEType != "audio/wav" {
		return nil, fmt.Errorf("gemini response must contain inline audio/wav")
	}
	wav, err := base64.StdEncoding.Strict().DecodeString(audio.Data)
	if err != nil {
		return nil, fmt.Errorf("invalid Gemini audio base64")
	}
	return providers.DecodeWAV(wav)
}
