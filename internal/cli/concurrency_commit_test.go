package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
)

func TestConcurrentCancellationAfterPublicationStillSucceeds(t *testing.T) {
	for _, vendor := range concurrencyVendors() {
		t.Run(vendor.id, func(t *testing.T) {
			clearProviderDefaults(t)
			var calls atomic.Int32
			var serverURL string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method == http.MethodGet {
					var index int
					if _, err := fmt.Sscanf(r.URL.Path, "/audio-%d.pcm", &index); err != nil {
						t.Error(err)
						return
					}
					w.Header().Set("Content-Type", "application/octet-stream")
					_, _ = w.Write(concurrencyPCM(index))
					return
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					return
				}
				concurrencyReply(w, vendor.id, serverURL, concurrencySegment(body))
			}))
			defer srv.Close()
			serverURL = srv.URL
			t.Setenv(vendor.keyEnv, "test-key")
			t.Setenv(vendor.endpointEnv, srv.URL)
			path, out := concurrencyFiles(t, concurrencyScript(vendor.id, false))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			// The CLI prints the result only after the complete WAV is published.
			err := run(ctx, []string{"synth", path, "-o", out, "--concurrency", "4"}, nil, cancelWriter{cancel}, &bytes.Buffer{})
			if err != nil || !errors.Is(ctx.Err(), context.Canceled) {
				t.Fatalf("committed synthesis: error=%v context=%v", err, ctx.Err())
			}
			var pcm []byte
			for i := 1; i <= concurrencySpeechCount; i++ {
				pcm = append(pcm, concurrencyPCM(i)...)
			}
			wav, err := os.ReadFile(out)
			if err != nil || !bytes.Equal(wav, concurrencyWAV(pcm)) {
				t.Fatalf("committed WAV changed after cancellation: bytes=%d error=%v", len(wav), err)
			}
			wantCalls := int32(concurrencySpeechCount)
			if vendor.id == "qwen" {
				wantCalls *= 2
			}
			if calls.Load() != wantCalls {
				t.Fatalf("requests=%d, want %d", calls.Load(), wantCalls)
			}
		})
	}
}
