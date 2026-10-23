package providers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteOutputPrivatePermissions(t *testing.T) {
	for _, existing := range []bool{false, true} {
		name := "new output"
		if existing {
			name = "replaced public output"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "out.wav")
			if existing {
				if err := os.WriteFile(path, []byte("KEEP"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			err := WriteOutput(context.Background(), path, func(f *os.File) error {
				_, err := f.WriteString("complete")
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "complete" {
				t.Fatalf("output=%q error=%v", data, err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm()&0o177 != 0 {
				t.Fatalf("output permissions are not private: %04o", info.Mode().Perm())
			}
			assertNoPartialFiles(t, filepath.Dir(path))
		})
	}
}

func TestWriteOutputFailureBeforeCommit(t *testing.T) {
	writeErr := errors.New("writer failed")
	for _, mode := range []string{"write", "close", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "out.wav")
			if err := os.WriteFile(path, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wantErr := writeErr
			err := WriteOutput(ctx, path, func(f *os.File) error {
				if _, err := f.WriteString("complete"); err != nil {
					return err
				}
				switch mode {
				case "close":
					wantErr = os.ErrClosed
					return f.Close() // The transaction's subsequent Close must fail.
				case "cancel":
					wantErr = context.Canceled
					cancel()
					return nil
				default:
					return writeErr
				}
			})
			if !errors.Is(err, wantErr) {
				t.Fatalf("error=%v, want %v", err, wantErr)
			}
			assertPreservedOutput(t, path)
		})
	}
}
