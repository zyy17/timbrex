package stub

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func TestWritesWAV(t *testing.T) {
	p, err := providers.Get("stub")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.wav")
	doc := &ir.Document{Utterances: []ir.Utterance{{
		Speaker: "lin",
		Nodes:   []ir.Node{{Kind: ir.Text, Text: "Hello"}},
	}}}
	res, err := p.Synthesize(context.Background(), doc, providers.Options{Out: out})
	if err != nil {
		t.Fatal(err)
	}
	if res.Provider != "stub" || res.Path != out {
		t.Fatalf("result = %+v", res)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 44+2000*2 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" ||
		binary.LittleEndian.Uint32(b[24:]) != 8000 || binary.LittleEndian.Uint32(b[28:]) != 16000 ||
		binary.LittleEndian.Uint16(b[22:]) != 1 || binary.LittleEndian.Uint16(b[34:]) != 16 ||
		binary.LittleEndian.Uint32(b[40:]) != 4000 {
		t.Fatalf("wrong 8 kHz mono S16LE tone WAV: %d bytes", len(b))
	}
	if binary.LittleEndian.Uint16(b[44:]) != 0 || binary.LittleEndian.Uint16(b[46:]) != 1354 {
		t.Fatal("stub tone samples changed")
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o177 != 0 {
		t.Fatalf("stub output permissions are not private: %04o", info.Mode().Perm())
	}
}

func TestOverwriteReplacesWav(t *testing.T) {
	p, err := providers.Get("stub")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.wav")
	if err := os.WriteFile(out, []byte("KEEP-WAV"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Synthesize(context.Background(), &ir.Document{}, providers.Options{Out: out}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 12 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		t.Fatalf("not replaced: %q", b[:min(12, len(b))])
	}
}

func TestOverwriteKeepsPreviousWavOnFailure(t *testing.T) {
	p, err := providers.Get("stub")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "out.wav")
	const keep = "KEEP-WAV"
	if err := os.WriteFile(out, []byte(keep), 0o644); err != nil {
		t.Fatal(err)
	}
	var old syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &old); err != nil {
		t.Fatal(err)
	}
	lim := old
	lim.Cur = 32
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &lim); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &old); err != nil {
			t.Errorf("restore rlimit: %v", err)
		}
	}()
	_, err = p.Synthesize(context.Background(), &ir.Document{}, providers.Options{Out: out})
	if err == nil {
		t.Fatal("expected a failed write")
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != keep {
		t.Fatalf("previous wav = %q", b)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".timbrex-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("partial files = %v", matches)
	}
}

func TestCanceledSynthesisKeepsExistingOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.wav")
	if err := os.WriteFile(path, []byte("KEEP"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := (provider{}).Synthesize(ctx, &ir.Document{}, providers.Options{Out: path})
	if !errors.Is(err, context.Canceled) || result != (providers.Result{}) {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "KEEP" {
		t.Fatalf("output=%q error=%v", data, err)
	}
	files, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(files) != 1 {
		t.Fatalf("temporary files remain: %v, %v", files, err)
	}
}

func TestConcurrencyRejectsWithoutReplacingOutput(t *testing.T) {
	for _, concurrency := range []int{-1, 2, 8, 9} {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.wav")
		if err := os.WriteFile(path, []byte("KEEP"), 0o600); err != nil {
			t.Fatal(err)
		}
		result, err := (provider{}).Synthesize(context.Background(), nil, providers.Options{Out: path, Concurrency: concurrency})
		if err == nil || !strings.Contains(err.Error(), "concurrency") || result != (providers.Result{}) {
			t.Fatalf("concurrency=%d result=%+v error=%v", concurrency, result, err)
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "KEEP" {
			t.Fatalf("output=%q error=%v", data, err)
		}
		files, err := os.ReadDir(dir)
		if err != nil || len(files) != 1 {
			t.Fatalf("temporary files=%v error=%v", files, err)
		}
	}
}
