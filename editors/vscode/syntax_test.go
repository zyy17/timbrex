package syntax

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"
)

// The TextMate grammar has to close nested tags the way the parser does.
// ]] is a literal bracket only inside a single open tag.
func TestGrammarNestedTagClose(t *testing.T) {
	path := filepath.Join("syntaxes", "tbx.tmLanguage.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Repository map[string]struct {
			Begin    string `yaml:"begin"`
			End      string `yaml:"end"`
			Patterns []struct {
				Include string `yaml:"include"`
			} `yaml:"patterns"`
		} `yaml:"repository"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	outer := doc.Repository["tag"]
	nested := doc.Repository["tag-nested"]
	content := doc.Repository["tag-content"]
	if outer.End == "" || nested.End == "" {
		t.Fatalf("missing tag rules")
	}
	if outer.End == nested.End {
		t.Fatal("nested tag uses the outer end rule, so ]] swallows the closer")
	}
	found := false
	for _, p := range content.Patterns {
		if p.Include == "#tag-nested" {
			found = true
		}
		if p.Include == "#tag" {
			t.Fatal("tag content includes #tag, which does not close before a following ]")
		}
	}
	if !found {
		t.Fatal("tag content does not include #tag-nested")
	}
}
