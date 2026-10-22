package syntax

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"
)

var updateGrammar = flag.Bool("update", false, "regenerate the VS Code JSON grammar from YAML")

func TestPackagedGrammar(t *testing.T) {
	raw, err := os.ReadFile("syntaxes/tbx.tmLanguage.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	want, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	want = append(want, '\n')
	path := "syntaxes/tbx.tmLanguage.json"
	if *updateGrammar {
		if err := os.WriteFile(path, want, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("run go test ./editors/vscode -update to regenerate the JSON grammar")
	}
	raw, err = os.ReadFile("package.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Contributes struct{ Grammars []struct{ Path string } }
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Contributes.Grammars) != 1 || manifest.Contributes.Grammars[0].Path != "./"+path {
		t.Fatal("extension must load the generated JSON grammar")
	}
}

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

func TestGrammarNamedArgumentKeysMatchIdent(t *testing.T) {
	raw, err := os.ReadFile("syntaxes/tbx.tmLanguage.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`([A-Za-z][A-Za-z0-9_.-]*)(=)`)) {
		t.Fatal("named-argument keys must use the parser ident, not a leading underscore")
	}
}
