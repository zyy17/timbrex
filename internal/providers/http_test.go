package providers

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestHTTPStatusAndRedirects(t *testing.T) {
	for _, status := range []int{301, 401, 429, 500} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "https://example.invalid/secret")
			w.WriteHeader(status)
			_, _ = io.WriteString(w, "secret response body")
		}))
		_, err := Request(context.Background(), nil, http.MethodPost, srv.URL, "", nil, nil, ResponseBody)
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), "HTTP") || strings.Contains(err.Error(), "secret") {
			t.Fatalf("status %d = %v", status, err)
		}
	}
}

func TestHTTPAudioContentType(t *testing.T) {
	for _, tt := range []struct {
		contentType string
		valid       bool
	}{
		{"audio/pcm", true},
		{"audio/x-pcm", true},
		{"audio/raw", true},
		{"application/octet-stream", true},
		{"Audio/PCM; rate=24000", true},
		{"application/json", false},
		{"audio/mpeg", false},
		{"audio/wav", false},
		{"audio/L16", false},
		{"application/octet-stream-invalid", false},
		{"audio/pcm; malformed", false},
	} {
		t.Run(tt.contentType, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				_, _ = w.Write([]byte{1, 0, 2, 0})
			}))
			defer srv.Close()
			_, err := Request(context.Background(), nil, http.MethodGet, srv.URL, "", map[string]string{"Accept": "audio/*"}, nil, ResponsePCM)
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%t, err=%v", tt.valid, err)
			}
		})
	}
}

func TestHTTPCancellation(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { <-started; cancel() }()
	_, err := Request(ctx, nil, http.MethodGet, srv.URL, "", nil, nil, ResponseBody)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestPCMContentTypeParameters(t *testing.T) {
	for _, tt := range []struct {
		header string
		valid  bool
	}{
		{"", true},
		{"audio/pcm", true},
		{"application/octet-stream", true},
		{"audio/pcm; rate=24000; channels=1", true},
		{"audio/pcm; rate=24000", true},
		{"audio/pcm; channels=1", true},
		{"audio/x-pcm; rate=16000", false},
		{"audio/pcm; rate=48000", false},
		{"audio/raw; channels=2", false},
		{"application/octet-stream; rate=44100; channels=2", false},
		{"audio/pcm; rate=unknown", false},
	} {
		t.Run(tt.header, func(t *testing.T) {
			err := checkPCMContentType(tt.header)
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%t, error=%v", tt.valid, err)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type trackedBody struct {
	io.Reader
	read      int
	readCalls int
	closed    bool
	closeErr  error
}

func (b *trackedBody) Read(p []byte) (int, error) {
	b.readCalls++
	n, err := b.Reader.Read(p)
	b.read += n
	return n, err
}

func (b *trackedBody) Close() error {
	b.closed = true
	if closer, ok := b.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return b.closeErr
}

// Generate the boundary fixtures without allocating a second 64 MiB buffer.
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestHTTPResponseSizeBoundary(t *testing.T) {
	for _, tt := range []struct {
		name          string
		size          int
		contentLength int64
		wantErr       string
	}{
		{"empty", 0, 0, "empty response"},
		{"known exact limit", MaxResponseBytes, MaxResponseBytes, ""},
		{"unknown over limit", MaxResponseBytes + 1, -1, fmt.Sprintf("response exceeds %d bytes", MaxResponseBytes)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := &trackedBody{Reader: io.LimitReader(zeroReader{}, int64(tt.size))}
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: tt.contentLength}, nil
			})}
			data, err := Request(context.Background(), client, http.MethodGet, "https://example.invalid/audio", "", nil, nil, ResponseBody)
			if !body.closed || body.read != min(tt.size, MaxResponseBytes+1) {
				t.Fatalf("closed=%t read=%d, want %d", body.closed, body.read, min(tt.size, MaxResponseBytes+1))
			}
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr || data != nil {
					t.Fatalf("bytes=%d error=%v, want nil data and %q", len(data), err, tt.wantErr)
				}
				return
			}
			if err != nil || len(data) != tt.size {
				t.Fatalf("bytes=%d error=%v", len(data), err)
			}
		})
	}
}

