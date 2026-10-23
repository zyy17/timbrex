package openai

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/lower"
	"github.com/zyy17/timbrex/internal/parse"
	"github.com/zyy17/timbrex/internal/providers"
)

// BenchmarkSynthesizeConcurrency measures local HTTP scheduling, decoding, and
// atomic WAV output, not vendor latency, rate limits, price, or speech quality.
// Run: go test ./internal/providers/openai -run '^$' -bench BenchmarkSynthesizeConcurrency -benchmem -benchtime=10x -count=3
func BenchmarkSynthesizeConcurrency(b *testing.B) {
	const segments = 24
	var source strings.Builder
	for i := range segments {
		fmt.Fprintf(&source, "Segment%02d\n\n", i)
	}
	script, err := parse.Parse("benchmark.tbx", source.String())
	if err != nil {
		b.Fatal(err)
	}
	if err := check.Failed(check.Apply(script, false)); err != nil {
		b.Fatal(err)
	}
	doc := lower.Script(script)
	p := provider{}
	for _, fixture := range []struct {
		name  string
		delay func(int) time.Duration
	}{
		{"no_delay", func(int) time.Duration { return 0 }},
		{"fixed_2ms", func(int) time.Duration { return 2 * time.Millisecond }},
		{"mixed_0_4ms", func(i int) time.Duration { return time.Duration(i%5) * time.Millisecond }},
	} {
		b.Run(fixture.name, func(b *testing.B) {
			for _, workers := range []int{0, 1, 2, 4, 8} {
				name := fmt.Sprintf("concurrency_%d", workers)
				if workers == 0 {
					name = "default_serial"
				}
				b.Run(name, func(b *testing.B) {
					var calls atomic.Int64
					payloads := make([][]byte, segments)
					var audioBytes int
					for i := range payloads {
						payloads[i] = make([]byte, 2*(64+(i%4)*512))
						for j := 0; j < len(payloads[i]); j += 2 {
							binary.LittleEndian.PutUint16(payloads[i][j:], uint16(i*100+j/2))
						}
						audioBytes += len(payloads[i])
					}
					srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						var request struct{ Input string }
						if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
							b.Error(err)
							http.Error(w, "bad benchmark request", http.StatusBadRequest)
							return
						}
						i, err := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(request.Input), "Segment"))
						if err != nil || i < 0 || i >= segments {
							b.Errorf("invalid benchmark segment: %q", request.Input)
							http.Error(w, "bad benchmark segment", http.StatusBadRequest)
							return
						}
						if delay := fixture.delay(i); delay > 0 {
							timer := time.NewTimer(delay)
							defer timer.Stop()
							select {
							case <-timer.C:
							case <-r.Context().Done():
								return
							}
						}
						w.Header().Set("Content-Type", "application/octet-stream")
						_, _ = w.Write(payloads[i])
					}))
					defer srv.Close()
					opt := providers.Options{
						Out:   filepath.Join(b.TempDir(), "benchmark.wav"),
						Model: "gpt-4o-mini-tts", Voice: "coral", APIKey: "offline-test-key", Endpoint: srv.URL,
						Concurrency: workers,
					}
					ctx := context.Background()
					// Warm the HTTP connection and output path equally for each case.
					if _, err := p.Synthesize(ctx, doc, opt); err != nil {
						b.Fatal(err)
					}
					calls.Store(0)
					b.ReportAllocs()
					b.SetBytes(int64(audioBytes))
					b.ResetTimer()
					for range b.N {
						if _, err := p.Synthesize(ctx, doc, opt); err != nil {
							b.Fatal(err)
						}
					}
					b.StopTimer()
					if calls.Load() != int64(b.N)*segments {
						b.Fatalf("HTTP requests=%d, want %d", calls.Load(), b.N*segments)
					}
					b.ReportMetric(float64(calls.Load())/float64(b.N), "requests/op")
					b.ReportMetric(float64(audioBytes), "audio_bytes/op")
				})
			}
		})
	}
}
