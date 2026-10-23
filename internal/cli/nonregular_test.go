//go:build darwin || linux

package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRejectNonregularInputs(t *testing.T) {
	for _, command := range []string{"check", "synth"} {
		for _, mode := range []string{"no-writer", "held-writer", "symlink"} {
			t.Run(command+"/"+mode, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				path := inputFIFO(t, dir, "script.tbx", mode, "Hello\n")
				args := []string{command, path}
				out := filepath.Join(dir, "out.wav")
				if command == "synth" {
					args = append(args, "--provider", "stub", "-o", out)
				}
				output, err := runInputProcess(t, dir, args...)
				if err == nil || !strings.Contains(output, "input must be a regular file") {
					t.Fatalf("err=%v output=%q", err, output)
				}
				if _, err := os.Stat(out); !os.IsNotExist(err) {
					t.Fatalf("unexpected output: %v", err)
				}
			})
		}
	}
	for _, config := range []string{"explicit", "default"} {
		for _, mode := range []string{"no-writer", "held-writer", "symlink"} {
			t.Run("config/"+config+"/"+mode, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				path := inputFIFO(t, dir, "timbrex.yaml", mode, "provider: stub\n")
				args := []string{"config"}
				if config == "explicit" {
					args = append(args, "--config", path)
				}
				output, err := runInputProcess(t, dir, args...)
				if err == nil || !strings.Contains(output, "input must be a regular file") {
					t.Fatalf("err=%v output=%q", err, output)
				}
			})
		}
	}
}

func TestCheckRegularInputs(t *testing.T) {
	for _, mode := range []string{"regular", "symlink", "empty"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "script.tbx")
			content := "Hello\n"
			if mode == "empty" {
				content = ""
			}
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if mode == "symlink" {
				link := filepath.Join(dir, "linked.tbx")
				if err := os.Symlink(path, link); err != nil {
					t.Fatal(err)
				}
				path = link
			}
			output, err := runInputProcess(t, dir, "check", path)
			if err != nil || strings.TrimSpace(output) != "ok" {
				t.Fatalf("err=%v output=%q", err, output)
			}
		})
	}
}

func inputFIFO(t *testing.T, dir, name, mode, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if mode == "symlink" {
		path = filepath.Join(dir, "target-"+name)
	}
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	switch mode {
	case "held-writer":
		f, err := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = f.Close() })
		if _, err := f.WriteString(content); err != nil {
			t.Fatal(err)
		}
	case "symlink":
		link := filepath.Join(dir, name)
		if err := os.Symlink(path, link); err != nil {
			t.Fatal(err)
		}
		return link
	}
	return path
}

func runInputProcess(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, append([]string{"-test.run=^TestInputHelperProcess$", "--"}, args...)...)
	cmd.Dir = dir
	cmd.Env = inputProcessEnv(os.Environ())
	cmd.WaitDelay = time.Second
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("command did not finish within 5 seconds: %v; output=%q", args, output)
	}
	return string(output), err
}

// Keep runtime controls without inheriting provider settings or credentials.
// GOCOVERDIR lets child exit hooks add counters to the parent's coverage profile.
func inputProcessEnv(environ []string) []string {
	env := []string{"GO_WANT_TIMBREX_INPUT_HELPER=1"}
	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		switch name {
		case "PATH", "TMPDIR", "TMP", "TEMP", "GOCOVERDIR", "GOCOVERDEBUG",
			"GODEBUG", "GOMAXPROCS", "GOMEMLIMIT", "GOTRACEBACK", "GORACE":
			env = append(env, entry)
		}
	}
	return env
}

func TestInputProcessEnv(t *testing.T) {
	runtime := []string{
		"PATH=/bin", "TMPDIR=/tmp", "TMP=/tmp", "TEMP=/tmp",
		"GOCOVERDIR=/tmp/coverage", "GOCOVERDEBUG=1", "GODEBUG=asyncpreemptoff=1",
		"GOMAXPROCS=2", "GOMEMLIMIT=1GiB", "GOTRACEBACK=all", "GORACE=halt_on_error=1",
	}
	environ := append(slices.Clone(runtime),
		"TIMBREX_PROVIDER=openai", "OPENAI_API_KEY=secret", "ELEVENLABS_API_KEY=secret",
		"MINIMAX_API_KEY=secret", "AZURE_SPEECH_KEY=secret", "QWEN_API_KEY=secret",
		"DASHSCOPE_API_KEY=secret", "GEMINI_API_KEY=secret", "FISH_API_KEY=secret",
		"OTHER_API_KEY=secret", "GOCOVERDIR_API_KEY=secret", "GO_WANT_TIMBREX_INPUT_HELPER=0",
	)
	want := append([]string{"GO_WANT_TIMBREX_INPUT_HELPER=1"}, runtime...)
	if got := inputProcessEnv(environ); !slices.Equal(got, want) {
		t.Fatalf("environment = %q, want %q", got, want)
	}
}

func TestInputHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_TIMBREX_INPUT_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			if err := Run(os.Args[i+1:]); err != nil {
				fmt.Fprintln(os.Stderr, FormatError(err))
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
	t.Fatal("missing helper arguments")
}
