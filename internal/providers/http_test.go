package providers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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
		_, err := Request(context.Background(), nil, http.MethodPost, srv.URL, "", nil, nil)
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
			_, err := Request(context.Background(), nil, http.MethodGet, srv.URL, "", map[string]string{"Accept": "audio/*"}, nil)
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
	_, err := Request(ctx, nil, http.MethodGet, srv.URL, "", nil, nil)
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
