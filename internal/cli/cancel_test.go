package cli

import (
	"bytes"
	"context"
	"errors"
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
