package parse

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/zyy17/timbrex/internal/ast"
)

// grammarVersion is the only timbrex: value this parser accepts.
const grammarVersion = 1

type frontDoc struct {
	Timbrex    *frontVersion `yaml:"timbrex"`
	Provider   string        `yaml:"provider"`
	Model      string        `yaml:"model"`
	Voice      string        `yaml:"voice"`
	Lang       string        `yaml:"lang"`
	StrictTags *bool         `yaml:"strict_tags"`
	Narrator   string        `yaml:"narrator"`
	Cast       *castYAML     `yaml:"cast"`
}

// frontVersion rejects YAML floats before they can be truncated into an int.
type frontVersion int

func (v *frontVersion) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode || value.ShortTag() != "!!int" {
		return fmt.Errorf("timbrex must be an integer")
	}
	var version int
	if err := value.Decode(&version); err != nil {
		return err
	}
	*v = frontVersion(version)
	return nil
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
	// Resolve aliases and merges on the original graph, retaining yaml.v3's
	// whole-cast alias limits. Node.Decode has no KnownFields option.
	var members map[string]map[string]any
	if err := value.Decode(&members); err != nil {
		return err
	}
	for _, member := range members {
		for key := range member {
			if key != "name" && key != "voices" {
				return fmt.Errorf("unknown cast field %q", key)
			}
		}
	}
	return value.Decode(&c.Members)
}

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
	opening, rest, ok := strings.Cut(src, "\n")
	if !ok || !yamlFenceLine(opening) {
		return ast.Front{}, src, nil
	}
	metadata, body, closed := closeFrontMatter(rest)
	if !closed {
		return ast.Front{}, "", &Error{File: file, Line: 1, Col: 1, Msg: "unclosed front matter"}
	}
	var doc frontDoc
	if err := decodeFront([]byte(metadata), &doc); err != nil {
		return ast.Front{}, "", &Error{File: file, Line: 1, Col: 1, Msg: "front matter: " + err.Error()}
	}
	if doc.Timbrex != nil && int(*doc.Timbrex) != grammarVersion {
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
		front.Timbrex = int(*doc.Timbrex)
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

// closeFrontMatter preserves indentation: an indented --- can be YAML scalar
// content, not a closing delimiter. The body remains a suffix for line offsets.
func closeFrontMatter(src string) (metadata, body string, closed bool) {
	for offset := 0; offset <= len(src); {
		line, rest, newline := strings.Cut(src[offset:], "\n")
		if yamlFenceLine(line) {
			return src[:offset], rest, true
		}
		if !newline {
			break
		}
		offset += len(line) + 1
	}
	return "", "", false
}

func yamlFenceLine(line string) bool {
	return strings.TrimRight(line, " \t") == "---"
}
