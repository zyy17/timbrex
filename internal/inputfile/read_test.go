package inputfile

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadRegularFiles(t *testing.T) {
	for _, body := range []string{"", "Hello.\n"} {
		t.Run(body, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "source.tbx")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(dir, "link.tbx")
			if err := os.Symlink(path, link); err != nil {
				t.Fatal(err)
			}
			for _, input := range []string{path, link} {
				got, err := Read(input)
				if err != nil || !bytes.Equal(got, []byte(body)) {
					t.Fatalf("Read(%q) = %q, %v", input, got, err)
				}
			}
		})
	}
}

func TestReadRejectsNonregularFiles(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(t.TempDir(), "directory-link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{dir, link, os.DevNull} {
		_, err := Read(path)
		var pe *os.PathError
		if !errors.As(err, &pe) || pe.Path != path || pe.Op != "read" || !strings.Contains(pe.Err.Error(), "input must be a regular file") {
			t.Fatalf("Read(%q) = %v", path, err)
		}
	}
}

func TestReadPreservesMissingFileError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	_, err := Read(path)
	var pe *os.PathError
	if !errors.Is(err, os.ErrNotExist) || !errors.As(err, &pe) || pe.Path != path {
		t.Fatalf("Read(%q) = %v", path, err)
	}
}
