package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func TestConfiguredEndpointDelimitersFailBeforeRequests(t *testing.T) {
	for _, vendor := range concurrencyVendors() {
		for _, suffix := range []string{"?", "#", "?#"} {
			t.Run(vendor.id+"/"+suffix, func(t *testing.T) {
				var calls atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
				defer srv.Close()
				t.Setenv(vendor.keyEnv, "test-key")
				t.Setenv(vendor.endpointEnv, srv.URL+"/proxy"+suffix)
				path, out := concurrencyFiles(t, "Hello.")
				var stdout bytes.Buffer
				err := run(context.Background(), []string{"synth", path, "--provider", vendor.id, "-o", out}, nil, &stdout, &bytes.Buffer{})
				if err == nil || !strings.Contains(err.Error(), "endpoint") || calls.Load() != 0 || stdout.Len() != 0 {
					t.Fatalf("endpoint did not fail locally: error=%v requests=%d stdout=%q", err, calls.Load(), stdout.String())
				}
				assertConcurrentOutputPreserved(t, out)
			})
		}
	}
}

func TestConfiguredEscapedEndpointPathsReachProviders(t *testing.T) {
	for _, vendor := range concurrencyVendors() {
		t.Run(vendor.id, func(t *testing.T) {
			clearProviderDefaults(t)
			const prefix = "/proxy%3Froute%23"
			voice := "en-US-GuyNeural"
			if vendor.id == "gemini" {
				voice = "Kore"
			}
			wantPath := prefix + vendor.path
			switch vendor.id {
			case "azure", "qwen":
				wantPath = prefix
			case "elevenlabs":
				wantPath += voice
			}
			var posts, downloads atomic.Int32
			var serverURL string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					downloads.Add(1)
					if vendor.id != "qwen" || r.URL.Path != "/audio-1.pcm" {
						t.Errorf("unexpected audio download: %s", r.URL)
					}
					w.Header().Set("Content-Type", "audio/pcm")
					_, _ = w.Write(concurrencyPCM(1))
					return
				}
				posts.Add(1)
				wantQuery := ""
				if vendor.id == "elevenlabs" {
					wantQuery = "output_format=pcm_24000"
				}
				if r.Method != http.MethodPost || r.URL.EscapedPath() != wantPath || r.URL.RawQuery != wantQuery || r.Header.Get(vendor.authHeader) != vendor.authValue {
					t.Errorf("escaped endpoint routing changed: %s %s", r.Method, r.URL)
				}
				concurrencyReply(w, vendor.id, serverURL, 1)
			}))
			defer srv.Close()
			serverURL = srv.URL
			t.Setenv(vendor.keyEnv, "test-key")
			t.Setenv(vendor.endpointEnv, srv.URL+prefix)
			path, out := concurrencyFiles(t, "Hello.")
			err := run(context.Background(), []string{"synth", path, "--provider", vendor.id, "--voice", voice, "-o", out}, nil, &bytes.Buffer{}, &bytes.Buffer{})
			if err != nil || posts.Load() != 1 {
				t.Fatalf("escaped endpoint rejected: error=%v POST requests=%d", err, posts.Load())
			}
			wantDownloads := int32(0)
			if vendor.id == "qwen" {
				wantDownloads = 1
			}
			if downloads.Load() != wantDownloads {
				t.Fatalf("audio downloads=%d, want %d", downloads.Load(), wantDownloads)
			}
			wav, err := os.ReadFile(out)
			if err != nil || !bytes.Equal(wav, concurrencyWAV(concurrencyPCM(1))) {
				t.Fatalf("escaped endpoint output changed: error=%v bytes=%d", err, len(wav))
			}
		})
	}
}
