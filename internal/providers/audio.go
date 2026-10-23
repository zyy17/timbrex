package providers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const SampleRate = 24000

// Render writes consecutive 24 kHz mono S16LE segments into one WAV.
// Speech may run concurrently, but audio and local pauses keep script order.
// Only a complete file replaces the output. The synth callback must be safe
// for simultaneous calls when Concurrency exceeds one.
func Render(ctx context.Context, id string, opt Options, segments []Segment, synth func(context.Context, int) ([]byte, error)) (Result, error) {
	if err := ValidateConcurrency(opt.Concurrency); err != nil {
		return Result{}, err
	}
	err := WriteOutput(ctx, opt.Out, func(f *os.File) error {
		return renderWAV(ctx, f, id, opt.Concurrency, segments, synth)
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Path: opt.Out, Provider: id}, nil
}

// A slot is retained until its audio is written, not merely until its request
// finishes. A slow early request therefore cannot accumulate the whole script.
type audioSlot struct {
	pcm  []byte
	done chan struct{}
}

type audioJob struct {
	index int
	slot  *audioSlot
}

type wavWriter interface {
	io.Writer
	io.WriterAt
}

func renderWAV(ctx context.Context, f wavWriter, id string, concurrency int, segments []Segment, synth func(context.Context, int) ([]byte, error)) (err error) {
	ctx, cancel := context.WithCancelCause(ctx)
	var workers sync.WaitGroup
	var jobs chan audioJob
	defer func() {
		// Record a writer error before cancellation reaches the workers, then
		// join even workers that return late or do not honor cancellation.
		cancel(err)
		if jobs != nil {
			close(jobs)
		}
		workers.Wait()
		if err != nil {
			err = context.Cause(ctx)
		}
	}()
	if _, err := f.Write(make([]byte, 44)); err != nil {
		return err
	}
	var slots []*audioSlot
	if concurrency > 1 {
		slots = make([]*audioSlot, concurrency)
		jobs = make(chan audioJob, concurrency)
		for range concurrency {
			workers.Add(1)
			go func() {
				defer workers.Done()
				for job := range jobs {
					if ctx.Err() != nil {
						return
					}
					var err error
					job.slot.pcm, err = synthPCM(ctx, id, job.index, synth)
					if err != nil {
						// Record the failure now, not when the ordered writer
						// reaches it. Sibling cancellation must not hide it.
						cancel(err)
						return
					}
					close(job.slot.done)
				}
			}()
		}
	}
	var size uint64
	next := 0
	for i, seg := range segments {
		if err := ctx.Err(); err != nil {
			return context.Cause(ctx)
		}
		// Pauses also occupy the rolling window, but need no worker or PCM
		// allocation until their turn to be written.
		for concurrency > 1 && next < len(segments) && next-i < concurrency {
			if err := ctx.Err(); err != nil {
				return context.Cause(ctx)
			}
			index := next
			next++
			if segments[index].Pause > 0 {
				continue
			}
			slot := &audioSlot{done: make(chan struct{})}
			slots[index%concurrency] = slot
			jobs <- audioJob{index: index, slot: slot}
		}
		var pcm []byte
		if seg.Pause > 0 {
			pcm = make([]byte, int(seg.Pause*time.Duration(SampleRate)/time.Second)*2)
		} else if concurrency > 1 {
			slot := slots[i%concurrency]
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case <-slot.done:
				pcm = slot.pcm
				slot.pcm = nil
			}
			slots[i%concurrency] = nil
		} else {
			var err error
			pcm, err = synthPCM(ctx, id, i, synth)
			if err != nil {
				return err
			}
		}
		if err := ctx.Err(); err != nil {
			return context.Cause(ctx)
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
		return context.Cause(ctx)
	}
	header := wavHeader(uint32(size))
	if _, err := f.WriteAt(header, 0); err != nil {
		return err
	}
	return nil
}

func synthPCM(ctx context.Context, id string, i int, synth func(context.Context, int) ([]byte, error)) ([]byte, error) {
	pcm, err := synth(ctx, i)
	if err != nil {
		return nil, fmt.Errorf("%s segment %d: %w", id, i+1, err)
	}
	if len(pcm) == 0 || len(pcm)%2 != 0 {
		return nil, fmt.Errorf("%s segment %d: invalid 16-bit PCM audio", id, i+1)
	}
	// PCM has no header. Reject recognizable audio containers, including
	// those returned as application/octet-stream or decoded from JSON.
	kind := http.DetectContentType(pcm)
	if bytes.HasPrefix(pcm, []byte("fLaC")) {
		kind = "audio/flac"
	}
	if strings.HasPrefix(kind, "audio/") || strings.HasPrefix(kind, "video/") || kind == "application/ogg" {
		return nil, fmt.Errorf("%s segment %d: expected raw PCM, got %s", id, i+1, kind)
	}
	return pcm, nil
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
