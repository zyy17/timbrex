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
	if !strings.Contains(ciText, "make test") || !strings.Contains(ciText, "make lint") {
		t.Fatal("CI does not run make test and make lint")
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
