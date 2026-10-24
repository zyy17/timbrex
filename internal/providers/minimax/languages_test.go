package minimax

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

var languageCases = []struct {
	code, value string
	modernOnly  bool
}{
	{"zh", "Chinese", false}, {"yue", "Chinese,Yue", false},
	{"en", "English", false}, {"ar", "Arabic", false},
	{"ru", "Russian", false}, {"es", "Spanish", false},
	{"fr", "French", false}, {"pt", "Portuguese", false},
	{"de", "German", false}, {"tr", "Turkish", false},
	{"nl", "Dutch", false}, {"uk", "Ukrainian", false},
	{"vi", "Vietnamese", false}, {"id", "Indonesian", false},
	{"ja", "Japanese", false}, {"it", "Italian", false},
	{"ko", "Korean", false}, {"th", "Thai", false},
	{"pl", "Polish", false}, {"ro", "Romanian", false},
	{"el", "Greek", false}, {"cs", "Czech", false},
	{"fi", "Finnish", false}, {"hi", "Hindi", false},
	{"bg", "Bulgarian", false}, {"da", "Danish", false},
	{"he", "Hebrew", false}, {"ms", "Malay", false},
	{"fa", "Persian", true}, {"sk", "Slovak", false},
	{"sv", "Swedish", false}, {"hr", "Croatian", false},
	{"fil", "Filipino", true}, {"hu", "Hungarian", false},
	{"no", "Norwegian", false}, {"sl", "Slovenian", false},
	{"ca", "Catalan", false}, {"nn", "Nynorsk", false},
	{"ta", "Tamil", true}, {"af", "Afrikaans", false},
	{"auto", "auto", false}, {"", "", false},
}

var modelCases = []struct {
	name   string
	modern bool
}{
	{"speech-2.8-hd", true}, {"speech-2.8-turbo", true},
	{"speech-2.6-hd", true}, {"speech-2.6-turbo", true},
	{"speech-02-hd", false}, {"speech-02-turbo", false},
	{"speech-01-hd", false}, {"speech-01-turbo", false},
}

func TestAllLanguageModels(t *testing.T) {
	for _, model := range modelCases {
		for _, tc := range languageCases {
			t.Run(model.name+"/"+tc.code, func(t *testing.T) {
				seg := providers.Segment{Voice: "voice", Lang: tc.code, Nodes: []ir.Node{{Text: "Hello"}}}
				r, err := build(seg, model.name)
				if tc.modernOnly && !model.modern {
					if err == nil || !strings.Contains(err.Error(), "requires speech-2.6") {
						t.Fatalf("legacy model accepted %s: request=%+v error=%v", tc.code, r, err)
					}
					return
				}
				if err != nil || r.Language != tc.value {
					t.Fatalf("language=%q, want %q; error=%v", r.Language, tc.value, err)
				}
			})
		}
	}
}

func TestLanguageLocalesAndCompatibility(t *testing.T) {
	for _, tc := range []struct{ hint, want string }{
		{"EN-us", "English"}, {"en-Latn-US", "English"}, {"es-419", "Spanish"},
		{"zh-CN", "Chinese"}, {"zh-TW", "Chinese"}, {"zh-Hans", "Chinese"},
		{"zh-Hant-TW", "Chinese"}, {"zh-Hant-HK", "Chinese"},
		{"ZH-hK", "Chinese,Yue"}, {"zh-MO", "Chinese,Yue"},
		{"yue-HK", "Chinese,Yue"}, {"yue-Hant-HK", "Chinese,Yue"},
		{"ja-JP", "Japanese"}, {"ko-KR", "Korean"}, {"fr-CA", "French"},
		{"de-DE", "German"}, {"ru-RU", "Russian"}, {"pt-BR", "Portuguese"},
		{"it-IT", "Italian"}, {"ar-SA", "Arabic"}, {"hi-IN", "Hindi"},
		{"th-TH", "Thai"}, {"id-ID", "Indonesian"}, {"vi-VN", "Vietnamese"},
		{"tr-TR", "Turkish"}, {"nl-NL", "Dutch"}, {"uk-UA", "Ukrainian"},
		{"pl-PL", "Polish"}, {"ro-RO", "Romanian"}, {"el-GR", "Greek"},
		{"cs-CZ", "Czech"}, {"fi-FI", "Finnish"}, {"bg-BG", "Bulgarian"},
		{"da-DK", "Danish"}, {"he-IL", "Hebrew"}, {"ms-MY", "Malay"},
		{"fa-IR", "Persian"}, {"sk-SK", "Slovak"}, {"sv-SE", "Swedish"},
		{"hr-HR", "Croatian"}, {"fil-PH", "Filipino"}, {"hu-HU", "Hungarian"},
		{"no-NO", "Norwegian"}, {"nn-NO", "Nynorsk"}, {"sl-SI", "Slovenian"},
		{"ca-ES", "Catalan"}, {"ta-IN", "Tamil"}, {"af-ZA", "Afrikaans"},
		{"AUTO", "auto"},
	} {
		t.Run(tc.hint, func(t *testing.T) {
			got, err := languageBoost(tc.hint, "speech-2.8-hd")
			if err != nil || got != tc.want {
				t.Fatalf("language=%q, want %q; error=%v", got, tc.want, err)
			}
		})
	}
	for _, model := range modelCases {
		if model.modern {
			continue
		}
		for _, hint := range []string{"FA-ir", "fil-Latn-PH", "ta-IN"} {
			if _, err := languageBoost(hint, model.name); err == nil {
				t.Errorf("%s accepted %s", model.name, hint)
			}
		}
	}
}

