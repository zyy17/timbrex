package azure

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

func TestDeploymentEndpointEncoding(t *testing.T) {
	id := "opaque / ? & % + # 部署"
	const base = "https://eastus.voice.speech.microsoft.com/cognitiveservices/v1"
	opt := providers.Options{Endpoint: base, Synthesis: options.Azure{DeploymentID: &id}}
	endpoint, err := synthesisEndpoint(opt)
	if err != nil {
		t.Fatal(err)
	}
	want := base + "?deploymentId=opaque+%2F+%3F+%26+%25+%2B+%23+%E9%83%A8%E7%BD%B2"
	if endpoint != want || opt.Endpoint != base || *opt.Synthesis.(options.Azure).DeploymentID != id {
		t.Fatalf("encoding or mutation error: %s", endpoint)
	}
	u, err := url.Parse(endpoint)
	if err != nil || len(u.Query()) != 1 || len(u.Query()["deploymentId"]) != 1 || u.Query().Get("deploymentId") != id {
		t.Fatalf("query did not preserve one opaque value: %v", err)
	}
}

func TestNoDeploymentPreservesEndpoint(t *testing.T) {
	for _, endpoint := range []string{"", "https://eastus.tts.speech.microsoft.com/cognitiveservices/v1", "https://proxy.example/speech?existing=query"} {
		for _, value := range []options.Value{nil, options.Azure{}} {
			got, err := synthesisEndpoint(providers.Options{Endpoint: endpoint, Synthesis: value})
			if err != nil || got != endpoint {
				t.Fatalf("no-deployment endpoint changed: %v", err)
			}
		}
	}
}

func TestNoDeploymentPreservesWire(t *testing.T) {
	const endpoint = "https://eastus.tts.speech.microsoft.com/cognitiveservices/v1"
	const body = `<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="https://www.w3.org/2001/mstts" xml:lang="en-US"><voice name="en-US-JennyNeural">Hello</voice></speak>`
	for _, value := range []options.Value{nil, options.Azure{}} {
		var calls int
		p := provider{client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			got, err := io.ReadAll(r.Body)
			if err != nil || string(got) != body || r.URL.String() != endpoint || r.Method != http.MethodPost {
				t.Errorf("ordinary synthesis request changed: %v", err)
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("\x00\x00"))}, nil
		})}}
		doc := &ir.Document{Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}}}
		opt := providers.Options{Out: filepath.Join(t.TempDir(), "out.wav"), Endpoint: endpoint, Voice: "en-US-JennyNeural", Synthesis: value}
		if _, err := p.Synthesize(context.Background(), doc, opt); err != nil || calls != 1 {
			t.Fatalf("ordinary synthesis failed: calls=%d err=%v", calls, err)
		}
	}
}

