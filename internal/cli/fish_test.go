package cli

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFishCLIContract(t *testing.T) {
	for _, invalid := range []string{"", "pitch", "lang", "voice", "model", "literal", "as"} {
		t.Run("invalid_"+invalid, func(t *testing.T) {
			t.Setenv("FISH_API_KEY", "FAKE-FISH-KEY")
			t.Setenv("FISH_MODEL", "")
			t.Setenv("FISH_VOICE_ID", "")
			var texts, voices []string
			var speeds []float64
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/tts" || r.Header.Get("Authorization") != "Bearer FAKE-FISH-KEY" || r.Header.Get("model") != "s2.1-pro" || r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("wrong request %s %s", r.Method, r.URL.Path)
				}
				var obj struct {
					Text        string
					ReferenceID string `json:"reference_id"`
					Format      string
					SampleRate  int `json:"sample_rate"`
					Normalize   bool
					Prosody     struct {
						Speed, Volume     float64
						NormalizeLoudness bool `json:"normalize_loudness"`
					}
				}
				if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
					t.Error(err)
					return
				}
				texts = append(texts, strings.TrimSpace(obj.Text))
				voices = append(voices, obj.ReferenceID)
				speeds = append(speeds, obj.Prosody.Speed)
				if obj.Format != "wav" || obj.SampleRate != 24000 || obj.Normalize || !obj.Prosody.NormalizeLoudness {
					t.Error("wrong audio/normalization settings")
				}
				wav, _ := hex.DecodeString("524946462800000057415645666d74201000000001000100c05d000080bb000002001000646174610400000001000200")
				// A valid non-44-byte header must be decoded, not blindly stripped.
				wav = append(append(append([]byte{}, wav[:12]...), []byte("JUNK\x01\x00\x00\x00x\x00")...), wav[12:]...)
				binary.LittleEndian.PutUint32(wav[4:], uint32(len(wav)-8))
				w.Header().Set("Content-Type", "audio/wav")
				_, _ = w.Write(wav)
			}))
			defer srv.Close()
			t.Setenv("FISH_BASE_URL", srv.URL)
			dir := t.TempDir()
			path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "out.wav")
			src := "---\nprovider: fish\ncast:\n  a: {voices: {fish: test-reference-a}}\n  b: {voices: {fish: test-reference-b}}\n---\na: [pause 10ms] [happy] One [pause 10ms] [speed 1.2|Two]\nb: Three [pause 10ms]\n"
			switch invalid {
			case "pitch":
				src += "\nb: [pitch 2st] Invalid\n"
			case "lang":
				src += "\nb: [lang en-US|Invalid]\n"
			case "voice":
				src += "\nNarration has no reference.\n"
			case "model":
				t.Setenv("FISH_MODEL", "unknown")
			case "literal":
				src += "\nb: Literal <|speaker:0|>\n"
			case "as":
				src += "\nb: [as ghost|Invalid]\n"
			}
			if err := os.WriteFile(path, []byte(src), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(out, []byte("KEEP"), 0600); err != nil {
				t.Fatal(err)
			}
			err := run(context.Background(), []string{"synth", path, "-o", out}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
			data, _ := os.ReadFile(out)
			if invalid != "" {
				if err == nil || len(texts) != 0 || string(data) != "KEEP" {
					t.Fatalf("err=%v calls=%d", err, len(texts))
				}
				if invalid == "as" && !strings.Contains(err.Error(), "[as ghost]") {
					t.Fatalf("as error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(texts, "|") != "[happy]  One|[happy] Two|Three" || strings.Join(voices, "|") != "test-reference-a|test-reference-a|test-reference-b" || speeds[0] != 1 || speeds[1] != 1.2 || speeds[2] != 1 {
				t.Fatalf("texts=%v voices=%v speed=%v", texts, voices, speeds)
			}
			if len(data) != 44+12+3*480 || !bytes.Equal(data[44:524], make([]byte, 480)) || !bytes.Equal(data[528:1008], make([]byte, 480)) || !bytes.Equal(data[len(data)-480:], make([]byte, 480)) {
				t.Fatalf("incorrect WAV/pause bytes: %d", len(data))
			}
		})
	}
}
