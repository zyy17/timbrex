package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeminiCLIContract(t *testing.T) {
	for _, lateInvalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "preflight"}[lateInvalid], func(t *testing.T) {
			t.Setenv("GEMINI_API_KEY", "FAKE-GEMINI-KEY")
			t.Setenv("GEMINI_MODEL", "")
			t.Setenv("GEMINI_VOICE", "")
			var texts, styles, voices []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/models/gemini-3.8-flash-tts:generateContent" || r.URL.RawQuery != "" || r.Header.Get("x-goog-api-key") != "FAKE-GEMINI-KEY" || r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("wrong request %s %s", r.Method, r.URL.Path)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					return
				}
				var obj struct {
					Contents []struct {
						Parts []struct {
							Text           string
							SpeechMetadata struct{ Style string }
						}
					}
					GenerationConfig struct {
						ResponseModalities []string
						ResponseFormat     struct {
							Audio struct {
								MimeType, Delivery string
								SampleRate         int
							}
						}
						SpeechConfig struct {
							VoiceConfig struct{ PrebuiltVoiceConfig struct{ VoiceName string } }
						}
					}
				}
				if err := json.Unmarshal(body, &obj); err != nil {
					t.Error(err)
					return
				}
				if len(obj.Contents) != 1 || len(obj.Contents[0].Parts) != 1 {
					t.Error("bad content")
					return
				}
				part := obj.Contents[0].Parts[0]
				texts = append(texts, part.Text)
				styles = append(styles, part.SpeechMetadata.Style)
				voices = append(voices, obj.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName)
				a := obj.GenerationConfig.ResponseFormat.Audio
				if a.MimeType != "AUDIO_WAV" || a.Delivery != "INLINE" || a.SampleRate != 24000 || strings.Join(obj.GenerationConfig.ResponseModalities, ",") != "AUDIO" {
					t.Error("wrong output settings")
				}
				if len(texts) > 3 {
					t.Error("unexpected extra request")
					return
				}
				wantText := []string{" One ", "Two", "Three "}[len(texts)-1]
				wantStyle := []string{`{"style":"happy"}`, `{"style":"happy. whisper"}`, `{}`}[len(texts)-1]
				wantVoice := []string{"Kore", "Kore", "Puck"}[len(texts)-1]
				assertJSONRequest(t, body, json.RawMessage(fmt.Sprintf(`{"contents":[{"role":"user","parts":[{"text":%q,"speechMetadata":%s}]}],"generationConfig":{"responseModalities":["AUDIO"],"responseFormat":{"audio":{"mimeType":"AUDIO_WAV","sampleRate":24000,"delivery":"INLINE"}},"speechConfig":{"voiceConfig":{"prebuiltVoiceConfig":{"voiceName":%q}}}}}`, wantText, wantStyle, wantVoice)))
				wav, err := hex.DecodeString("524946462800000057415645666d74201000000001000100c05d000080bb000002001000646174610400000001000200")
				if err != nil {
					t.Error(err)
					return
				}
				copy(wav[44:], contractPCM(len(texts)))
				_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []any{map[string]any{"finishReason": "STOP", "content": map[string]any{"parts": []any{map[string]any{"inlineData": map[string]string{"mimeType": "audio/wav", "data": base64.StdEncoding.EncodeToString(wav)}}}}}}})
			}))
			defer srv.Close()
			t.Setenv("GEMINI_BASE_URL", srv.URL)
			dir := t.TempDir()
			path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "out.wav")
			src := "---\nprovider: gemini\ncast:\n  a: {voices: {gemini: Kore}}\n  b: {voices: {gemini: Puck}}\n---\na: [pause 10ms] [happy] One [pause 10ms] [whisper|Two]\nb: Three [pause 10ms]\n"
			if lateInvalid {
				src += "\nb: [speed 1.2] Invalid\n"
			}
			if err := os.WriteFile(path, []byte(src), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(out, []byte("KEEP"), 0600); err != nil {
				t.Fatal(err)
			}
			err := run(context.Background(), []string{"synth", path, "-o", out}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
			data, readErr := os.ReadFile(out)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if lateInvalid {
				if err == nil || !strings.Contains(err.Error(), "speed") || len(texts) != 0 || string(data) != "KEEP" {
					t.Fatalf("err=%v calls=%d", err, len(texts))
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(texts, "|") != " One |Two|Three " || strings.Join(voices, "|") != "Kore|Kore|Puck" || styles[0] != "happy" || styles[1] != "happy. whisper" || styles[2] != "" {
				t.Fatalf("texts=%v voices=%v styles=%v", texts, voices, styles)
			}
			wantPCM := append(make([]byte, 480), contractPCM(1)...)
			wantPCM = append(wantPCM, make([]byte, 480)...)
			wantPCM = append(wantPCM, contractPCM(2)...)
			wantPCM = append(wantPCM, contractPCM(3)...)
			wantPCM = append(wantPCM, make([]byte, 480)...)
			if len(data) != 44+len(wantPCM) || !bytes.Equal(data[44:], wantPCM) {
				t.Fatalf("incorrect WAV/PCM order/pause bytes: %x", data)
			}
		})
	}
}
