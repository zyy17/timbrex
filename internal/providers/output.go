package providers

import (
	"context"
	"os"
	"path/filepath"
)

// WriteOutput publishes a complete sibling temporary file without exposing
// partially written audio. New and replaced files keep CreateTemp's private
// 0600 mode (before umask); existing destination permissions are not preserved.
// Cancellation is checked before creation and immediately before the rename.
func WriteOutput(ctx context.Context, path string, write func(*os.File) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".timbrex-*.partial")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	defer f.Close()
	if err := write(f); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
