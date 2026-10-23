package providers

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type renderOutcome struct {
	result Result
	err    error
}

func TestRenderBoundedOrderedWindow(t *testing.T) {
	for _, concurrency := range []int{0, 1, 2, 3, 8} {
		t.Run(fmt.Sprint(concurrency), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				window := max(1, concurrency)
				count := window*3 + 2
				releases := make([]chan struct{}, count)
				for i := range releases {
					releases[i] = make(chan struct{})
				}
				started := make(chan int, count)
				var active, peak atomic.Int32
				path := filepath.Join(t.TempDir(), "out.wav")
				done := make(chan renderOutcome, 1)
				go func() {
					result, err := Render(context.Background(), "test", Options{Out: path, Concurrency: concurrency}, make([]Segment, count), func(_ context.Context, i int) ([]byte, error) {
						n := active.Add(1)
						for old := peak.Load(); n > old; old = peak.Load() {
							if peak.CompareAndSwap(old, n) {
								break
							}
						}
						defer active.Add(-1)
						started <- i
						<-releases[i]
						return []byte{byte(i + 1), 0}, nil
					})
					done <- renderOutcome{result, err}
				}()
				for first := 0; first < count; first += window {
					last := min(first+window, count)
					synctest.Wait()
					var got []int
					for len(started) > 0 {
						got = append(got, <-started)
					}
					slices.Sort(got)
					want := make([]int, last-first)
					for i := range want {
						want[i] = first + i
					}
					if !slices.Equal(got, want) {
						t.Fatalf("admitted = %v, want %v", got, want)
					}
					// Finish later requests first. None may free admission until
					// the earliest uncommitted segment also completes.
					for i := last - 1; i > first; i-- {
						close(releases[i])
					}
					synctest.Wait()
					if len(started) != 0 {
						t.Fatal("completion opened the window past a blocked first segment")
					}
					close(releases[first])
				}
				outcome := <-done
				if outcome.err != nil || outcome.result.Path != path {
					t.Fatalf("result=%+v error=%v", outcome.result, outcome.err)
				}
				if active.Load() != 0 || peak.Load() != int32(window) {
					t.Fatalf("active=%d peak=%d, want peak %d", active.Load(), peak.Load(), window)
				}
				var want []byte
				for i := range count {
					want = append(want, byte(i+1), 0)
				}
				assertWAVPCM(t, path, want)
			})
		})
	}
}

func TestRenderPausesOccupyWindowWithoutRequests(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		segments := make([]Segment, 42)
		for i := range segments {
			segments[i].Pause = time.Millisecond
		}
		segments[10], segments[20] = Segment{}, Segment{}
		started := make(chan int, 2)
		release := make(chan struct{})
		path := filepath.Join(t.TempDir(), "out.wav")
		done := make(chan error, 1)
		go func() {
			_, err := Render(context.Background(), "test", Options{Out: path, Concurrency: 3}, segments, func(_ context.Context, i int) ([]byte, error) {
				started <- i
				<-release
				return []byte{byte(i), 0}, nil
			})
			done <- err
		}()
		synctest.Wait()
		if len(started) != 1 || <-started != 10 {
			t.Fatal("pauses did not bound admission behind the first speech segment")
		}
		release <- struct{}{}
		synctest.Wait()
		if len(started) != 1 || <-started != 20 {
			t.Fatal("unexpected request for a local pause")
		}
		release <- struct{}{}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		var want []byte
		for i, seg := range segments {
			if seg.Pause > 0 {
				want = append(want, make([]byte, 48)...)
			} else {
				want = append(want, byte(i), 0)
			}
		}
		assertWAVPCM(t, path, want)
	})
}

