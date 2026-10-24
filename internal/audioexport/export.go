// Package audioexport converts complete provider WAVs into final CLI files.
// Providers remain unaware of the optional, local FFmpeg dependency.
package audioexport

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type format struct {
	muxer string
	args  []string
}

func outputFormat(path string) (format, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".wav":
		return format{}, nil
	case ".mp3":
		return format{"mp3", []string{"-c:a", "libmp3lame", "-b:a", "96k", "-write_xing", "1"}}, nil
	case ".flac":
		return format{"flac", []string{"-c:a", "flac", "-compression_level", "5"}}, nil
	case ".opus":
		return format{"opus", []string{"-c:a", "libopus", "-b:a", "48k", "-vbr", "on", "-application", "audio"}}, nil
	case ".aac":
		return format{"adts", []string{"-c:a", "aac", "-b:a", "96k", "-profile:a", "aac_low"}}, nil
	default:
		return format{}, fmt.Errorf("output must use .wav, .mp3, .flac, .opus, or .aac extension")
	}
}

// ValidatePath rejects unsupported formats, special files, and unavailable parents.
// Source aliases are checked by the CLI before preparing an export.
func ValidatePath(path string) error {
	if _, err := outputFormat(path); err != nil {
		return err
	}
	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("output parent: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output parent must be a directory")
	}
	dst, err := os.Stat(path)
	if err == nil && !dst.Mode().IsRegular() {
		return fmt.Errorf("output path must be a regular file, not a directory or device")
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Export owns private staging files until Close. Prepare must precede synthesis.
type Export struct {
	path   string
	input  string
	dir    string
	ffmpeg string
	format format
}

// Prepare checks the actual decoder/encoder/muxer path using a tiny local WAV,
// before any paid work. WAV bypasses both executable lookup and staging.
func Prepare(ctx context.Context, path string) (_ *Export, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidatePath(path); err != nil {
		return nil, err
	}
	f, _ := outputFormat(path)
	e := &Export{path: path, input: path, format: f}
	if f.muxer == "" {
		return e, nil
	}
	e.ffmpeg, err = exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("%s export needs an installed FFmpeg on PATH (WAV needs none): %w", filepath.Ext(path), err)
	}
	// Absolute generated paths cannot be parsed as options or URL protocols.
	parent, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	e.dir, err = os.MkdirTemp(parent, ".timbrex-export-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			e.Close()
		}
	}()
	e.input = filepath.Join(e.dir, "source.wav")
	probe := filepath.Join(e.dir, "probe.wav")
	if err := os.WriteFile(probe, probeWAV(), 0o600); err != nil {
		return nil, err
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := e.encode(probeCtx, probe, filepath.Join(e.dir, "probe.audio")); err != nil {
		return nil, fmt.Errorf("FFmpeg %s export preflight: %w", filepath.Ext(path), err)
	}
	return e, nil
}

// InputPath is the provider's WAV destination, never a user-controlled URL.
func (e *Export) InputPath() string { return e.input }

// Finish encodes the complete WAV once, then publishes a seekable file.
// A WAV is already committed by the provider; cancellation after commit is success.
func (e *Export) Finish(ctx context.Context) error {
	if e.dir == "" {
		return nil
	}
	output := filepath.Join(e.dir, "output.audio")
	if err := e.encode(ctx, e.input, output); err != nil {
		return fmt.Errorf("FFmpeg %s export: %w", filepath.Ext(e.path), err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Rename(output, e.path)
}

// Close removes private files only after every encoder process has been joined.
func (e *Export) Close() {
	if e.dir != "" {
		_ = os.RemoveAll(e.dir)
	}
}

func (e *Export) encode(ctx context.Context, input, output string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Pre-create a private, fresh file. An existing file must never masquerade
	// as encoder success. A normal file (not a pipe) preserves MP3 Xing metadata.
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	args := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-nostats", "-y",
		"-protocol_whitelist", "file", "-f", "wav", "-c:a", "pcm_s16le", "-i", input,
		"-map", "0:a:0", "-map_metadata", "-1", "-map_chapters", "-1"}
	args = append(args, e.format.args...)
	args = append(args, "-f", e.format.muxer, output)
	if err := runFFmpeg(ctx, e.ffmpeg, args); err != nil {
		return err
	}
	return validateEncoded(output, e.format.muxer)
}

func probeWAV() []byte {
	// 100 ms of 24 kHz mono S16LE, the real providers' composition format.
	b := make([]byte, 44+4800)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b)-8))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], 24000)
	binary.LittleEndian.PutUint32(b[28:], 48000)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(len(b)-44))
	return b
}
