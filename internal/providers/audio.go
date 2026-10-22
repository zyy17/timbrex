package providers

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const SampleRate = 24000

// Render writes consecutive 24 kHz mono S16LE segments into one WAV.
// Requests and pauses are sequential. Only a complete file replaces the output.
func Render(ctx context.Context, id string, opt Options, segments []Segment, synth func(context.Context, int) ([]byte, error)) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	f, err := os.CreateTemp(filepath.Dir(opt.Out), ".timbrex-*.partial")
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	defer f.Close()
	if _, err := f.Write(make([]byte, 44)); err != nil {
		return Result{}, err
	}
	var size uint64
	for i, seg := range segments {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		var pcm []byte
		if seg.Pause > 0 {
			pcm = make([]byte, int(seg.Pause*time.Duration(SampleRate)/time.Second)*2)
		} else {
			pcm, err = synth(ctx, i)
			if err != nil {
				return Result{}, fmt.Errorf("%s segment %d: %w", id, i+1, err)
			}
			if len(pcm) == 0 || len(pcm)%2 != 0 {
				return Result{}, fmt.Errorf("%s segment %d: invalid 16-bit PCM audio", id, i+1)
			}
			// PCM has no header. Reject recognizable audio containers, including
			// those returned as application/octet-stream or decoded from JSON.
			if kind := http.DetectContentType(pcm); strings.HasPrefix(kind, "audio/") || kind == "application/ogg" {
				return Result{}, fmt.Errorf("%s segment %d: expected raw PCM, got %s", id, i+1, kind)
			}
		}
		size += uint64(len(pcm))
		if size > 1<<30 {
			return Result{}, fmt.Errorf("output exceeds 1 GiB")
		}
		if _, err := f.Write(pcm); err != nil {
			return Result{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	header := wavHeader(uint32(size))
	if _, err := f.WriteAt(header, 0); err != nil {
		return Result{}, err
	}
	if err := f.Close(); err != nil {
		return Result{}, err
	}
	if err := os.Rename(f.Name(), opt.Out); err != nil {
		return Result{}, err
	}
	return Result{Path: opt.Out, Provider: id}, nil
}

func wavHeader(size uint32) []byte {
	b := make([]byte, 44)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], size+36)
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], SampleRate)
	binary.LittleEndian.PutUint32(b[28:], SampleRate*2)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], size)
	return b
}
