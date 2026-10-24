package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/parse"
)

var (
	tbxBlocks     = regexp.MustCompile("(?ms)^```tbx\\r?\\n(.*?)^```[ \\t]*$")
	tbxOpen       = regexp.MustCompile("(?m)^```tbx\\r?\\n")
	mdLinks       = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	atxHeadings   = regexp.MustCompile(`(?m)^(#{1,6}) `)
	fenceLine     = regexp.MustCompile("^(`{3,})([^`]*)$")
	inlineCode    = regexp.MustCompile("`([^`]*)`")
	strongLabels  = regexp.MustCompile(`\*\*[^*\n]+[：:]\*\*(.)`)
	skipBilingual = map[string]bool{
		// Portable agent instructions stay English; human guides remain bilingual.
		"AGENTS.md":                              true,
		"skills/timbrex/SKILL.md":                true,
		"skills/timbrex/references/authoring.md": true,
		"skills/timbrex/references/running.md":   true,
	}
	fenceLanguages = map[string]bool{
		"go": true, "bash": true, "tbx": true, "yaml": true, "json": true, "text": true, "mermaid": true,
	}
)

func TestDocumentationExamplesAndLinks(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != "." && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules") {
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
			if path != "." && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules") {
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

func TestMarkdownFencesAndTables(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != "." && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		open := ""
		for i, line := range strings.Split(string(data), "\n") {
			line = strings.TrimRight(line, "\r")
			m := fenceLine.FindStringSubmatch(line)
			if m != nil {
				lang := strings.TrimSpace(m[2])
				if open != "" {
					if lang != "" {
						t.Errorf("%s:%d: closing fence must be %s alone", path, i+1, open)
					}
					if m[1] != open {
						t.Errorf("%s:%d: fence closer %q does not match opener %q", path, i+1, m[1], open)
					}
					open = ""
					continue
				}
				if lang == "" {
					t.Errorf("%s:%d: fenced block needs a language tag", path, i+1)
				} else if !fenceLanguages[lang] {
					t.Errorf("%s:%d: unknown fence language %q", path, i+1, lang)
				}
				open = m[1]
				continue
			}
			if open != "" {
				continue
			}
			if hasBrokenStrongLabel(inlineCode.ReplaceAllString(line, "")) {
				t.Errorf("%s:%d: move the label's colon outside **; adjacent text prevents GFM from closing the emphasis", path, i+1)
			}
			if !strings.HasPrefix(strings.TrimSpace(line), "|") {
				continue
			}
			for _, span := range inlineCode.FindAllStringSubmatch(line, -1) {
				if unescapedPipe(span[1]) {
					t.Errorf("%s:%d: unescaped | inside table code %s", path, i+1, span[0])
				}
			}
		}
		if open != "" {
			t.Errorf("%s: unclosed fenced block", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// GFM cannot close emphasis after punctuation when a letter follows it directly.
// This guards the label pattern that previously rendered literal ** in Chinese.
func hasBrokenStrongLabel(line string) bool {
	for _, match := range strongLabels.FindAllStringSubmatch(line, -1) {
		next, _ := utf8.DecodeRuneInString(match[1])
		if !unicode.IsSpace(next) && !unicode.IsPunct(next) && !unicode.IsSymbol(next) {
			return true
		}
	}
	return false
}

func TestMarkdownStrongLabelBoundaries(t *testing.T) {
	for _, line := range []string{"**音色系列：**包括标准", "**音色：**SSML", "**Voice:**text"} {
		if !hasBrokenStrongLabel(line) {
			t.Errorf("missed broken strong label: %s", line)
		}
	}
	for _, line := range []string{"**音色系列**：包括标准", "**Voice:** text", "**音色：**（标准）"} {
		if hasBrokenStrongLabel(line) {
			t.Errorf("rejected valid strong label: %s", line)
		}
	}
}

func unescapedPipe(s string) bool {
	for i := range len(s) {
		if s[i] == '|' && (i == 0 || s[i-1] != '\\') {
			return true
		}
	}
	return false
}

func TestSpecVersionLabel(t *testing.T) {
	for path, title := range map[string]string{
		"docs/spec.md":       "# DSL Specification v1\n",
		"docs/spec.zh-cn.md": "# DSL 规范 v1\n",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(data), title) {
			t.Errorf("%s: expected title %q", path, strings.TrimSpace(title))
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
