package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const MaxResponseBytes = 64 << 20

func HTTPClient() *http.Client {
	return &http.Client{
		Timeout:       2 * time.Minute,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func PostJSON(ctx context.Context, client *http.Client, endpoint string, headers map[string]string, body any) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return Request(ctx, client, http.MethodPost, endpoint, "application/json", headers, data)
}

func Request(ctx context.Context, client *http.Client, method, endpoint, contentType string, headers map[string]string, body []byte) ([]byte, error) {
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
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d %s", res.StatusCode, http.StatusText(res.StatusCode))
	}
	if strings.HasPrefix(req.Header.Get("Accept"), "audio/") {
		kind := res.Header.Get("Content-Type")
		if kind != "" && !strings.HasPrefix(kind, "audio/") && !strings.HasPrefix(kind, "application/octet-stream") {
			return nil, fmt.Errorf("expected audio response, got %s", kind)
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
