package providers_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"

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
			segments, requests, err := providers.Prepare(context.Background(), document(t, tc.source), "test", "voice", func(seg providers.Segment) (string, error) {
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
	segments, requests, err := providers.Prepare(context.Background(), document(t, "One\n\nTwo"), "test", "voice", func(seg providers.Segment) (string, error) {
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
	_, _, err = providers.Prepare(context.Background(), nil, "test", "voice", func(providers.Segment) (string, error) { calls++; return "", nil })
	if err == nil || calls != 0 {
		t.Fatalf("invalid plan built %d requests: %v", calls, err)
	}
}

func TestPrepareCancellation(t *testing.T) {
	for _, cancelAt := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(cancelAt), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if cancelAt == 0 {
				cancel()
			}
			calls := 0
			segments, requests, err := providers.Prepare(ctx, document(t, "One\n\nTwo [pause 1ms]"), "test", "voice", func(seg providers.Segment) (string, error) {
				calls++
				if calls == cancelAt {
					cancel()
				}
				return providers.Text(seg.Nodes, nil)
			})
			if !errors.Is(err, context.Canceled) || segments != nil || requests != nil || calls != cancelAt {
				t.Fatalf("segments=%v requests=%v calls=%d error=%v", segments, requests, calls, err)
			}
		})
	}
}

func TestPrepareCancellationDuringLastBuild(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		started, release := make(chan struct{}), make(chan struct{})
		done := make(chan error, 1)
		doc := document(t, "Only segment")
		go func() {
			segments, requests, err := providers.Prepare(ctx, doc, "test", "voice", func(providers.Segment) (string, error) {
				close(started)
				<-release
				return "prepared after cancellation", nil
			})
			if segments != nil || requests != nil {
				err = fmt.Errorf("partial preparation returned: segments=%v requests=%v", segments, requests)
			}
			done <- err
		}()
		<-started
		cancel()
		synctest.Wait()
		if len(done) != 0 {
			t.Fatal("Prepare left a request builder running")
		}
		close(release)
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	})
}

// Cancel at a context checkpoint rather than relying on a planner timing race.
type planningCancelContext struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (c *planningCancelContext) Err() error {
	c.checks++
	if c.checks == 8 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestPrepareCancellationDuringPlanning(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	checking := &planningCancelContext{Context: ctx, cancel: cancel}
	doc := document(t, "One [pause 1ms] Two [pause 1ms] Three [pause 1ms] Four")
	builds := 0
	segments, requests, err := providers.Prepare(checking, doc, "test", "voice", func(providers.Segment) (string, error) {
		builds++
		return "unexpected", nil
	})
	if !errors.Is(err, context.Canceled) || segments != nil || requests != nil || builds != 0 {
		t.Fatalf("segments=%v requests=%v builds=%d error=%v", segments, requests, builds, err)
	}
}
