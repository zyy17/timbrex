package audioexport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Re-exec the test binary as a controllable FFmpeg, without a shell or compiler.
func init() {
	if filepath.Base(os.Args[0]) != "ffmpeg" {
		return
	}
	args := os.Args[1:]
	mode := filepath.Base(filepath.Dir(os.Args[0]))
	if os.Getenv("OPENAI_API_KEY") != "" || os.Getenv("FFREPORT") != "" {
		os.Exit(91)
	}
	_ = os.WriteFile(filepath.Join(filepath.Dir(os.Args[0]), "args"), []byte(strings.Join(args, "\n")), 0o600)
	if mode == "cancel-preflight" {
		_ = os.WriteFile(filepath.Join(filepath.Dir(os.Args[0]), "started"), []byte(strconv.Itoa(os.Getpid())), 0o600)
		for {
			time.Sleep(time.Hour)
		}
	}
	if mode == "missing-encoder" {
		fmt.Fprintln(os.Stderr, "Unknown encoder 'libmp3lame'")
		os.Exit(1)
	}
	input := ""
	for i, arg := range args {
		if arg == "-i" && i+1 < len(args) {
			input = args[i+1]
		}
	}
	if len(args) == 0 {
		os.Exit(92)
	}
	output := args[len(args)-1]
	if filepath.Base(input) == "source.wav" {
		switch mode {
		case "fail":
			_ = os.WriteFile(output, []byte("ID3partial"), 0o600)
			fmt.Fprintln(os.Stderr, "encode failed")
			os.Exit(1)
		case "empty":
			os.Exit(0)
		case "wrong":
			_ = os.WriteFile(output, []byte("RIFFnotMP3"), 0o600)
			os.Exit(0)
		case "flood":
			_, _ = os.Stderr.Write(bytes.Repeat([]byte("diagnostic"), 1<<20))
			os.Exit(1)
		case "cancel":
			_ = os.WriteFile(filepath.Join(filepath.Dir(os.Args[0]), "started"), []byte(strconv.Itoa(os.Getpid())), 0o600)
			for {
				time.Sleep(time.Hour)
			}
		}
	}
	_ = os.WriteFile(output, []byte("ID3fake-success"), 0o600)
	os.Exit(0)
}

func fakeFFmpeg(t *testing.T, mode string) string {
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
	return dir
}

