package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		for _, prefix := range []string{"TIMBREX_", "OPENAI_", "ELEVENLABS_", "MINIMAX_", "AZURE_SPEECH_", "QWEN_", "DASHSCOPE_API_KEY", "GEMINI_"} {
			if strings.HasPrefix(name, prefix) {
				if err := os.Unsetenv(name); err != nil {
					panic(err)
				}
			}
		}
	}
	os.Exit(m.Run())
}

func TestFormatErrorKeepsParseLocation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.tbx")
	if err := os.WriteFile(path, []byte("Hello [slow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run(context.Background(), []string{"check", path}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	got := FormatError(err)
	if !strings.HasPrefix(got, path+":1:7: error[V012]:") {
		t.Fatalf("format = %q", got)
	}
	bad := filepath.Join(dir, "batch.tbx")
	if err := os.WriteFile(bad, []byte("[thoughtfully]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = run(context.Background(), []string{"check", "--strict-tags", bad}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if FormatError(err) != "error: check failed" {
		t.Fatalf("check = %q", FormatError(err))
	}
	missing := run(context.Background(), []string{"check", filepath.Join(dir, "missing.tbx")}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if !strings.HasPrefix(FormatError(missing), "error: ") {
		t.Fatalf("missing = %q", FormatError(missing))
	}
}

func TestVersion(t *testing.T) {
	var out, flag bytes.Buffer
	if err := run(context.Background(), []string{"version"}, strings.NewReader(""), &out, &out); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), []string{"--version"}, strings.NewReader(""), &flag, &flag); err != nil {
		t.Fatal(err)
	}
	want := "timbrex " + version + "\n"
	if out.String() != want || flag.String() != want {
		t.Fatalf("version = %q flag = %q", out.String(), flag.String())
	}
}

func TestCheckExample(t *testing.T) {
	var out bytes.Buffer
	err := run(context.Background(), []string{"check", "../../examples/hello.tbx"}, strings.NewReader(""), &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "ok" {
		t.Fatalf("out = %q", out.String())
	}
}

func TestCheckError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.tbx")
	if err := os.WriteFile(path, []byte("Hello [slow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := run(context.Background(), []string{"check", path}, strings.NewReader(""), &out, &out)
	if err == nil || !strings.Contains(err.Error(), "unclosed tag") {
		t.Fatalf("err = %v", err)
	}
}

func TestSynthStub(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hi.tbx")
	if err := os.WriteFile(path, []byte("Hello there.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "hi.wav")
	var out bytes.Buffer
	err := run(context.Background(), []string{"synth", path, "-o", outPath}, strings.NewReader(""), &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != outPath {
		t.Fatalf("out = %q", out.String())
	}
	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(b[:4]) != "RIFF" {
		t.Fatalf("header = %q", b[:4])
	}
}

func TestCheckStrict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "batch.tbx")
	body := "[act \"quietly\"]\n[thoughtfully]\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run(context.Background(), []string{"check", "--strict-tags", path}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "check failed") {
		t.Fatalf("err = %v", err)
	}
	var out bytes.Buffer
	err = run(context.Background(), []string{"check", "--no-strict-tags", path}, strings.NewReader(""), &out, &bytes.Buffer{})
	if err != nil || strings.TrimSpace(out.String()) != "ok" {
		t.Fatalf("err=%v out=%q", err, out.String())
	}
}

func TestFrontMatterStrictAndProvider(t *testing.T) {
	dir := t.TempDir()
	strictPath := filepath.Join(dir, "strict.tbx")
	body := "---\nstrict_tags: true\n---\n[thoughtfully]\n"
	if err := os.WriteFile(strictPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run(context.Background(), []string{"check", strictPath}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "check failed") {
		t.Fatalf("strict err = %v", err)
	}
	err = run(context.Background(), []string{"check", "--no-strict-tags", strictPath}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("flag overrides front matter: %v", err)
	}

	provPath := filepath.Join(dir, "prov.tbx")
	if err := os.WriteFile(provPath, []byte("---\nprovider: missing\n---\nHello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = run(context.Background(), []string{"synth", provPath}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("provider err = %v", err)
	}
	wav := filepath.Join(dir, "prov.wav")
	var out bytes.Buffer
	err = run(context.Background(), []string{"synth", "--provider", "stub", provPath}, strings.NewReader(""), &out, &bytes.Buffer{})
	if err != nil || strings.TrimSpace(out.String()) != wav {
		t.Fatalf("err=%v out=%q", err, out.String())
	}
	if _, err := os.Stat(wav); err != nil {
		t.Fatal(err)
	}
}

func TestCheckWarnsAndStillOk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "near.tbx")
	body := "[paus]\n[act \"caf\u00e9\"]\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{"check", path}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout.String()) != "ok" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "warning[W040]") || !strings.Contains(stderr.String(), "info[I050]") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestSynthCheckFailureWritesNoWav(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.tbx")
	if err := os.WriteFile(path, []byte("[thoughtfully]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wav := filepath.Join(dir, "bad.wav")
	if err := os.WriteFile(wav, []byte("KEEP"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run(context.Background(), []string{"synth", "--strict-tags", path}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "check failed") {
		t.Fatalf("err = %v", err)
	}
	b, err := os.ReadFile(wav)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "KEEP" {
		t.Fatalf("wav = %q", b)
	}
}

func TestSynthCanceledContextWritesNoWav(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hi.tbx")
	if err := os.WriteFile(path, []byte("Hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wav := filepath.Join(dir, "hi.wav")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := run(ctx, []string{"synth", path, "-o", wav}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected cancel")
	}
	if _, statErr := os.Stat(wav); !os.IsNotExist(statErr) {
		t.Fatalf("stat = %v", statErr)
	}
}

func TestStrictCastAutoCheckThenSynth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.tbx")
	body := "---\ncast: auto\n---\nlin: [act \"quietly\"] hello\nchen: [thoughtfully]\nlin: again\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{"check", "--strict-tags", path}, strings.NewReader(""), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "check failed") {
		t.Fatalf("err = %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	msg := stderr.String()
	if !strings.Contains(msg, "error[V041]") || !strings.Contains(msg, "thoughtfully") {
		t.Fatalf("stderr = %q", msg)
	}
	if strings.Contains(msg, "error[V041]") && strings.Contains(msg, "quietly") {
		t.Fatalf("explicit act rejected: %q", msg)
	}
	lin := strings.Count(msg, `cast auto recognized speaker "lin"`)
	chen := strings.Count(msg, `cast auto recognized speaker "chen"`)
	if lin != 1 || chen != 1 {
		t.Fatalf("speakers lin=%d chen=%d stderr=%q", lin, chen, msg)
	}
	wav := filepath.Join(dir, "scene.wav")
	err = run(context.Background(), []string{"synth", "--strict-tags", path, "-o", wav}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "check failed") {
		t.Fatalf("synth err = %v", err)
	}
	if _, statErr := os.Stat(wav); !os.IsNotExist(statErr) {
		t.Fatalf("wav stat = %v", statErr)
	}
}

func TestSynthUnknownProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hi.tbx")
	if err := os.WriteFile(path, []byte("Hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run(context.Background(), []string{"synth", path, "--provider", "missing"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("err = %v", err)
	}
}
