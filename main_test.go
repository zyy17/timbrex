package main

import (
	"os"
	"strings"
	"testing"
)

func TestCIMatchesMakefile(t *testing.T) {
	makeText := string(mustRead(t, "Makefile"))
	ciText := string(mustRead(t, ".github/workflows/ci.yml"))
	if !strings.Contains(makeText, "golangci-lint fmt --diff") {
		t.Fatal("make lint does not check formatting")
	}
	if !strings.Contains(makeText, "test: build") {
		t.Fatal("make test does not build the timbrex binary")
	}
	for _, target := range []string{"tidy", "test", "lint", "editor-deps", "test-editor"} {
		if !strings.Contains(ciText, "- run: make "+target+"\n") {
			t.Fatalf("CI does not run make %s", target)
		}
	}
	if !strings.Contains(makeText, "verify: tidy test lint test-editor\n") {
		t.Fatal("make verify does not match the offline CI checks")
	}
	if !strings.Contains(makeText, "test-editor:\n\tnpm --prefix editors/vscode test\n") ||
		!strings.Contains(makeText, "editor-deps:\n\tnpm --prefix editors/vscode ci --ignore-scripts\n") {
		t.Fatal("editor dependency installation and runtime tests are not separate targets")
	}
	if !strings.Contains(ciText, "v2.14.0") || !strings.Contains(ciText, "go-version-file: go.mod") {
		t.Fatal("CI Go or golangci-lint version drifted")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
