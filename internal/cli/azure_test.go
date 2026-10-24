package cli

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAzureSSMLCLIContracts(t *testing.T) {
	const emphasis = `<emphasis level="strong"><prosody rate="1.2">Hello.</prosody></emphasis>`
	const combined = `<lang xml:lang="de-DE"><mstts:express-as style="cheerful" role="YoungAdultMale"><emphasis level="strong"><prosody rate="1.2" pitch="+2st" volume="-49.8813%">Hello &amp; goodbye.</prosody></emphasis></mstts:express-as></lang>`
	// Combined controls test the request shape, not voice-dependent availability.
	// HD cases deliberately contain no prosody, style, or emphasis claims.
	for _, tc := range []struct{ name, voice, source, locale, content string }{
		{"emphasis-first", "en-US-GuyNeural", "[emph][speed 1.2]Hello.", "en-US", emphasis},
		{"speed-first", "en-US-GuyNeural", "[speed 1.2][emph]Hello.", "en-US", emphasis},
		{"emphasis-outer-span", "en-US-GuyNeural", "[emph|[speed 1.2|Hello.]]", "en-US", emphasis},
		{"speed-outer-span", "en-US-GuyNeural", "[speed 1.2|[emph|Hello.]]", "en-US", emphasis},
		{"last-speed-wins", "en-US-GuyNeural", "[emph][slow][speed 1.2]Hello.", "en-US", emphasis},
		{"combined-forward", "en-US-GuyNeural", "[lang de-DE|[style cheerful][role YoungAdultMale][emph][speed 1.2][pitch +2st][volume -6dB]Hello & goodbye.]", "en-US", combined},
		{"combined-reverse", "en-US-GuyNeural", "[volume -6dB][pitch +2st][speed 1.2][emph][role YoungAdultMale][style cheerful][lang de-DE|Hello & goodbye.]", "en-US", combined},
		{"script-locale", "sr-Latn-RS-NicholasNeural", "Zdravo.", "sr-Latn-RS", "Zdravo."},
		{"script-locale-override", "sr-Latn-RS-SophieNeural", "[lang sr-RS|Zdravo.]", "sr-Latn-RS", `<lang xml:lang="sr-RS">Zdravo.</lang>`},
		{"lowercase-hd-locale", "en-us-Ava:DragonHDLatestNeural", "Hello.", "en-US", "Hello."},
		{"lowercase-hd-override", "en-us-Ava:DragonHDLatestNeural", "[lang de-DE|Hallo.]", "en-US", `<lang xml:lang="de-DE">Hallo.</lang>`},
		{"locale-case-only", "en-US-GuyNeural", "[lang en-us|Hello.]", "en-US", "Hello."},
		{"custom-explicit-locale", "my-custom-voice", "[lang fr-FR|Bonjour.]", "fr-FR", "Bonjour."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearProviderDefaults(t)
			requests := make(chan string, 2)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/cognitiveservices/v1" || r.URL.RawQuery != "" {
					t.Errorf("method/URL = %s %s", r.Method, r.URL)
				}
				for name, want := range map[string]string{
					"Ocp-Apim-Subscription-Key": "FAKE-AZURE-KEY",
					"Content-Type":              "application/ssml+xml",
					"Accept":                    "audio/pcm, application/octet-stream",
					"X-Microsoft-OutputFormat":  "raw-24khz-16bit-mono-pcm",
				} {
					if got := r.Header.Get(name); got != want {
						t.Errorf("header %s = %q, want %q", name, got, want)
					}
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					return
				}
				select {
				case requests <- string(body):
				default:
					t.Error("unexpected extra synthesis request")
				}
				w.Header().Set("Content-Type", "audio/pcm; rate=24000; channels=1")
				_, _ = w.Write(contractPCM(1))
			}))
			defer srv.Close()
			t.Setenv("AZURE_SPEECH_KEY", "FAKE-AZURE-KEY")
			t.Setenv("AZURE_SPEECH_ENDPOINT", srv.URL+"/cognitiveservices/v1")
			dir := t.TempDir()
			path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "out.wav")
			if err := os.WriteFile(path, []byte(tc.source), 0o600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			if err := run(context.Background(), []string{"synth", path, "--provider", "azure", "--voice", tc.voice, "-o", out}, strings.NewReader(""), &stdout, &stderr); err != nil {
				t.Fatalf("%v; %s", err, stderr.String())
			}
			if len(requests) != 1 {
				t.Fatalf("request count = %d", len(requests))
			}
			body := <-requests
			want := `<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="https://www.w3.org/2001/mstts" xml:lang="` + tc.locale + `"><voice name="` + tc.voice + `">` + tc.content + `</voice></speak>`
			if body != want {
				t.Fatalf("SSML = %s\nwant = %s", body, want)
			}
			assertAzureNoEmphasisInsideProsody(t, body)
			wav, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if len(wav) != 48 || string(wav[:4]) != "RIFF" || binary.LittleEndian.Uint32(wav[24:28]) != 24000 || !bytes.Equal(wav[44:], contractPCM(1)) {
				t.Fatalf("invalid output WAV: %x", wav)
			}
			if strings.TrimSpace(stdout.String()) != out {
				t.Fatalf("stdout = %q", stdout.String())
			}
		})
	}
}

func TestAzureInvalidControlsFailBeforeRequests(t *testing.T) {
	for _, tc := range []struct{ source, error string }{
		{`[emph strong]Hello.`, "emph"},
		{`[speed 2.1]Hello.`, "speed"},
		{`[pitch +12st]Hello.`, "pitch"},
		{`[volume +21dB]Hello.`, "volume"},
		{`[happy][whisper]Hello.`, "one style"},
		{`[role Boy]Hello.`, "requires a style"},
		{`[act "warmly"]Hello.`, "unsupported tag [act]"},
		{`[voice custom-voice|Hello.]`, "set lang for voice"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			clearProviderDefaults(t)
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
			defer srv.Close()
			t.Setenv("AZURE_SPEECH_KEY", "FAKE-AZURE-KEY")
			t.Setenv("AZURE_SPEECH_ENDPOINT", srv.URL)
			dir := t.TempDir()
			path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "out.wav")
			if err := os.WriteFile(path, []byte("Valid earlier speech.\n\n"+tc.source), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			err := run(context.Background(), []string{"synth", path, "--provider", "azure", "--voice", "en-US-GuyNeural", "-o", out}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), tc.error) || calls.Load() != 0 {
				t.Fatalf("err = %v, requests = %d", err, calls.Load())
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != "KEEP" {
				t.Fatalf("output = %q, err = %v", data, err)
			}
		})
	}
}

func assertAzureNoEmphasisInsideProsody(t *testing.T, body string) {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(body))
	prosodyDepth := 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		switch token := token.(type) {
		case xml.StartElement:
			if token.Name.Local == "emphasis" && prosodyDepth > 0 {
				t.Fatalf("emphasis inside prosody: %s", body)
			}
			if token.Name.Local == "prosody" {
				prosodyDepth++
			}
		case xml.EndElement:
			if token.Name.Local == "prosody" {
				prosodyDepth--
			}
		}
	}
}
