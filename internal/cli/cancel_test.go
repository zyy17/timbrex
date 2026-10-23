package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanceledCommandDoesNotLoadInputs(t *testing.T) {
	for _, command := range []string{"check", "synth", "config"} {
		t.Run(command, func(t *testing.T) {
			dir := t.TempDir()
			args := []string{"--config", filepath.Join(dir, "missing.yaml"), command}
			if command != "config" {
				args = append(args, filepath.Join(dir, "missing.tbx"))
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			var out bytes.Buffer
			err := run(ctx, args, strings.NewReader(""), &out, &out)
			if !errors.Is(err, context.Canceled) || out.Len() != 0 {
				t.Fatalf("canceled command: err=%v output=%q", err, out.String())
			}
		})
	}
}

// A diagnostic writer makes cancellation during command execution deterministic.
type cancelWriter struct {
	cancel context.CancelFunc
}

func (w cancelWriter) Write(p []byte) (int, error) {
	w.cancel()
	return len(p), nil
}

func TestCancellationAfterDiagnostic(t *testing.T) {
	for _, command := range []string{"check", "synth"} {
		t.Run(command, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "scene.tbx")
			if err := os.WriteFile(path, []byte("---\ncast: auto\n---\nalice: Hello\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			outPath := filepath.Join(dir, "scene.wav")
			if err := os.WriteFile(outPath, []byte("KEEP"), 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var out bytes.Buffer
			err := run(ctx, []string{command, path}, nil, &out, cancelWriter{cancel})
			if !errors.Is(err, context.Canceled) || out.Len() != 0 {
				t.Fatalf("canceled after diagnostic: err=%v stdout=%q", err, out.String())
			}
			data, err := os.ReadFile(outPath)
			if err != nil || string(data) != "KEEP" {
				t.Fatalf("existing output changed: err=%v data=%q", err, data)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 2 {
				t.Fatalf("unexpected temporary output: err=%v entries=%v", err, entries)
			}
		})
	}
}

func TestCancellationAfterSynthesisCommitStillSucceeds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.tbx")
	if err := os.WriteFile(path, []byte("Hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := run(ctx, []string{"synth", path, "--provider", "stub"}, nil, cancelWriter{cancel}, &bytes.Buffer{})
	if err != nil || !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("completed synthesis: err=%v context=%v", err, ctx.Err())
	}
	data, err := os.ReadFile(filepath.Join(dir, "scene.wav"))
	if err != nil || !bytes.HasPrefix(data, []byte("RIFF")) {
		t.Fatalf("missing committed WAV: err=%v data=%q", err, data)
	}
}
