package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/parse"
)

func TestDocumentationExamplesAndLinks(t *testing.T) {
	blocks := regexp.MustCompile("(?ms)^```tbx\\r?\\n(.*?)^```[ \\t]*$")
	links := regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") && path != "." {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" && filepath.Ext(path) != ".tbx" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".tbx" {
			checkExample(t, path, string(data))
			return nil
		}
		for _, block := range blocks.FindAllSubmatch(data, -1) {
			checkExample(t, path, string(block[1]))
		}
		for _, link := range links.FindAllSubmatch(data, -1) {
			target := string(link[1])
			if strings.Contains(target, "://") || strings.HasPrefix(target, "#") {
				continue
			}
			target, _, _ = strings.Cut(target, "#")
			if _, err := os.Stat(filepath.Join(filepath.Dir(path), target)); err != nil {
				t.Errorf("%s: broken link %q: %v", path, target, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func checkExample(t *testing.T, file, src string) {
	t.Helper()
	script, err := parse.Parse(file, src)
	if err != nil {
		t.Errorf("example: %v", err)
		return
	}
	strict := script.Front.StrictTags != nil && *script.Front.StrictTags
	for _, d := range check.Apply(script, strict) {
		if d.Severity == "error" {
			t.Errorf("example: %s", d.Error())
		}
	}
}