func TestRenderFailureCancelsAndJoinsLateWorkers(t *testing.T) {
	for _, lateError := range []bool{false, true} {
		t.Run(fmt.Sprint(lateError), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				path := preservedOutput(t)
				original := errors.New("original provider failure")
				fail, late := make(chan struct{}), make(chan struct{})
				started := make(chan int, 6)
				canceled := make(chan int, 2)
				var active atomic.Int32
				done := make(chan renderOutcome, 1)
				go func() {
					result, err := Render(context.Background(), "test", Options{Out: path, Concurrency: 3}, make([]Segment, 6), func(ctx context.Context, i int) ([]byte, error) {
						active.Add(1)
						defer active.Add(-1)
						started <- i
						if i == 2 {
							<-fail
							return nil, original
						}
						<-ctx.Done()
						canceled <- i
						if i == 0 {
							<-late
							if lateError {
								return nil, errors.New("later provider failure")
							}
							return []byte{1, 0}, nil
						}
						return nil, ctx.Err()
					})
					done <- renderOutcome{result, err}
				}()
				synctest.Wait()
				if len(started) != 3 {
					t.Fatalf("started=%d, want 3", len(started))
				}
				close(fail)
				synctest.Wait()
				if len(canceled) != 2 || len(started) != 3 || len(done) != 0 {
					t.Fatalf("canceled=%d started=%d returned=%d", len(canceled), len(started), len(done))
				}
				assertPendingOutput(t, path)
				close(late)
				outcome := <-done
				if !errors.Is(outcome.err, original) || !strings.Contains(outcome.err.Error(), "test segment 3:") || outcome.result != (Result{}) {
					t.Fatalf("result=%+v error=%v", outcome.result, outcome.err)
				}
				if active.Load() != 0 {
					t.Fatalf("%d workers remain", active.Load())
				}
				assertPreservedOutput(t, path)
			})
		})
	}
}

func TestRenderConcurrentCancellationJoinsWorkers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		path := preservedOutput(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		started, canceled := make(chan struct{}, 2), make(chan struct{}, 2)
		late := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			_, err := Render(ctx, "test", Options{Out: path, Concurrency: 2}, make([]Segment, 8), func(ctx context.Context, _ int) ([]byte, error) {
				started <- struct{}{}
				<-ctx.Done()
				canceled <- struct{}{}
				<-late
				return []byte{1, 0}, nil
			})
			done <- err
		}()
		synctest.Wait()
		if len(started) != 2 {
			t.Fatal("workers did not start")
		}
		cancel()
		synctest.Wait()
		if len(canceled) != 2 || len(done) != 0 || len(started) != 2 {
			t.Fatal("cancellation returned early or admitted new work")
		}
		assertPendingOutput(t, path)
		close(late)
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
		assertPreservedOutput(t, path)
	})
}

func TestRenderConcurrentWriteFailureJoinsWorkers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		path := preservedOutput(t)
		started, canceled, late := make(chan struct{}), make(chan struct{}), make(chan struct{})
		done := make(chan error, 1)
		go func() {
			done <- WriteOutput(context.Background(), path, func(f *os.File) error {
				return renderWAV(context.Background(), f, "test", 2, make([]Segment, 4), func(ctx context.Context, i int) ([]byte, error) {
					if i == 0 {
						<-started
						if err := f.Close(); err != nil {
							return nil, err
						}
						return []byte{1, 0}, nil
					}
					if i != 1 {
						return nil, errors.New("admitted after writer failure")
					}
					close(started)
					<-ctx.Done()
					close(canceled)
					<-late
					return nil, ctx.Err()
				})
			})
		}()
		synctest.Wait()
		select {
		case <-canceled:
		default:
			t.Fatal("write failure did not cancel sibling")
		}
		if len(done) != 0 {
			t.Fatal("write failure returned before sibling exited")
		}
		assertPendingOutput(t, path)
		close(late)
		if err := <-done; !errors.Is(err, os.ErrClosed) {
			t.Fatalf("error=%v", err)
		}
		assertPreservedOutput(t, path)
	})
}

func TestRenderConcurrentRejectsInvalidPCM(t *testing.T) {
	for _, pcm := range [][]byte{nil, {1}, append(wavHeader(2), 1, 0), []byte("ID3\x04\x00\x00\x00\x00\x00\x00"), []byte("OggS\x00\x00\x00\x00"), []byte("fLaC\x00\x00\x00\x00")} {
		synctest.Test(t, func(t *testing.T) {
			path := preservedOutput(t)
			ready := make(chan struct{})
			_, err := Render(context.Background(), "test", Options{Out: path, Concurrency: 2}, make([]Segment, 4), func(ctx context.Context, i int) ([]byte, error) {
				if i == 1 {
					<-ready
					return pcm, nil
				}
				if i != 0 {
					return nil, errors.New("admitted after invalid audio")
				}
				close(ready)
				<-ctx.Done()
				return nil, ctx.Err()
			})
			if err == nil || errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "test segment 2:") {
				t.Fatalf("error=%v", err)
			}
			assertPreservedOutput(t, path)
		})
	}
}

