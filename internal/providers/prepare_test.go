package providers_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/providers"
)

func TestPrepareKeepsPauseIndicesThroughRender(t *testing.T) {
	for _, tc := range []struct {
		source   string
		requests []string
		indices  []int
	}{
		{"[pause 1ms] One [pause 1ms] Two [pause 1ms]", []string{"", "One", "", "Two", ""}, []int{1, 3}},
		{"[pause 1ms]", []string{""}, nil},
	} {
		t.Run(tc.source, func(t *testing.T) {
			builds := 0
			segments, requests, err := providers.Prepare(document(t, tc.source), "test", "voice", func(seg providers.Segment) (string, error) {
				builds++
				text, err := providers.Text(seg.Nodes, nil)
				return strings.TrimSpace(text), err
			})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(requests, tc.requests) {
				t.Fatalf("requests=%q", requests)
			}
			out := filepath.Join(t.TempDir(), "out.wav")
			var indices []int
			_, err = providers.Render(context.Background(), "test", providers.Options{Out: out}, segments, func(_ context.Context, i int) ([]byte, error) {
				if builds != len(tc.indices) {
					t.Fatalf("incomplete preflight: %d builds", builds)
				}
				indices = append(indices, i)
				if requests[i] == "" {
					t.Fatal("request/segment indices lost")
				}
				return []byte{1, 0}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(indices, tc.indices) {
				t.Fatalf("rendered indices=%v", indices)
			}
			wav, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			pauseCount := len(tc.requests) - len(tc.indices)
			if len(wav) != 44+pauseCount*48+len(tc.indices)*2 {
				t.Fatalf("WAV bytes=%d", len(wav))
			}
		})
	}
}

func TestPrepareRejectsBeforeReturningRequests(t *testing.T) {
	calls := 0
	segments, requests, err := providers.Prepare(document(t, "One\n\nTwo"), "test", "voice", func(seg providers.Segment) (string, error) {
		calls++
		if calls == 2 {
			return "", context.Canceled
		}
		return providers.Text(seg.Nodes, nil)
	})
	if calls != 2 || segments != nil || requests != nil || !errors.Is(err, context.Canceled) || !strings.HasPrefix(err.Error(), "test: ") {
		t.Fatalf("segments=%v requests=%v calls=%d error=%v", segments, requests, calls, err)
	}
	calls = 0
	_, _, err = providers.Prepare(nil, "test", "voice", func(providers.Segment) (string, error) { calls++; return "", nil })
	if err == nil || calls != 0 {
		t.Fatalf("invalid plan built %d requests: %v", calls, err)
	}
}
