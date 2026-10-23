package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const MaxResponseBytes = 64 << 20

// ResponseKind selects transport-level validation independently of Accept.
// JSON and WAV bodies are validated by their adapters after the bounded read.
type ResponseKind uint8

const (
	ResponseBody ResponseKind = iota
	ResponsePCM
)

func HTTPClient() *http.Client {
	return &http.Client{
		Timeout:       2 * time.Minute,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func PostJSON(ctx context.Context, client *http.Client, endpoint string, headers map[string]string, body any, expected ResponseKind) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return Request(ctx, client, http.MethodPost, endpoint, "application/json", headers, data, expected)
}

func Request(ctx context.Context, client *http.Client, method, endpoint, contentType string, headers map[string]string, body []byte, expected ResponseKind) ([]byte, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return nil, fmt.Errorf("invalid HTTP endpoint")
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("User-Agent", "timbrex")
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	if client == nil {
		client = HTTPClient()
	}
	res, err := client.Do(req)
	if err != nil {
		// url.Error may include a signed audio URL; keep it out of diagnostics.
		cause := err
		var e *url.Error
		if errors.As(err, &e) {
			cause = e.Err
		}
		return nil, fmt.Errorf("HTTP request failed: %w", cause)
	}
	defer res.Body.Close()
	// These adapters request complete responses and never send Range headers.
	if res.StatusCode == http.StatusPartialContent || res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d %s", res.StatusCode, http.StatusText(res.StatusCode))
	}
	if expected == ResponsePCM {
		if err := checkPCMContentType(res.Header.Get("Content-Type")); err != nil {
			return nil, err
		}
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, MaxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if len(data) > MaxResponseBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", MaxResponseBytes)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	return data, nil
}

func checkPCMContentType(header string) error {
	if header == "" {
		return nil
	}
	kind, params, err := mime.ParseMediaType(header)
	if err != nil {
		return fmt.Errorf("invalid audio Content-Type")
	}
	switch kind {
	case "audio/pcm", "audio/x-pcm", "audio/raw", "application/octet-stream":
	default:
		return fmt.Errorf("expected raw PCM response, got %s", kind)
	}
	for _, field := range []struct {
		name string
		want int
	}{
		{"rate", SampleRate},
		{"channels", 1},
	} {
		if value, ok := params[field.name]; ok {
			got, err := strconv.Atoi(value)
			if err != nil || got != field.want {
				return fmt.Errorf("expected PCM %s=%d", field.name, field.want)
			}
		}
	}
	return nil
}
