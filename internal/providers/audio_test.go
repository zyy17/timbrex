package providers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
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
