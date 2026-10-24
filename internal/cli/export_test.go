package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Test-only FFmpeg replacement. The CLI runs this binary directly, not a shell.
func init() {
	if filepath.Base(os.Args[0]) != "ffmpeg" {
		return
	}
	if strings.Contains(filepath.Dir(os.Args[0]), "missing-encoder") {
		fmt.Fprintln(os.Stderr, "Unknown encoder 'libmp3lame'")
		os.Exit(1)
	}
	if len(os.Args) < 2 {
		os.Exit(2)
	}
	_ = os.WriteFile(os.Args[len(os.Args)-1], []byte("ID3fake-preflight"), 0o600)
	os.Exit(0)
}

func exportTestFFmpeg(t *testing.T, mode string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), mode)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(exe, filepath.Join(dir, "ffmpeg")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestExportFailuresBeforeProviderRequests(t *testing.T) {
	for _, mode := range []string{"unknown-format", "missing-parent", "missing-ffmpeg", "missing-encoder", "source", "hardlink", "symlink", "directory", "invalid-last-segment"} {
		t.Run(mode, func(t *testing.T) {
			clearProviderDefaults(t)
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write([]byte{1, 0, 2, 0})
			}))
			defer srv.Close()
			t.Setenv("OPENAI_API_KEY", "test-key")
			t.Setenv("OPENAI_BASE_URL", srv.URL)
			exportTestFFmpeg(t, "success")
			dir := t.TempDir()
			path, out := filepath.Join(dir, "script.tbx"), filepath.Join(dir, "out.mp3")
			src := "First. [pause 10ms] Last.\n"
			if mode == "invalid-last-segment" {
				src = "First. [pause 10ms] [pitch +2st|Last.]\n"
			}
			if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "unknown-format":
				out = filepath.Join(dir, "out.ogg")
			case "missing-parent":
				out = filepath.Join(dir, "missing", "out.mp3")
			case "missing-ffmpeg":
				t.Setenv("PATH", t.TempDir())
			case "missing-encoder":
				exportTestFFmpeg(t, "missing-encoder")
			case "source":
				out = filepath.Join(dir, "script.mp3")
				if err := os.Rename(path, out); err != nil {
					t.Fatal(err)
				}
				path = out
			case "hardlink":
				if err := os.Link(path, out); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(path, out); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(out, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			err := run(context.Background(), []string{"synth", path, "--provider", "openai", "-o", out}, strings.NewReader(""), io.Discard, io.Discard)
			if err == nil || calls.Load() != 0 {
				t.Fatalf("error=%v provider calls=%d", err, calls.Load())
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != src {
				t.Fatalf("source changed: %q %v", data, err)
			}
			leftovers, err := filepath.Glob(filepath.Join(dir, ".timbrex-*"))
			if err != nil || len(leftovers) != 0 {
				t.Fatalf("staging leaked: %v %v", leftovers, err)
			}
		})
	}
}

func TestExportSynthesisFailurePreservesOldOutput(t *testing.T) {
	clearProviderDefaults(t)
	exportTestFFmpeg(t, "success")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "fixture failure", http.StatusBadRequest)
	}))
	defer srv.Close()
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", srv.URL)
	dir := t.TempDir()
	path, out := filepath.Join(dir, "script.tbx"), filepath.Join(dir, "out.mp3")
	if err := os.WriteFile(path, []byte("Hello."), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, []byte("old output"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	err := run(context.Background(), []string{"synth", path, "--provider", "openai", "-o", out}, strings.NewReader(""), &stdout, io.Discard)
	if err == nil || calls.Load() != 1 || stdout.Len() != 0 {
		t.Fatalf("error=%v calls=%d stdout=%q", err, calls.Load(), stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "old output" {
		t.Fatalf("old output changed: %q %v", data, err)
	}
	leftovers, err := filepath.Glob(filepath.Join(dir, ".timbrex-*"))
	if err != nil || len(leftovers) != 0 {
		t.Fatalf("staging leaked: %v %v", leftovers, err)
	}
}

func TestWAVCLIWithoutFFmpeg(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	path := filepath.Join(dir, "script.tbx")
	if err := os.WriteFile(path, []byte("Hello."), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := run(context.Background(), []string{"synth", path, "--provider", "stub"}, strings.NewReader(""), &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "script.wav")
	data, err := os.ReadFile(out)
	if err != nil || !bytes.HasPrefix(data, []byte("RIFF")) || stdout.String() != out+"\n" {
		t.Fatalf("default WAV changed: stdout=%q read=%v", stdout.String(), err)
	}
}

func TestCLIExportPreservesProviderContract(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		if os.Getenv("TIMBREX_TEST_FFMPEG") != "1" && errors.Is(err, exec.ErrNotFound) {
			t.Skip("optional real-codec test needs ffmpeg on PATH")
		}
		t.Fatalf("required real-codec FFmpeg: %v", err)
	}
	clearProviderDefaults(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := int(calls.Add(1))
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		voice := "en-US-FirstNeural"
		if call == 3 {
			voice = "en-US-SecondNeural"
		}
		assertRequest(t, "openai", r, body, call, voice)
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(contractPCM(call))
	}))
	defer srv.Close()
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", srv.URL)
	dir := t.TempDir()
	path, out := filepath.Join(dir, "scene.tbx"), filepath.Join(dir, "scene.FLAC")
	source := "---\nprovider: openai\ncast:\n  a: {voices: {openai: en-US-FirstNeural}}\n  b: {voices: {openai: en-US-SecondNeural}}\n---\na: [happy] One [pause 10ms] [speed 1.2|Two]\nb: Three\n"
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := run(ctx, []string{"synth", path, "-o", out}, strings.NewReader(""), &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 || stdout.String() != out+"\n" {
		t.Fatalf("calls=%d stdout=%q", calls.Load(), stdout.String())
	}
	decode := exec.CommandContext(ctx, ffmpeg, "-nostdin", "-v", "error", "-protocol_whitelist", "file", "-i", out, "-f", "s16le", "-c:a", "pcm_s16le", "pipe:1")
	decode.WaitDelay = time.Second
	pcm, err := decode.Output()
	if err != nil {
		t.Fatal(err)
	}
	want := append(contractPCM(1), make([]byte, 480)...)
	want = append(want, contractPCM(2)...)
	want = append(want, contractPCM(3)...)
	if !bytes.Equal(pcm, want) {
		t.Fatalf("FLAC changed composed PCM: got %x want %x", pcm, want)
	}
}
