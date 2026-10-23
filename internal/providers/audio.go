package providers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const SampleRate = 24000

// Render writes consecutive 24 kHz mono S16LE segments into one WAV.
// Requests and pauses are sequential. Only a complete file replaces the output.
func Render(ctx context.Context, id string, opt Options, segments []Segment, synth func(context.Context, int) ([]byte, error)) (Result, error) {
	err := WriteOutput(ctx, opt.Out, func(f *os.File) error {
		return renderWAV(ctx, f, id, segments, synth)
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Path: opt.Out, Provider: id}, nil
}

func renderWAV(ctx context.Context, f *os.File, id string, segments []Segment, synth func(context.Context, int) ([]byte, error)) error {
	if _, err := f.Write(make([]byte, 44)); err != nil {
		return err
	}
	var size uint64
	for i, seg := range segments {
		if err := ctx.Err(); err != nil {
			return err
		}
		var pcm []byte
		var err error
		if seg.Pause > 0 {
			pcm = make([]byte, int(seg.Pause*time.Duration(SampleRate)/time.Second)*2)
		} else {
			pcm, err = synth(ctx, i)
			if err != nil {
				return fmt.Errorf("%s segment %d: %w", id, i+1, err)
			}
			if len(pcm) == 0 || len(pcm)%2 != 0 {
				return fmt.Errorf("%s segment %d: invalid 16-bit PCM audio", id, i+1)
			}
			// PCM has no header. Reject recognizable audio containers, including
			// those returned as application/octet-stream or decoded from JSON.
			kind := http.DetectContentType(pcm)
			if bytes.HasPrefix(pcm, []byte("fLaC")) {
				kind = "audio/flac"
			}
			if strings.HasPrefix(kind, "audio/") || strings.HasPrefix(kind, "video/") || kind == "application/ogg" {
				return fmt.Errorf("%s segment %d: expected raw PCM, got %s", id, i+1, kind)
			}
		}
		size += uint64(len(pcm))
		if size > 1<<30 {
			return fmt.Errorf("output exceeds 1 GiB")
		}
		if _, err := f.Write(pcm); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	header := wavHeader(uint32(size))
	if _, err := f.WriteAt(header, 0); err != nil {
		return err
	}
	return nil
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
