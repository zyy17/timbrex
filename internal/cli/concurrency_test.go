package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func concurrencyVendors() []vendorTest {
	return append(append([]vendorTest{}, vendors...),
		vendorTest{"gemini", "GEMINI_API_KEY", "GEMINI_BASE_URL", "/models/gemini-3.8-flash-tts:generateContent", "x-goog-api-key", "test-key"},
		vendorTest{"fish", "FISH_API_KEY", "FISH_BASE_URL", "/tts", "Authorization", "Bearer test-key"})
}

const concurrencySpeechCount = 10

func concurrencyScript(vendor string, pauses bool) string {
	first, second := "en-US-FirstNeural", "en-US-SecondNeural"
	if vendor == "gemini" {
		first, second = "Kore", "Puck"
	}
	var src strings.Builder
	fmt.Fprintf(&src, "---\nprovider: %s\ncast:\n  a: {voices: {%s: %s}}\n  b: {voices: {%s: %s}}\n---\n", vendor, vendor, first, vendor, second)
	if pauses {
		src.WriteString("a: [pause 1ms]\n")
	}
	for i := 1; i <= concurrencySpeechCount; i++ {
		if i%2 == 1 {
			fmt.Fprintf(&src, "a: [happy] Segment%02d\n", i)
		} else {
			control := "speed 1.2"
			if vendor == "gemini" {
				control = "whisper"
			}
			fmt.Fprintf(&src, "b: [%s|Segment%02d]\n", control, i)
		}
		if pauses && i == 8 {
			src.WriteString("a: [pause 2ms]\n")
		}
	}
	if pauses {
		src.WriteString("b: [pause 3ms]\n")
	}
	return src.String()
}

func concurrencySegment(body []byte) int {
	for i := 1; i <= concurrencySpeechCount; i++ {
		if bytes.Contains(body, fmt.Appendf(nil, "Segment%02d", i)) {
			return i
		}
	}
	return 0
}

// Unequal response lengths expose accidental reuse as well as reordering.
func concurrencyPCM(i int) []byte {
	pcm := make([]byte, 2*(1+i*i))
	for j := 0; j < len(pcm); j += 2 {
		binary.LittleEndian.PutUint16(pcm[j:], uint16(i*100+j/2))
	}
	return pcm
}

