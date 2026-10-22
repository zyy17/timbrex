package parse

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/adrg/frontmatter"
	"go.yaml.in/yaml/v3"

	"github.com/zyy17/timbrex/internal/ast"
)

// grammarVersion is the only timbrex: value this parser accepts.
const grammarVersion = 1

type frontDoc struct {
	Timbrex    *int      `yaml:"timbrex"`
	Provider   string    `yaml:"provider"`
	Model      string    `yaml:"model"`
	Voice      string    `yaml:"voice"`
	Lang       string    `yaml:"lang"`
	StrictTags *bool     `yaml:"strict_tags"`
	Narrator   string    `yaml:"narrator"`
	Cast       *castYAML `yaml:"cast"`
}

type frontCast struct {
	Name   string            `yaml:"name"`
	Voices map[string]string `yaml:"voices"`
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
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	return decodeFront(data, &c.Members)
}

// yamlFence is the only front matter form the grammar allows.
var yamlFence = frontmatter.NewFormat("---", "---", decodeFront)

func decodeFront(data []byte, value any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(value); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("front matter must contain one YAML document")
	}
	return nil
}

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
	if doc.Timbrex != nil && *doc.Timbrex != grammarVersion {
		return ast.Front{}, "", &Error{
			File: file,
			Line: 1,
			Col:  1,
			Msg:  "unsupported grammar version",
		}
	}
	front := ast.Front{
		Provider:   doc.Provider,
		Model:      doc.Model,
		Voice:      doc.Voice,
		Lang:       doc.Lang,
		StrictTags: doc.StrictTags,
		Narrator:   strings.TrimSpace(doc.Narrator),
	}
	if doc.Timbrex != nil {
		front.Timbrex = *doc.Timbrex
	}
	if front.Narrator != "" && identLen(front.Narrator) != len(front.Narrator) {
		return ast.Front{}, "", &Error{File: file, Line: 1, Col: 1, Msg: "narrator must be an ASCII identifier"}
	}
	if doc.Cast != nil {
		front.CastAuto = doc.Cast.Auto
		if doc.Cast.Members != nil {
			front.Cast = make(map[string]ast.Cast, len(doc.Cast.Members))
			for id, member := range doc.Cast.Members {
				if id == "" || identLen(id) != len(id) {
					return ast.Front{}, "", &Error{File: file, Line: 1, Col: 1, Msg: fmt.Sprintf("invalid cast identifier %q", id)}
				}
				front.Cast[id] = ast.Cast{Name: member.Name, Voices: member.Voices}
			}
		}
	}
	return front, body, nil
}
