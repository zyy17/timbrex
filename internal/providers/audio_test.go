package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestRenderFailurePreservesExistingOutput(t *testing.T) {
	for _, mode := range []string{"request", "odd", "empty", "cancel", "wav", "mp3", "ogg", "flac", "mp4"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "out.wav")
			if err := os.WriteFile(path, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, err := Render(ctx, "test", Options{Out: path}, []Segment{{}, {}}, func(_ context.Context, i int) ([]byte, error) {
				if i == 0 {
					return []byte{1, 0}, nil
				}
				switch mode {
				case "odd":
					return []byte{1}, nil
				case "empty":
					return nil, nil
				case "wav":
					return append(wavHeader(2), 1, 0), nil
				case "mp3":
					return []byte("ID3\x04\x00\x00\x00\x00\x00\x00"), nil
				case "ogg":
					return []byte("OggS\x00\x00\x00\x00"), nil
				case "flac":
					return append([]byte("fLaC\x80\x00\x00\x22"), make([]byte, 34)...), nil
				case "mp4":
					return []byte("\x00\x00\x00\x18ftypM4A \x00\x00\x00\x00M4A mp42"), nil
				case "cancel":
					cancel()
					return []byte{1, 0}, nil
				default:
					return nil, errors.New("request failed")
				}
			})
			if err == nil {
				t.Fatal("expected failure")
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancel = %v", err)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "KEEP" {
				t.Fatalf("output = %q %v", data, err)
			}
			files, err := os.ReadDir(dir)
			if err != nil || len(files) != 1 {
				t.Fatalf("temporary files remain: %v %v", files, err)
			}
		})
	}
}

func TestRenderCancellationCheckpoints(t *testing.T) {
	for _, tt := range []struct {
		name        string
		cancelAfter int
		wantCalls   int
	}{
		{"before creation", 0, 0},
		{"between segments", 1, 1},
		{"after final segment", 2, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "out.wav")
			if err := os.WriteFile(path, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancelAfter == 0 {
				cancel()
			}
			calls := 0
			result, err := Render(ctx, "test", Options{Out: path}, []Segment{{}, {Pause: time.Millisecond}, {}}, func(context.Context, int) ([]byte, error) {
				calls++
				if calls == tt.cancelAfter {
					cancel()
				}
				return []byte{1, 0}, nil
			})
			if !errors.Is(err, context.Canceled) || result != (Result{}) || calls != tt.wantCalls {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls)
			}
			assertPreservedOutput(t, path)
		})
	}
}

func TestRenderFilesystemFailures(t *testing.T) {
	t.Run("create missing parent", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "missing", "out.wav")
		result, err := Render(context.Background(), "test", Options{Out: path}, []Segment{{}}, func(context.Context, int) ([]byte, error) {
			t.Error("synthesis started before opening output")
			return []byte{1, 0}, nil
		})
		if !errors.Is(err, os.ErrNotExist) || result != (Result{}) {
			t.Fatalf("result=%+v error=%v", result, err)
		}
		assertNoPartialFiles(t, dir)
	})
	t.Run("create nondirectory parent", func(t *testing.T) {
		dir := t.TempDir()
		blocker := filepath.Join(dir, "out.wav")
		if err := os.WriteFile(blocker, []byte("KEEP"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Render(context.Background(), "test", Options{Out: filepath.Join(blocker, "out.wav")}, []Segment{{}}, func(context.Context, int) ([]byte, error) {
			t.Error("synthesis started before opening output")
			return []byte{1, 0}, nil
		})
		if err == nil {
			t.Fatal("accepted a file as the parent directory")
		}
		assertPreservedOutput(t, blocker)
	})
	t.Run("rename over directory", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.wav")
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		keep := filepath.Join(path, "keep")
		if err := os.WriteFile(keep, []byte("KEEP"), 0o600); err != nil {
			t.Fatal(err)
		}
		calls := 0
		result, err := Render(context.Background(), "test", Options{Out: path}, []Segment{{}}, func(context.Context, int) ([]byte, error) {
			calls++
			return []byte{1, 0}, nil
		})
		var linkErr *os.LinkError
		if !errors.As(err, &linkErr) || linkErr.Op != "rename" || result != (Result{}) || calls != 1 {
			t.Fatalf("result=%+v error=%v calls=%d", result, err, calls)
		}
		assertPreservedOutput(t, keep)
		assertNoPartialFiles(t, dir)
	})
	for _, limit := range []uint64{32, 45} {
		t.Run(fmt.Sprintf("write limit %d", limit), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "out.wav")
			if err := os.WriteFile(path, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			// 32 fails the placeholder header; 45 allows the header and then
			// fails partway through PCM. This works even under privileged CI.
			var old syscall.Rlimit
			if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &old); err != nil {
				t.Fatal(err)
			}
			lim := old
			lim.Cur = limit
			if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &lim); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &old); err != nil {
					t.Errorf("restore file limit: %v", err)
				}
			}()
			calls := 0
			result, err := Render(context.Background(), "test", Options{Out: path}, []Segment{{}}, func(context.Context, int) ([]byte, error) {
				calls++
				return []byte{1, 0, 2, 0}, nil
			})
			wantCalls := 0
			if limit > 44 {
				wantCalls = 1
			}
			if !errors.Is(err, syscall.EFBIG) || result != (Result{}) || calls != wantCalls {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls)
			}
			assertPreservedOutput(t, path)
		})
	}
}

func assertPreservedOutput(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "KEEP" {
		t.Fatalf("previous output = %q, %v", data, err)
	}
	assertNoPartialFiles(t, filepath.Dir(path))
}

func assertNoPartialFiles(t *testing.T, dir string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, ".timbrex-*.partial"))
	if err != nil || len(files) != 0 {
		t.Fatalf("partial files = %v, %v", files, err)
	}
}
