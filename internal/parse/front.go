package parse

import (
	"fmt"
	"strings"

	"github.com/adrg/frontmatter"
	"go.yaml.in/yaml/v3"

	"github.com/zyy17/timbrex/internal/ast"
)

// grammarVersion is the only timbrex: value this parser accepts.
const grammarVersion = 1

type frontDoc struct {
	Timbrex    int       `yaml:"timbrex"`
	Provider   string    `yaml:"provider"`
	Lang       string    `yaml:"lang"`
	StrictTags *bool     `yaml:"strict_tags"`
	Narrator   string    `yaml:"narrator"`
	Cast       *castYAML `yaml:"cast"`
}

type frontCast struct {
	Name string `yaml:"name"`
}

type castYAML struct {
	Auto    bool
	Members map[string]frontCast
}

func (c *castYAML) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		if value.Value == "auto" {
			c.Auto = true
			return nil
		}
		return fmt.Errorf("cast must be a map or auto")
	}
	return value.Decode(&c.Members)
}

// yamlFence is the only front matter form the grammar allows.
var yamlFence = frontmatter.NewFormat("---", "---", yaml.Unmarshal)

func splitFront(file, src string) (ast.Front, string, error) {
	if !strings.HasPrefix(src, "---\n") {
		return ast.Front{}, src, nil
	}
	var doc frontDoc
	rest, err := frontmatter.Parse(strings.NewReader(src), &doc, yamlFence)
	if err != nil {
		return ast.Front{}, "", &Error{File: file, Line: 1, Col: 1, Msg: "front matter: " + err.Error()}
	}
	body := string(rest)
	if body == src {
		return ast.Front{}, "", &Error{File: file, Line: 1, Col: 1, Msg: "unclosed front matter"}
	}
	if doc.Timbrex != 0 && doc.Timbrex != grammarVersion {
		return ast.Front{}, "", &Error{
			File: file,
			Line: 1,
			Col:  1,
			Msg:  "unsupported grammar version",
		}
	}
	front := ast.Front{
		Timbrex:    doc.Timbrex,
		Provider:   doc.Provider,
		Lang:       doc.Lang,
		StrictTags: doc.StrictTags,
		Narrator:   strings.TrimSpace(doc.Narrator),
	}
	if doc.Cast != nil {
		front.CastAuto = doc.Cast.Auto
		if doc.Cast.Members != nil {
			front.Cast = make(map[string]ast.Cast, len(doc.Cast.Members))
			for id, member := range doc.Cast.Members {
				front.Cast[id] = ast.Cast{Name: member.Name}
			}
		}
	}
	return front, body, nil
}