func concurrencyWAV(pcm []byte) []byte {
	wav := make([]byte, 44+len(pcm))
	copy(wav, "RIFF")
	binary.LittleEndian.PutUint32(wav[4:], uint32(len(wav)-8))
	copy(wav[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(wav[16:], 16)
	binary.LittleEndian.PutUint16(wav[20:], 1)
	binary.LittleEndian.PutUint16(wav[22:], 1)
	binary.LittleEndian.PutUint32(wav[24:], 24000)
	binary.LittleEndian.PutUint32(wav[28:], 48000)
	binary.LittleEndian.PutUint16(wav[32:], 2)
	binary.LittleEndian.PutUint16(wav[34:], 16)
	copy(wav[36:], "data")
	binary.LittleEndian.PutUint32(wav[40:], uint32(len(pcm)))
	copy(wav[44:], pcm)
	return wav
}

func concurrencyReply(w http.ResponseWriter, vendor, serverURL string, i int) {
	pcm := concurrencyPCM(i)
	switch vendor {
	case "minimax":
		_, _ = fmt.Fprintf(w, `{"base_resp":{"status_code":0},"data":{"status":2,"audio":"%x"}}`, pcm)
	case "qwen":
		_, _ = fmt.Fprintf(w, `{"output":{"finish_reason":"stop","audio":{"url":%q}}}`, fmt.Sprintf("%s/audio-%d.pcm", serverURL, i))
	case "gemini":
		_, _ = fmt.Fprintf(w, `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"inlineData":{"mimeType":"audio/wav","data":%q}}]}}]}`, base64.StdEncoding.EncodeToString(concurrencyWAV(pcm)))
	case "fish":
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write(concurrencyWAV(pcm))
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(pcm)
	}
}

type concurrencyRequest struct {
	Method, Path, Query, Body string
	Header                    http.Header
}

// Serial requests are the reference contract: scheduling must not alter their
// complete contents, authentication, selected voices, controls, or audio format.
func TestConcurrentProviderContracts(t *testing.T) {
	for _, vendor := range concurrencyVendors() {
		t.Run(vendor.id, func(t *testing.T) {
			var reference map[string]concurrencyRequest
			for _, workers := range []int{0, 1, 2, 4, 8} {
				t.Run(fmt.Sprintf("concurrency_%d", workers), func(t *testing.T) {
					requests, wav := runConcurrentContract(t, vendor, workers)
					if workers == 0 {
						reference = requests
					} else if !reflect.DeepEqual(requests, reference) {
						t.Errorf("requests differ from the default serial contract:\ngot: %#v\nwant: %#v", requests, reference)
					}
					want := make([]byte, 48)
					for i := 1; i <= concurrencySpeechCount; i++ {
						want = append(want, concurrencyPCM(i)...)
						if i == 8 {
							want = append(want, make([]byte, 96)...)
						}
					}
					want = append(want, make([]byte, 144)...)
					if !bytes.Equal(wav, concurrencyWAV(want)) {
						t.Errorf("WAV changed: got %d bytes, want %d ordered PCM and pause bytes", len(wav), 44+len(want))
					}
				})
			}
		})
	}
}

func runConcurrentContract(t *testing.T, vendor vendorTest, workers int) (map[string]concurrencyRequest, []byte) {
	t.Helper()
	clearProviderDefaults(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	started, finished := make(chan int, concurrencySpeechCount), make(chan int, concurrencySpeechCount)
	gates := make([]chan struct{}, concurrencySpeechCount+1)
	for i := 1; i <= concurrencySpeechCount; i++ {
		gates[i] = make(chan struct{})
		if workers < 2 || i > workers {
			close(gates[i])
		}
	}
	requests := make(map[string]concurrencyRequest)
	var mu sync.Mutex
	var frontReleased atomic.Bool
	var serverURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		i := concurrencySegment(body)
		if r.Method == http.MethodGet {
			_, err = fmt.Sscanf(r.URL.Path, "/audio-%d.pcm", &i)
			if err != nil || vendor.id != "qwen" || r.Header.Get("Authorization") != "" {
				t.Errorf("unexpected audio download: %s %s", r.Method, r.URL)
			}
		}
		if i < 1 || i > concurrencySpeechCount {
			t.Errorf("unrecognized speech request: %s", body)
			return
		}
		if r.Method == http.MethodPost && r.Header.Get(vendor.authHeader) != vendor.authValue {
			t.Error("wrong authentication header")
		}
		key := fmt.Sprintf("%s/%d", r.Method, i)
		mu.Lock()
		if _, exists := requests[key]; exists {
			t.Errorf("duplicate request %s", key)
		}
		requests[key] = concurrencyRequest{r.Method, r.URL.Path, r.URL.RawQuery, string(body), r.Header.Clone()}
		mu.Unlock()
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(concurrencyPCM(i))
			return
		}
		if workers > 1 && i > workers && !frontReleased.Load() {
			t.Errorf("segment %d started while the first bounded window was uncommitted", i)
		}
		started <- i
		select {
		case <-gates[i]:
		case <-r.Context().Done():
			return
		}
		concurrencyReply(w, vendor.id, serverURL, i)
		finished <- i
	}))
	defer srv.Close()
	serverURL = srv.URL
	t.Setenv(vendor.keyEnv, "test-key")
	t.Setenv(vendor.endpointEnv, srv.URL)
	path, out := concurrencyFiles(t, concurrencyScript(vendor.id, true))
	args := []string{"synth", path, "-o", out}
	if workers != 0 {
		args = append(args, "--concurrency", strconv.Itoa(workers))
	}
	done := make(chan error, 1)
	go func() { done <- run(ctx, args, nil, &bytes.Buffer{}, &bytes.Buffer{}) }()
	if workers > 1 {
		seen := make(map[int]bool)
		for range workers {
			select {
			case i := <-started:
				if i > workers || seen[i] {
					t.Fatalf("unexpected initial request %d with concurrency %d", i, workers)
				}
				seen[i] = true
			case err := <-done:
				t.Fatalf("synthesis finished before concurrent barrier: %v", err)
			case <-ctx.Done():
				t.Fatal("concurrent requests did not reach barrier")
			}
		}
		// Force distinct server completion order without timing-based sleeps.
		for i := workers; i >= 1; i-- {
			if i == 1 {
				frontReleased.Store(true)
			}
			close(gates[i])
			select {
			case got := <-finished:
				if got != i {
					t.Errorf("response order = %d, want %d", got, i)
				}
			case <-ctx.Done():
				t.Fatal("response did not finish")
			}
		}
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("synthesis did not finish")
	}
	// Close waits for handlers before inspecting their counters or snapshots.
	srv.Close()
	wantCalls := concurrencySpeechCount
	if vendor.id == "qwen" {
		wantCalls *= 2
	}
	if len(requests) != wantCalls {
		t.Errorf("requests=%d, want %d", len(requests), wantCalls)
	}
	wav, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return requests, wav
}