func TestLanguageRejectsUnknownAndMalformedHints(t *testing.T) {
	for _, hint := range []string{
		"unknown", "US", "UA", "GR", "CZ", "PH", "nb", "nb-NO", "tl", "tl-PH",
		"English", "Chinese,Yue", "eng", "iw", "in", "und", "zh-yue", "zh-cmn-CN",
		" ", " en", "en ", "-en", "en-", "en--US", "en_US", "en-foo", "en-US-Latn",
		"en-US-extra", "en-x-private", "en-u-ca-gregory", "en-1", "en-1234", "en-ÜS",
		"auto-US", "auto-Hant", "auto-", "en\n", "en\x00",
	} {
		t.Run(hint, func(t *testing.T) {
			if _, err := languageBoost(hint, "speech-2.8-hd"); err == nil {
				t.Fatal("accepted unsupported or malformed hint")
			}
		})
	}
	for _, model := range []string{"", "speech-2.8", "speech-2.8-hd-preview", "speech-02", "SPEECH-2.8-HD", "custom-model"} {
		for _, hint := range []string{"en", "fa", "auto"} {
			if _, err := languageBoost(hint, model); err == nil {
				t.Errorf("unknown model %q accepted %q", model, hint)
			}
		}
		// Model passthrough for requests without language hints is unchanged.
		r, err := build(providers.Segment{Voice: "voice", Nodes: []ir.Node{{Text: "Hello"}}}, model)
		if err != nil || r.Model != model || r.Language != "" {
			t.Errorf("plain custom model changed: request=%+v error=%v", r, err)
		}
	}
}

