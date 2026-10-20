package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatErrorKeepsParseLocation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.tbx")
	if err := os.WriteFile(path, []byte("Hello [slow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"check", path}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	got := FormatError(err)
	if !strings.HasPrefix(got, path+":1:7: error[V012]:") {
		t.Fatalf("format = %q", got)
	}
	bad := filepath.Join(dir, "batch.tbx")
	if err := os.WriteFile(bad, []byte("[thoughtfully]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = run([]string{"check", "--strict-tags", bad}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if FormatError(err) != "error: check failed" {
		t.Fatalf("check = %q", FormatError(err))
	}
	missing := run([]string{"check", filepath.Join(dir, "missing.tbx")}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if !strings.HasPrefix(FormatError(missing), "error: ") {
		t.Fatalf("missing = %q", FormatError(missing))
	}
}

func TestVersion(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"version"}, strings.NewReader(""), &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "timbrex "+version) {
		t.Fatalf("out = %q", out.String())
	}
}

func TestCheckExample(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"check", "../../examples/hello.tbx"}, strings.NewReader(""), &out, &out)
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
	err := run([]string{"check", path}, strings.NewReader(""), &out, &out)
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
	err := run([]string{"synth", path, "-o", outPath}, strings.NewReader(""), &out, &out)
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
	err := run([]string{"check", "--strict-tags", path}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "check failed") {
		t.Fatalf("err = %v", err)
	}
	var out bytes.Buffer
	err = run([]string{"check", "--no-strict-tags", path}, strings.NewReader(""), &out, &bytes.Buffer{})
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
	err := run([]string{"check", strictPath}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "check failed") {
		t.Fatalf("strict err = %v", err)
	}
	err = run([]string{"check", "--no-strict-tags", strictPath}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("flag overrides front matter: %v", err)
	}

	provPath := filepath.Join(dir, "prov.tbx")
	if err := os.WriteFile(provPath, []byte("---\nprovider: missing\n---\nHello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = run([]string{"synth", provPath}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("provider err = %v", err)
	}
	wav := filepath.Join(dir, "prov.wav")
	var out bytes.Buffer
	err = run([]string{"synth", "--provider", "stub", provPath}, strings.NewReader(""), &out, &bytes.Buffer{})
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
	err := run([]string{"check", path}, strings.NewReader(""), &stdout, &stderr)
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

func TestSynthUnknownProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hi.tbx")
	if err := os.WriteFile(path, []byte("Hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"synth", path, "--provider", "missing"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("err = %v", err)
	}
}
