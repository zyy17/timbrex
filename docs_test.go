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

var (
	tbxBlocks     = regexp.MustCompile("(?ms)^```tbx\\r?\\n(.*?)^```[ \\t]*$")
	tbxOpen       = regexp.MustCompile("(?m)^```tbx\\r?\\n")
	mdLinks       = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	atxHeadings   = regexp.MustCompile(`(?m)^(#{1,6}) `)
	specVersion   = []byte("SPEC v0.3.4")
	skipBilingual = map[string]bool{
		"AGENTS.md": true,
	}
)

func TestDocumentationExamplesAndLinks(t *testing.T) {
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
		for _, block := range tbxBlocks.FindAllSubmatch(data, -1) {
			checkExample(t, path, string(block[1]))
		}
		for _, link := range mdLinks.FindAllSubmatch(data, -1) {
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

func TestBilingualDocPairs(t *testing.T) {
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
		if filepath.Ext(path) != ".md" || strings.HasSuffix(path, ".zh-cn.md") || skipBilingual[path] {
			return nil
		}
		zh := strings.TrimSuffix(path, ".md") + ".zh-cn.md"
		enData, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		zhData, err := os.ReadFile(zh)
		if err != nil {
			t.Errorf("%s: missing Chinese companion %s", path, zh)
			return nil
		}
		enHeads := headingLevels(enData)
		zhHeads := headingLevels(zhData)
		if enHeads != zhHeads {
			t.Errorf("%s vs %s: heading levels %q != %q", path, zh, enHeads, zhHeads)
		}
		enTbx := len(tbxOpen.FindAll(enData, -1))
		zhTbx := len(tbxOpen.FindAll(zhData, -1))
		if enTbx != zhTbx {
			t.Errorf("%s vs %s: ```tbx count %d != %d", path, zh, enTbx, zhTbx)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSpecVersionLabel(t *testing.T) {
	for _, path := range []string{"docs/spec.md", "docs/spec.zh-cn.md"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), string(specVersion)) {
			t.Errorf("%s: missing %s", path, specVersion)
		}
	}
}

func headingLevels(data []byte) string {
	matches := atxHeadings.FindAllSubmatch(data, -1)
	parts := make([]string, len(matches))
	for i, m := range matches {
		parts[i] = string(m[1])
	}
	return strings.Join(parts, " ")
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