func TestRenderRejectsConcurrencyBeforeOutput(t *testing.T) {
	for _, concurrency := range []int{-1, 9, 1000} {
		path := preservedOutput(t)
		result, err := Render(context.Background(), "test", Options{Out: path, Concurrency: concurrency}, []Segment{{}}, func(context.Context, int) ([]byte, error) {
			t.Fatal("invalid concurrency reached synthesis")
			return nil, nil
		})
		if err == nil || !strings.Contains(err.Error(), "concurrency") || result != (Result{}) {
			t.Fatalf("result=%+v error=%v", result, err)
		}
		assertPreservedOutput(t, path)
	}
}

func TestRenderConcurrentAlreadyCanceled(t *testing.T) {
	path := preservedOutput(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Render(ctx, "test", Options{Out: path, Concurrency: 8}, make([]Segment, 20), func(context.Context, int) ([]byte, error) {
		t.Error("started synthesis after cancellation")
		return nil, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	assertPreservedOutput(t, path)
}

func preservedOutput(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "out.wav")
	if err := os.WriteFile(path, []byte("KEEP"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertPendingOutput(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "KEEP" {
		t.Fatalf("output=%q error=%v", data, err)
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".timbrex-*.partial"))
	if err != nil || len(files) != 1 {
		t.Fatalf("temporary output cleaned before workers joined: files=%v error=%v", files, err)
	}
}

func assertWAVPCM(t *testing.T, path string, want []byte) {
	t.Helper()
	wav, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(wav) < 44 || !bytes.Equal(wav[:44], wavHeader(uint32(len(want)))) || !bytes.Equal(wav[44:], want) {
		t.Fatalf("WAV order/header mismatch: size=%d, want PCM bytes=%d", len(wav), len(want))
	}
	if size := binary.LittleEndian.Uint32(wav[40:44]); int(size) != len(want) {
		t.Fatalf("data size=%d, want %d", size, len(want))
	}
	assertNoPartialFiles(t, filepath.Dir(path))
}

type cancelWAVWriter struct {
	*os.File
	cancel context.CancelFunc
	header bool
}

func (w cancelWAVWriter) Write(p []byte) (int, error) {
	n, err := w.File.Write(p)
	if !w.header && len(p) != 44 {
		w.cancel()
	}
	return n, err
}

func (w cancelWAVWriter) WriteAt(p []byte, off int64) (int, error) {
	n, err := w.File.WriteAt(p, off)
	if w.header {
		w.cancel()
	}
	return n, err
}

func TestRenderCancellationDuringPCMWriteJoinsWorkers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		path := preservedOutput(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		started, canceled, late := make(chan struct{}), make(chan struct{}), make(chan struct{})
		done := make(chan error, 1)
		go func() {
			done <- WriteOutput(ctx, path, func(f *os.File) error {
				return renderWAV(ctx, cancelWAVWriter{File: f, cancel: cancel}, "test", 2, make([]Segment, 4), func(ctx context.Context, i int) ([]byte, error) {
					if i == 0 {
						<-started
						return []byte{1, 0}, nil
					}
					if i != 1 {
						return nil, errors.New("admitted after canceled write")
					}
					close(started)
					<-ctx.Done()
					close(canceled)
					<-late
					return []byte{2, 0}, nil
				})
			})
		}()
		synctest.Wait()
		select {
		case <-canceled:
		default:
			t.Fatal("cancellation during write did not reach sibling")
		}
		if len(done) != 0 {
			t.Fatal("returned before late sibling exited")
		}
		assertPendingOutput(t, path)
		close(late)
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
		assertPreservedOutput(t, path)
	})
}

func TestRenderCancellationDuringHeaderWritePreventsPublication(t *testing.T) {
	path := preservedOutput(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := WriteOutput(ctx, path, func(f *os.File) error {
		return renderWAV(ctx, cancelWAVWriter{File: f, cancel: cancel, header: true}, "test", 2, make([]Segment, 3), func(context.Context, int) ([]byte, error) {
			return []byte{1, 0}, nil
		})
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	assertPreservedOutput(t, path)
}