func TestHTTPRejectsDeclaredOversizeBeforeRead(t *testing.T) {
	body := &trackedBody{Reader: zeroReader{}}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: MaxResponseBytes + 8192}, nil
	})}
	data, err := Request(context.Background(), client, http.MethodGet, "https://example.invalid/audio", "", nil, nil, ResponseBody)
	wantErr := fmt.Sprintf("response exceeds %d bytes", MaxResponseBytes)
	if err == nil || err.Error() != wantErr || data != nil || !body.closed || body.readCalls != 0 {
		t.Fatalf("bytes=%d error=%v closed=%t read=%d calls=%d, want nil data, %q, closed body, and no reads", len(data), err, body.closed, body.read, body.readCalls, wantErr)
	}
}

func TestHTTPStreamedResponseSizeBoundary(t *testing.T) {
	for _, encoding := range []string{"chunked", "gzip"} {
		t.Run(encoding, func(t *testing.T) {
			var compressed bytes.Buffer
			if encoding == "gzip" {
				zw := gzip.NewWriter(&compressed)
				if _, err := io.CopyN(zw, zeroReader{}, MaxResponseBytes+4096); err != nil {
					t.Fatal(err)
				}
				if err := zw.Close(); err != nil {
					t.Fatal(err)
				}
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if encoding == "gzip" {
					w.Header().Set("Content-Encoding", "gzip")
					w.Header().Set("Content-Length", strconv.Itoa(compressed.Len()))
					_, _ = w.Write(compressed.Bytes())
					return
				}
				w.(http.Flusher).Flush()
				_, _ = io.CopyN(w, zeroReader{}, MaxResponseBytes+4096)
			}))
			defer srv.Close()
			transport := http.DefaultTransport.(*http.Transport).Clone()
			defer transport.CloseIdleConnections()
			var body *trackedBody
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				res, err := transport.RoundTrip(req)
				if err != nil {
					return nil, err
				}
				if res.ContentLength != -1 {
					t.Errorf("length=%d, want unknown length", res.ContentLength)
				}
				if encoding == "gzip" && !res.Uncompressed {
					t.Error("body was not automatically decompressed")
				}
				if encoding == "chunked" && (len(res.TransferEncoding) != 1 || res.TransferEncoding[0] != "chunked") {
					t.Errorf("transfer encoding=%v, want chunked", res.TransferEncoding)
				}
				body = &trackedBody{Reader: res.Body}
				res.Body = body
				return res, nil
			})}
			data, err := Request(context.Background(), client, http.MethodGet, srv.URL, "", nil, nil, ResponseBody)
			wantErr := fmt.Sprintf("response exceeds %d bytes", MaxResponseBytes)
			if err == nil || err.Error() != wantErr || data != nil {
				t.Fatalf("bytes=%d error=%v, want nil data and %q", len(data), err, wantErr)
			}
			if body == nil || !body.closed || body.read != MaxResponseBytes+1 {
				t.Fatalf("body=%+v, want closed body and exactly %d decoded bytes read", body, MaxResponseBytes+1)
			}
		})
	}
}

func TestHTTPIncompleteResponseBodies(t *testing.T) {
	for _, tt := range []struct {
		name    string
		cancel  bool
		wantErr error
	}{
		{"short body", false, io.ErrUnexpectedEOF},
		{"canceled after headers", true, context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "4")
				_, _ = io.WriteString(w, "ab")
				w.(http.Flusher).Flush()
				if tt.cancel {
					<-r.Context().Done()
				}
			}))
			defer srv.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			transport := http.DefaultTransport.(*http.Transport).Clone()
			defer transport.CloseIdleConnections()
			var body *trackedBody
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				res, err := transport.RoundTrip(req)
				if err != nil {
					return nil, err
				}
				body = &trackedBody{Reader: res.Body}
				res.Body = body
				if tt.cancel {
					cancel()
				}
				return res, nil
			})}
			data, err := Request(ctx, client, http.MethodGet, srv.URL, "", nil, nil, ResponseBody)
			if !errors.Is(err, tt.wantErr) || data != nil || body == nil || !body.closed {
				t.Fatalf("data=%q error=%v body=%+v, want nil data, wrapped %v, and closed body", data, err, body, tt.wantErr)
			}
		})
	}
}