func TestLanguageHTTPContracts(t *testing.T) {
	for _, concurrency := range []int{1, 3} {
		for _, model := range modelCases {
			t.Run(fmt.Sprintf("%s/concurrency=%d", model.name, concurrency), func(t *testing.T) {
				doc := &ir.Document{}
				wantRequests := make(map[string]map[string]any)
				wantAudio := make(map[string]byte)
				var pcm []byte
				for _, tc := range languageCases {
					if tc.modernOnly && !model.modern {
						continue
					}
					text := "text " + tc.code
					nodes := []ir.Node{{Text: text}}
					if tc.code != "" {
						nodes = []ir.Node{{Kind: ir.SpanStart, Name: "lang", Args: []ir.Arg{{Raw: tc.code}}}, {Text: text}, {Kind: ir.SpanEnd, Name: "lang"}}
					}
					doc.Utterances = append(doc.Utterances, ir.Utterance{Speaker: "narrator", Nodes: nodes})
					want := map[string]any{
						"model": model.name, "text": text, "stream": false, "output_format": "hex",
						"voice_setting": map[string]any{"voice_id": "custom-voice", "speed": float64(1), "vol": float64(1), "pitch": float64(0)},
						"audio_setting": map[string]any{"sample_rate": float64(24000), "format": "pcm", "channel": float64(1)},
					}
					if tc.code != "" {
						want["language_boost"] = tc.value
					}
					wantRequests[text] = want
					wantAudio[text] = byte(len(doc.Utterances))
					pcm = append(pcm, wantAudio[text], 0)
				}
				var calls atomic.Int32
				p := provider{client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if r.Method != http.MethodPost || r.URL.String() != "https://example.invalid/v1/t2a_v2" || r.Header.Get("Authorization") != "Bearer test-key" || r.Header.Get("Content-Type") != "application/json" {
						t.Errorf("unexpected HTTP request: %s %s %v", r.Method, r.URL, r.Header)
					}
					var got map[string]any
					if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
						return nil, err
					}
					text, _ := got["text"].(string)
					if !reflect.DeepEqual(got, wantRequests[text]) {
						t.Errorf("body=%v, want %v", got, wantRequests[text])
					}
					body := fmt.Sprintf(`{"base_resp":{"status_code":0},"data":{"status":2,"audio":"%02x00"}}`, wantAudio[text])
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
				})}}
				out := filepath.Join(t.TempDir(), "out.wav")
				_, err := p.Synthesize(context.Background(), doc, providers.Options{Out: out, Voice: "custom-voice", Model: model.name, Endpoint: "https://example.invalid/v1///", APIKey: "test-key", Concurrency: concurrency})
				if err != nil || calls.Load() != int32(len(doc.Utterances)) {
					t.Fatalf("calls=%d, want %d; error=%v", calls.Load(), len(doc.Utterances), err)
				}
				wav, err := os.ReadFile(out)
				if err != nil {
					t.Fatal(err)
				}
				gotPCM, err := providers.DecodeWAV(wav)
				if err != nil || !bytes.Equal(gotPCM, pcm) {
					t.Fatalf("ordered PCM=%x, want %x; error=%v", gotPCM, pcm, err)
				}
			})
		}
	}
}

func TestLateInvalidLanguageMakesNoRequests(t *testing.T) {
	for _, concurrency := range []int{1, 3} {
		for _, tc := range []struct{ model, hint string }{
			{"speech-2.8-hd", "en-"}, {"speech-2.8-hd", "en-foo"},
			{"speech-2.8-hd", "unknown"}, {"speech-2.8-hd", "nb-NO"},
			{"speech-2.8-hd", "tl-PH"}, {"speech-2.8-hd", "auto-US"},
			{"custom-model", "en"}, {"custom-model", "auto"},
			{"speech-01-hd", "fa"}, {"speech-01-turbo", "fil-PH"},
			{"speech-02-hd", "ta-IN"}, {"speech-02-turbo", "FA-ir"},
		} {
			t.Run(fmt.Sprintf("%s/%s/concurrency=%d", tc.model, tc.hint, concurrency), func(t *testing.T) {
				nodes := []ir.Node{{Kind: ir.SpanStart, Name: "lang", Args: []ir.Arg{{Raw: tc.hint}}}, {Text: "Late invalid"}, {Kind: ir.SpanEnd, Name: "lang"}}
				assertLatePreflightFailure(t, tc.model, concurrency, nodes)
			})
		}
	}
}

func assertLatePreflightFailure(t *testing.T, model string, concurrency int, nodes []ir.Node) {
	t.Helper()
	var calls atomic.Int32
	p := provider{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, fmt.Errorf("unexpected request")
	})}}
	out := filepath.Join(t.TempDir(), "out.wav")
	if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
		t.Fatal(err)
	}
	doc := &ir.Document{Utterances: []ir.Utterance{
		{Speaker: "narrator", Nodes: []ir.Node{{Text: "Valid first"}}},
		{Speaker: "narrator", Nodes: nodes},
	}}
	_, err := p.Synthesize(context.Background(), doc, providers.Options{Out: out, Voice: "voice", Model: model, Endpoint: "https://example.invalid", Concurrency: concurrency})
	if err == nil || calls.Load() != 0 {
		t.Fatalf("error=%v calls=%d", err, calls.Load())
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "KEEP" {
		t.Fatalf("output=%q error=%v", data, err)
	}
	partials, err := filepath.Glob(filepath.Join(filepath.Dir(out), ".timbrex-*.partial"))
	if err != nil || len(partials) != 0 {
		t.Fatalf("temporary files=%v error=%v", partials, err)
	}
}