func TestWAVDoesNotUseFFmpeg(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	out := filepath.Join(t.TempDir(), "out.WAV")
	e, err := Prepare(context.Background(), out)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if e.InputPath() != out || e.dir != "" || e.ffmpeg != "" {
		t.Fatalf("WAV should be direct: %+v", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := e.Finish(ctx); err != nil {
		t.Fatalf("already committed WAV should succeed: %v", err)
	}
}

func TestFormatAndPathPreflight(t *testing.T) {
	dir := t.TempDir()
	for _, ext := range []string{".raw", ".pcm", ".m4a", ".ogg", ""} {
		if _, err := Prepare(context.Background(), filepath.Join(dir, "out"+ext)); err == nil {
			t.Fatalf("accepted extension %q", ext)
		}
	}
	if err := ValidatePath(filepath.Join(dir, "missing", "out.wav")); err == nil {
		t.Fatal("accepted missing parent")
	}
	t.Setenv("PATH", t.TempDir())
	if _, err := Prepare(context.Background(), filepath.Join(dir, "out.mp3")); err == nil || !strings.Contains(err.Error(), "installed FFmpeg") {
		t.Fatalf("missing FFmpeg: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("preflight leaked files: %v %v", entries, err)
	}
}

func TestEncoderPreflightFailureCleans(t *testing.T) {
	fakeFFmpeg(t, "missing-encoder")
	dir := t.TempDir()
	_, err := Prepare(context.Background(), filepath.Join(dir, "out.mp3"))
	if err == nil || !strings.Contains(err.Error(), "Unknown encoder") {
		t.Fatalf("error=%v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("preflight leaked files: %v %v", entries, err)
	}
}

func TestEncoderFailurePreservesDestination(t *testing.T) {
	for _, mode := range []string{"fail", "empty", "wrong", "flood"} {
		t.Run(mode, func(t *testing.T) {
			fakeFFmpeg(t, mode)
			out := filepath.Join(t.TempDir(), "out.mp3")
			if err := os.WriteFile(out, []byte("original"), 0o600); err != nil {
				t.Fatal(err)
			}
			e, err := Prepare(context.Background(), out)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(e.InputPath(), probeWAV(), 0o600); err != nil {
				t.Fatal(err)
			}
			err = e.Finish(context.Background())
			if err == nil || len(err.Error()) > diagnosticLimit+512 {
				t.Fatalf("expected bounded failure, got %v", err)
			}
			e.Close()
			assertOriginal(t, out)
			if _, err := os.Stat(e.dir); !os.IsNotExist(err) {
				t.Fatalf("staging directory survived: %v", err)
			}
		})
	}
}

func TestEncoderCancellationKillsAndWaits(t *testing.T) {
	bin := fakeFFmpeg(t, "cancel")
	out := filepath.Join(t.TempDir(), "out.mp3")
	if err := os.WriteFile(out, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	e, err := Prepare(context.Background(), out)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- e.Finish(ctx) }()
	pid := waitForPID(t, filepath.Join(bin, "started"))
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("encoder cancellation did not join")
	}
	process, err := os.FindProcess(pid)
	if err == nil {
		defer func() { _ = process.Release() }()
		if err := process.Signal(syscall.Signal(0)); err == nil {
			t.Fatal("encoder still alive after Finish returned")
		}
	}
	e.Close()
	assertOriginal(t, out)
}

func TestEncoderArgumentsPrivateAndLocal(t *testing.T) {
	bin := fakeFFmpeg(t, "success")
	t.Setenv("OPENAI_API_KEY", "must-not-inherit")
	t.Setenv("FFREPORT", "file=must-not-create")
	out := filepath.Join(t.TempDir(), "https:evil;$(touch pwned).MP3")
	e, err := Prepare(context.Background(), out)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	info, err := os.Stat(e.dir)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("private directory: %v %v", info, err)
	}
	if err := e.Finish(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(bin, "args"))
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Split(string(data), "\n")
	for _, pair := range [][2]string{{"-protocol_whitelist", "file"}, {"-i", e.InputPath()}, {"-write_xing", "1"}, {"-f", "mp3"}} {
		found := false
		for i := range len(args) - 1 {
			found = found || (args[i] == pair[0] && args[i+1] == pair[1])
		}
		if !found {
			t.Fatalf("missing %v in %v", pair, args)
		}
	}
	if !slices.Contains(args, "-nostdin") || slices.Contains(args, out) || !filepath.IsAbs(args[len(args)-1]) {
		t.Fatalf("unsafe arguments: %v", args)
	}
	info, err = os.Stat(out)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("private output: %v %v", info, err)
	}
}

func TestEncoderCancellationBeforeCommit(t *testing.T) {
	fakeFFmpeg(t, "success")
	out := filepath.Join(t.TempDir(), "out.mp3")
	if err := os.WriteFile(out, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	e, err := Prepare(context.Background(), out)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := e.Finish(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	assertOriginal(t, out)
}

func TestDiagnosticsDrainBound(t *testing.T) {
	var d diagnostics
	input := bytes.Repeat([]byte("a"), 10*diagnosticLimit)
	for range 3 {
		n, err := d.Write(input)
		if n != len(input) || err != nil || len(d.data) != diagnosticLimit {
			t.Fatalf("write=%d %v stored=%d", n, err, len(d.data))
		}
	}
}

func assertOriginal(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "original" {
		t.Fatalf("old output changed: %q %v", data, err)
	}
}

func waitForPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil {
			pid, err := strconv.Atoi(string(data))
			if err != nil {
				t.Fatal(err)
			}
			return pid
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("encoder did not start")
	return 0
}

func TestPreflightCancellationCleans(t *testing.T) {
	bin := fakeFFmpeg(t, "cancel-preflight")
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		_, err := Prepare(ctx, filepath.Join(dir, "out.mp3"))
		finished <- err
	}()
	waitForPID(t, filepath.Join(bin, "started"))
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("preflight cancellation did not join")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("preflight leaked files: %v %v", entries, err)
	}
}

func TestRenameFailureDoesNotPublish(t *testing.T) {
	fakeFFmpeg(t, "success")
	dir := t.TempDir()
	out := filepath.Join(dir, "out.mp3")
	e, err := Prepare(context.Background(), out)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	// A destination can become unavailable after preflight.
	if err := os.Mkdir(out, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := e.Finish(context.Background()); err == nil {
		t.Fatal("unexpected publication success")
	}
	info, err := os.Stat(out)
	if err != nil || !info.IsDir() {
		t.Fatalf("destination changed: %v %v", info, err)
	}
}