func BenchmarkHTTPDeclaredOversize(b *testing.B) {
	var body *trackedBody
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		body = &trackedBody{Reader: zeroReader{}}
		return &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: MaxResponseBytes + 8192}, nil
	})}
	wantErr := fmt.Sprintf("response exceeds %d bytes", MaxResponseBytes)
	var readBytes int64
	b.ReportAllocs()
	for b.Loop() {
		data, err := Request(context.Background(), client, http.MethodGet, "https://example.invalid/audio", "", nil, nil, ResponseBody)
		if err == nil || err.Error() != wantErr || data != nil || !body.closed {
			b.Fatalf("bytes=%d error=%v closed=%t", len(data), err, body.closed)
		}
		readBytes += int64(body.read)
	}
	b.ReportMetric(float64(readBytes)/float64(b.N), "read-B/op")
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestHTTPClosesResponsesOnFailure(t *testing.T) {
	readErr := errors.New("interrupted body")
	for _, tt := range []struct {
		name, contentType, wantErr string
		status                     int
		contentLength              int64
		reader                     io.Reader
		wantRead                   int
	}{
		{"status", "", "HTTP 401 Unauthorized", 401, MaxResponseBytes + 1, strings.NewReader("secret body"), 0},
		{"partial status", "", "HTTP 206 Partial Content", 206, MaxResponseBytes + 1, strings.NewReader("partial body"), 0},
		{"content type", "application/json", "expected raw PCM response, got application/json", 200, MaxResponseBytes + 1, strings.NewReader("{}"), 0},
		{"interrupted read", "audio/pcm", "read response: interrupted body", 200, 4, io.MultiReader(strings.NewReader("ab"), failingReader{readErr}), 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := &trackedBody{Reader: tt.reader}
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tt.status, Header: http.Header{"Content-Type": {tt.contentType}}, Body: body, ContentLength: tt.contentLength}, nil
			})}
			data, err := Request(context.Background(), client, http.MethodGet, "https://example.invalid/audio", "", nil, nil, ResponsePCM)
			if err == nil || err.Error() != tt.wantErr || data != nil || !body.closed || body.read != tt.wantRead {
				t.Fatalf("data=%q error=%v closed=%t read=%d", data, err, body.closed, body.read)
			}
			if tt.name == "interrupted read" && !errors.Is(err, readErr) {
				t.Fatalf("read error is not wrapped: %v", err)
			}
		})
	}
}

func TestHTTPResponseKindIsIndependentOfAccept(t *testing.T) {
	for _, tt := range []struct {
		name, accept, contentType string
		kind                      ResponseKind
		valid                     bool
	}{
		{"PCM without Accept", "", "application/json", ResponsePCM, false},
		{"PCM reordered Accept", "application/octet-stream, audio/pcm", "application/json", ResponsePCM, false},
		{"PCM explicit Accept", "audio/pcm", "audio/pcm", ResponsePCM, true},
		{"JSON with audio Accept", "audio/pcm", "application/json", ResponseBody, true},
		{"WAV with audio Accept", "audio/wav", "audio/wav", ResponseBody, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// A close failure after a complete read does not discard valid data.
			body := &trackedBody{Reader: strings.NewReader("data"), closeErr: errors.New("close failed")}
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("Accept") != tt.accept {
					t.Errorf("Accept = %q", r.Header.Get("Accept"))
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {tt.contentType}}, Body: body}, nil
			})}
			data, err := Request(context.Background(), client, http.MethodGet, "https://example.invalid/audio", "", map[string]string{"Accept": tt.accept}, nil, tt.kind)
			if (err == nil) != tt.valid || !body.closed || (tt.valid && string(data) != "data") || (!tt.valid && data != nil) {
				t.Fatalf("data=%q error=%v closed=%t", data, err, body.closed)
			}
		})
	}
}
