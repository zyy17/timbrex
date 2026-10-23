// Package stub is a fake provider that writes a short tone as WAV.
// It does not call a speech service. Tests and the default synth path use it.
package stub

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

const (
	sampleRate = 8000
	toneHz     = 440.0
)

func init() {
	providers.Register(provider{})
}

type provider struct{}

func (provider) ID() string { return "stub" }

func (provider) Synthesize(ctx context.Context, doc *ir.Document, opt providers.Options) (providers.Result, error) {
	if err := ctx.Err(); err != nil {
		return providers.Result{}, err
	}
	if err := providers.ValidateConcurrency(opt.Concurrency); err != nil {
		return providers.Result{}, err
	}
	if opt.Concurrency > 1 {
		return providers.Result{}, fmt.Errorf("stub does not support concurrency greater than 1")
	}
	pcm := tone(sampleCount(doc))
	if err := writeWAV(ctx, opt.Out, pcm, sampleRate); err != nil {
		return providers.Result{}, err
	}
	return providers.Result{Path: opt.Out, Provider: "stub"}, nil
}

func sampleCount(doc *ir.Document) int {
	n := 0
	if doc != nil {
		for _, utt := range doc.Utterances {
			for _, node := range utt.Nodes {
				if node.Kind == ir.Text {
					n += len([]rune(node.Text))
				}
			}
		}
	}
	if n < 1 {
		n = 1
	}
	count := n * sampleRate / 20
	if count > sampleRate {
		count = sampleRate
	}
	return count
}

func tone(n int) []int16 {
	out := make([]int16, n)
	for i := range out {
		sample := math.Sin(2 * math.Pi * toneHz * float64(i) / sampleRate)
		out[i] = int16(sample * 4000)
	}
	return out
}

func writeWAV(ctx context.Context, path string, pcm []int16, rate int) error {
	data := make([]byte, 44+len(pcm)*2)
	copy(data[0:], "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
	copy(data[8:], "WAVE")
	copy(data[12:], "fmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], uint32(rate))
	binary.LittleEndian.PutUint32(data[28:], uint32(rate*2))
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(len(pcm)*2))
	for i, sample := range pcm {
		binary.LittleEndian.PutUint16(data[44+2*i:], uint16(sample))
	}
	return providers.WriteOutput(ctx, path, func(f *os.File) error {
		_, err := f.Write(data)
		return err
	})
}
