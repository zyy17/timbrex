// Package inputfile reads regular script and configuration files.
package inputfile

import (
	"errors"
	"io"
	"os"
	"syscall"
)

// Read follows symlinks and rejects nonregular files before reading them.
func Read(path string) ([]byte, error) {
	// A FIFO must not block in open before its file type can be checked.
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, &os.PathError{Op: "read", Path: path, Err: errors.New("input must be a regular file")}
	}
	return io.ReadAll(f)
}