func TestDeploymentFullWire(t *testing.T) {
	for _, concurrency := range []int{1, 8} {
		t.Run(fmt.Sprint(concurrency), func(t *testing.T) {
			id := "opaque / ? & % + # 部署"
			const voice = `MyCustom"Voice&<>`
			const key = "FAKE-SECRET-DO-NOT-LOG"
			const text = `Hello & <goodbye>.`
			const expectedSSML = `<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="https://www.w3.org/2001/mstts" xml:lang="en-US"><voice name="MyCustom&#34;Voice&amp;&lt;&gt;">Hello &amp; &lt;goodbye&gt;.</voice></speak>`
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodPost || r.URL.EscapedPath() != "/proxy/speech%20route" || len(r.URL.Query()) != 1 || len(r.URL.Query()["deploymentId"]) != 1 || r.URL.Query().Get("deploymentId") != id {
					t.Error("wrong synthesis method, path, or deployment routing")
				}
				for name, want := range map[string]string{
					"Ocp-Apim-Subscription-Key": key,
					"Content-Type":              "application/ssml+xml",
					"Accept":                    "audio/pcm, application/octet-stream",
					"X-Microsoft-OutputFormat":  "raw-24khz-16bit-mono-pcm",
					"User-Agent":                "timbrex",
				} {
					if r.Header.Get(name) != want {
						t.Errorf("changed %s header", name)
					}
				}
				if r.Header.Get("Authorization") != "" || strings.Contains(r.URL.String(), key) {
					t.Error("changed authentication or leaked a key into the URL")
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if string(body) != expectedSSML {
					t.Errorf("unexpected complete SSML: %s", body)
				}
				var ssml struct {
					XMLName xml.Name `xml:"http://www.w3.org/2001/10/synthesis speak"`
					Lang    string   `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
					Voice   struct {
						Name     string `xml:"name,attr"`
						Text     string `xml:",chardata"`
						Children []any  `xml:",any"`
					} `xml:"voice"`
				}
				if err := xml.Unmarshal(body, &ssml); err != nil || ssml.Lang != "en-US" || ssml.Voice.Name != voice || ssml.Voice.Text != text || len(ssml.Voice.Children) != 0 {
					t.Errorf("custom voice SSML changed: %v", err)
				}
				if strings.Contains(string(body), id) || strings.Contains(string(body), key) {
					t.Error("deployment routing or key leaked into SSML")
				}
				w.Header().Set("Content-Type", "audio/pcm")
				_, _ = w.Write([]byte{0, 0})
			}))
			defer server.Close()
			out := filepath.Join(t.TempDir(), "out.wav")
			doc := &ir.Document{Lang: "en-US", Utterances: []ir.Utterance{
				{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: text}}},
				{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: text}}},
			}}
			opt := providers.Options{Out: out, Voice: voice, APIKey: key, Endpoint: server.URL + "/proxy/speech%20route", Concurrency: concurrency, Synthesis: options.Azure{DeploymentID: &id}}
			if _, err := (provider{}).Synthesize(context.Background(), doc, opt); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(out)
			if err != nil || len(data) != 48 || string(data[:4]) != "RIFF" || calls.Load() != 2 {
				t.Fatalf("output or request count changed: size=%d calls=%d err=%v", len(data), calls.Load(), err)
			}
		})
	}
}

func TestDeploymentEndpointSnapshot(t *testing.T) {
	const original = "original & opaque deployment"
	for _, concurrency := range []int{1, 8} {
		t.Run(fmt.Sprint(concurrency), func(t *testing.T) {
			id := original
			var once sync.Once
			var calls atomic.Int32
			p := provider{client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				once.Do(func() { id = "changed after first request began" })
				calls.Add(1)
				if r.URL.Query().Get("deploymentId") != original {
					t.Error("in-flight synthesis reread the mutable deployment ID")
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("\x00\x00"))}, nil
			})}}
			doc := &ir.Document{Lang: "en-US", Utterances: []ir.Utterance{
				{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "First"}}},
				{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Second"}}},
			}}
			opt := providers.Options{Out: filepath.Join(t.TempDir(), "out.wav"), Endpoint: "https://eastus.voice.speech.microsoft.com/cognitiveservices/v1", Voice: "MyCustomVoice", Concurrency: concurrency, Synthesis: options.Azure{DeploymentID: &id}}
			if _, err := p.Synthesize(context.Background(), doc, opt); err != nil || calls.Load() != 2 || id == original {
				t.Fatalf("snapshot fixture failed: calls=%d err=%v", calls.Load(), err)
			}
		})
	}
}

func TestDeploymentHTTPFailureKeepsSecretsOutOfErrors(t *testing.T) {
	const secret = "FAKE-SECRET-DO-NOT-LOG"
	id := secret
	p := provider{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(secret))}, nil
	})}}
	out := filepath.Join(t.TempDir(), "out.wav")
	if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
		t.Fatal(err)
	}
	doc := &ir.Document{Lang: "en-US", Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}}}
	opt := providers.Options{Out: out, Endpoint: "https://eastus.voice.speech.microsoft.com/cognitiveservices/v1", Voice: "MyCustomVoice", APIKey: secret, Synthesis: options.Azure{DeploymentID: &id}}
	if _, err := p.Synthesize(context.Background(), doc, opt); err == nil || strings.Contains(err.Error(), secret) || !strings.Contains(err.Error(), "HTTP 400") {
		t.Fatalf("unexpected or unsafe HTTP failure: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "KEEP" {
		t.Fatal("HTTP failure changed output")
	}
}

func TestDeploymentTransportFailureRedactsURL(t *testing.T) {
	const secret = "FAKE-SECRET-DO-NOT-LOG"
	id := secret
	failure := errors.New("synthetic transport failure")
	var calls int
	p := provider{client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if !strings.Contains(r.URL.String(), secret) || r.Header.Get("Ocp-Apim-Subscription-Key") != secret {
			t.Fatal("fixture did not exercise a sensitive request URL and key")
		}
		// http.Client wraps this in url.Error, whose URL includes deploymentId.
		return nil, failure
	})}}
	dir := t.TempDir()
	out := filepath.Join(dir, "out.wav")
	if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
		t.Fatal(err)
	}
	doc := &ir.Document{Lang: "en-US", Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}}}
	opt := providers.Options{Out: out, Endpoint: "https://eastus.voice.speech.microsoft.com/cognitiveservices/v1", Voice: "MyCustomVoice", APIKey: secret, Synthesis: options.Azure{DeploymentID: &id}}
	if _, err := p.Synthesize(context.Background(), doc, opt); !errors.Is(err, failure) || strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "deploymentId") || calls != 1 {
		t.Fatalf("transport failure was changed or leaked the URL: calls=%d err=%v", calls, err)
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "KEEP" {
		t.Fatal("transport failure changed output")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatal("transport failure left temporary files")
	}
}

func TestDeploymentRejectsUnsafeRoutingBeforeRequests(t *testing.T) {
	const secret = "secret-value"
	id := "existing"
	for _, endpoint := range []string{
		"", "relative/path", "file:///" + secret, "https://user:" + secret + "@example.com/speech",
		"https://example.com?deploymentId=existing", "https://example.com?deploymentId=" + secret,
		"https://example.com?deploymentId=existing&deploymentId=existing",
		"https://example.com?deploymentId=existing&deploymentId=" + secret,
		"https://example.com?%64eploymentId=existing", "https://example.com?key=" + secret,
		"https://example.com?", "https://example.com#" + secret, "https://example.com#",
		"https://example.com/%zz/" + secret, "https://eastus.tts.speech.microsoft.com/cognitiveservices/v1",
		"https://EASTUS.TTS.SPEECH.MICROSOFT.COM.:443/cognitiveservices/v1",
		"https://usgovvirginia.tts.speech.azure.us/cognitiveservices/v1",
		"https://chinanorth3.tts.speech.azure.cn/cognitiveservices/v1",
	} {
		for _, concurrency := range []int{1, 8} {
			t.Run(fmt.Sprintf("%s/%d", endpoint, concurrency), func(t *testing.T) {
				opt := providers.Options{Endpoint: endpoint, Voice: "en-US-JennyNeural", APIKey: secret, Concurrency: concurrency, Synthesis: options.Azure{DeploymentID: &id}}
				assertDeploymentRejected(t, &ir.Document{Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}}}, opt, secret)
			})
		}
	}
	for _, invalidID := range []string{"", "  ", secret + "\n", secret + "\x00", secret + "\xff"} {
		opt := providers.Options{Endpoint: "https://example.com/speech", Voice: "en-US-JennyNeural", Synthesis: options.Azure{DeploymentID: &invalidID}}
		assertDeploymentRejected(t, &ir.Document{Utterances: []ir.Utterance{{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Hello"}}}}}, opt, secret)
	}
}

func TestDeploymentRejectsInvalidLaterSegmentsBeforeRequests(t *testing.T) {
	id := "existing"
	for _, concurrency := range []int{1, 8} {
		for _, bad := range []ir.Utterance{
			{Speaker: "custom", Nodes: []ir.Node{{Kind: ir.Text, Text: "Missing custom voice language"}}},
			{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.State, Name: "act", Args: []ir.Arg{{Raw: "unsupported"}}}, {Kind: ir.Text, Text: "Unsupported later control"}}},
		} {
			doc := &ir.Document{Voices: map[string]map[string]string{"custom": {"azure": "MyCustomVoice"}}, Utterances: []ir.Utterance{
				{Speaker: "narrator", Nodes: []ir.Node{{Kind: ir.Text, Text: "Valid first segment"}}}, bad,
			}}
			opt := providers.Options{Endpoint: "https://eastus.voice.speech.microsoft.com/cognitiveservices/v1", Voice: "en-US-JennyNeural", Concurrency: concurrency, Synthesis: options.Azure{DeploymentID: &id}}
			assertDeploymentRejected(t, doc, opt, "secret-value")
		}
	}
}

func assertDeploymentRejected(t *testing.T, doc *ir.Document, opt providers.Options, secret string) {
	t.Helper()
	for _, exists := range []bool{false, true} {
		var calls atomic.Int32
		p := provider{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("\x00\x00"))}, nil
		})}}
		dir := t.TempDir()
		opt.Out = filepath.Join(dir, "out.wav")
		if exists {
			if err := os.WriteFile(opt.Out, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		_, err := p.Synthesize(context.Background(), doc, opt)
		if err == nil || calls.Load() != 0 || strings.Contains(err.Error(), secret) {
			t.Fatalf("invalid input accepted, requested, or leaked: calls=%d err=%v", calls.Load(), err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if exists {
			data, err := os.ReadFile(opt.Out)
			if err != nil || string(data) != "KEEP" || len(entries) != 1 {
				t.Fatal("rejected synthesis altered output or left temporary files")
			}
		} else if len(entries) != 0 {
			t.Fatal("rejected synthesis created output or temporary files")
		}
	}
}
