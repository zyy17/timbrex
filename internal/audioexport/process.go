package audioexport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const diagnosticLimit = 8192

// Keep draining diagnostics after the memory bound; a verbose child must not
// block on a full pipe. Only stderr writes this buffer and Run joins the copier.
type diagnostics struct{ data []byte }

func (d *diagnostics) Write(p []byte) (int, error) {
	n := len(p)
	if left := diagnosticLimit - len(d.data); left > 0 {
		d.data = append(d.data, p[:min(left, n)]...)
	}
	return n, nil
}

func runFFmpeg(ctx context.Context, executable string, args []string) error {
	cmd := exec.CommandContext(ctx, executable, args...)
	// Do not pass provider credentials, FFREPORT destinations, or loader hooks.
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	cmd.WaitDelay = time.Second
	var stderr diagnostics
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	err := cmd.Run() // CommandContext kills, and Run waits, before cleanup.
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		message := strings.TrimSpace(string(stderr.data))
		if message != "" {
			return fmt.Errorf("encoder failed: %w: %s", err, message)
		}
		return fmt.Errorf("encoder failed: %w", err)
	}
	return nil
}

func validateEncoded(path, muxer string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > 1<<30 {
		return fmt.Errorf("encoder must produce a nonempty regular file no larger than 1 GiB")
	}
	header := make([]byte, 64)
	n, err := io.ReadFull(f, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return err
	}
	header = header[:n]
	var valid bool
	switch muxer {
	case "mp3":
		valid = bytes.HasPrefix(header, []byte("ID3")) || (n >= 2 && header[0] == 0xff && header[1]&0xe0 == 0xe0)
	case "flac":
		valid = bytes.HasPrefix(header, []byte("fLaC"))
	case "opus":
		valid = bytes.HasPrefix(header, []byte("OggS")) && bytes.Contains(header, []byte("OpusHead"))
	case "adts":
		valid = n >= 2 && header[0] == 0xff && header[1]&0xf6 == 0xf0
	}
	if !valid {
		return fmt.Errorf("encoder did not produce the expected %s file", muxer)
	}
	return nil
}