func concurrencyFiles(t *testing.T, source string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	path, out := filepath.Join(dir, "script.tbx"), filepath.Join(dir, "out.wav")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, []byte("KEEP"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path, out
}

func assertConcurrentOutputPreserved(t *testing.T, out string) {
	t.Helper()
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "KEEP" {
		t.Errorf("output changed: %q, error=%v", data, err)
	}
	partials, err := filepath.Glob(filepath.Join(filepath.Dir(out), ".timbrex-*.partial"))
	if err != nil || len(partials) != 0 {
		t.Errorf("partial output remains: %v, error=%v", partials, err)
	}
}

func TestConcurrentProvidersPreflightWholeScript(t *testing.T) {
	for _, vendor := range concurrencyVendors() {
		t.Run(vendor.id, func(t *testing.T) {
			clearProviderDefaults(t)
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
			defer srv.Close()
			t.Setenv(vendor.keyEnv, "test-key")
			t.Setenv(vendor.endpointEnv, srv.URL)
			path, out := concurrencyFiles(t, concurrencyScript(vendor.id, true)+"b: Later [sfx missing.wav]\n")
			err := run(context.Background(), []string{"synth", path, "-o", out, "--concurrency", "8"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), "sfx") || calls.Load() != 0 {
				t.Errorf("error=%v, HTTP requests=%d", err, calls.Load())
			}
			assertConcurrentOutputPreserved(t, out)
		})
	}
}

type concurrencyTransport struct {
	base           http.RoundTripper
	opened, closed atomic.Int32
	ready          chan struct{}
}

func (tr *concurrencyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	res, err := tr.base.RoundTrip(r)
	if err == nil {
		tr.opened.Add(1)
		res.Body = &concurrencyBody{ReadCloser: res.Body, closed: &tr.closed}
		tr.ready <- struct{}{}
	}
	return res, err
}

type concurrencyBody struct {
	io.ReadCloser
	closed *atomic.Int32
	once   sync.Once
}

func (body *concurrencyBody) Close() error {
	body.once.Do(func() { body.closed.Add(1) })
	return body.ReadCloser.Close()
}

func TestConcurrentProviderFailureCancelsInflightBodies(t *testing.T) {
	for _, vendor := range concurrencyVendors() {
		t.Run(vendor.id, func(t *testing.T) {
			clearProviderDefaults(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			original := http.DefaultTransport
			transport := &concurrencyTransport{base: original, ready: make(chan struct{}, 8)}
			http.DefaultTransport = transport
			t.Cleanup(func() { http.DefaultTransport = original })
			streaming, ended := make(chan int, 3), make(chan int, 3)
			fail := make(chan struct{})
			var posts, downloads atomic.Int32
			var serverURL string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					return
				}
				i := concurrencySegment(body)
				if r.Method == http.MethodGet {
					downloads.Add(1)
					if _, err := fmt.Sscanf(r.URL.Path, "/audio-%d.pcm", &i); err != nil {
						t.Error(err)
					}
				} else {
					posts.Add(1)
				}
				if i < 1 || i > 4 {
					t.Errorf("request %d admitted beyond the failing initial window", i)
					http.Error(w, "unexpected request", http.StatusInternalServerError)
					return
				}
				if i == 2 {
					select {
					case <-fail:
						http.Error(w, "offline failure", http.StatusServiceUnavailable)
					case <-r.Context().Done():
					}
					return
				}
				if vendor.id == "qwen" && r.Method == http.MethodPost {
					concurrencyReply(w, vendor.id, serverURL, i)
					return
				}
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write([]byte{1, 0})
				w.(http.Flusher).Flush()
				streaming <- i
				<-r.Context().Done()
				ended <- i
			}))
			defer srv.Close()
			serverURL = srv.URL
			t.Setenv(vendor.keyEnv, "test-key")
			t.Setenv(vendor.endpointEnv, srv.URL)
			path, out := concurrencyFiles(t, concurrencyScript(vendor.id, false))
			done := make(chan error, 1)
			go func() {
				done <- run(ctx, []string{"synth", path, "-o", out, "--concurrency", "4"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
			}()
			for range 3 {
				select {
				case <-streaming:
				case err := <-done:
					t.Fatalf("finished before all response bodies opened: %v", err)
				case <-ctx.Done():
					t.Fatal("in-flight response bodies did not open")
				}
			}
			wantDownloads := int32(0)
			if vendor.id == "qwen" {
				wantDownloads = 3
			}
			// A server Flush alone does not prove the client received headers.
			// Wait for actual response bodies before inducing cancellation.
			for range 3 + wantDownloads {
				select {
				case <-transport.ready:
				case <-ctx.Done():
					t.Fatal("client response bodies did not open")
				}
			}
			close(fail)
			select {
			case err := <-done:
				if err == nil || !strings.Contains(err.Error(), "HTTP 503") {
					t.Errorf("original failure lost: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("failed synthesis did not return")
			}
			for range 3 {
				select {
				case <-ended:
				case <-ctx.Done():
					t.Fatal("canceled HTTP handler did not exit")
				}
			}
			srv.Close()
			if posts.Load() != 4 || downloads.Load() != wantDownloads {
				t.Errorf("posts=%d downloads=%d, want 4 and %d", posts.Load(), downloads.Load(), wantDownloads)
			}
			if transport.opened.Load() != 4+wantDownloads || transport.closed.Load() != transport.opened.Load() {
				t.Errorf("response bodies opened=%d closed=%d", transport.opened.Load(), transport.closed.Load())
			}
			assertConcurrentOutputPreserved(t, out)
		})
	}
}

func TestSynthConcurrencyFlagValidation(t *testing.T) {
	synth, _, err := newCommand().Find([]string{"synth"})
	if err != nil {
		t.Fatal(err)
	}
	flag := synth.Flags().Lookup("concurrency")
	if flag == nil || flag.DefValue != "1" {
		t.Fatalf("default concurrency is not 1: %v", flag)
	}
	for _, value := range []string{"-1", "0", "9", "9999999999999999999999999", "many"} {
		t.Run(value, func(t *testing.T) {
			missing := filepath.Join(t.TempDir(), "missing.tbx")
			err := run(context.Background(), []string{"synth", missing, "--concurrency", value}, nil, &bytes.Buffer{}, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), "concurrency") || strings.Contains(err.Error(), "no such file") {
				t.Errorf("invalid concurrency did not fail before reading source: %v", err)
			}
		})
	}
	path, out := concurrencyFiles(t, "Hello\n")
	err = run(context.Background(), []string{"synth", path, "-o", out, "--provider", "stub", "--concurrency", "2"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "concurrency") {
		t.Errorf("stub accepted concurrency: %v", err)
	}
	assertConcurrentOutputPreserved(t, out)
}
