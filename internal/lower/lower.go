// Package lower flattens an AST into an IR document.
package lower

import (
	"strconv"
	"strings"

	"github.com/zyy17/timbrex/internal/ast"
	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/vocab"
)

// Script converts a parsed script into utterances.
// Comments are dropped. Narration is spoken by "narrator" unless
// front matter narrator: names another speaker.
// Call check.Apply first so free-form tags already carry their note.
func Script(script *ast.Script) *ir.Document {
	doc := &ir.Document{Provider: script.Front.Provider}
	narrator := script.Front.Narrator
	if narrator == "" {
		narrator = "narrator"
	}
	for _, block := range script.Blocks {
		switch block.Kind {
		case ast.Comment:
			continue
		case ast.Fence:
			doc.Utterances = append(doc.Utterances, ir.Utterance{
				Nodes: []ir.Node{{Kind: ir.Raw, Name: block.Info, Text: block.Raw, Scope: "span"}},
			})
		case ast.Scene:
			doc.Utterances = append(doc.Utterances, ir.Utterance{Nodes: flatten(block.Nodes, "scene")})
		case ast.Turn:
			doc.Utterances = append(doc.Utterances, ir.Utterance{
				Speaker: block.Speaker,
				Nodes:   flatten(block.Nodes, "turn"),
			})
		default:
			doc.Utterances = append(doc.Utterances, ir.Utterance{
				Speaker: narrator,
				Nodes:   flatten(block.Nodes, "turn"),
			})
		}
	}
	return doc
}

func flatten(nodes []ast.Node, blockScope string) []ir.Node {
	var out []ir.Node
	for _, node := range nodes {
		if node.Kind == ast.Text {
			if node.Text != "" {
				out = append(out, ir.Node{Kind: ir.Text, Text: node.Text})
			}
			continue
		}
		name := canonical(node.Name)
		if node.Span {
			out = append(out, ir.Node{
				Kind: ir.SpanStart, Name: name, Args: formatArgs(node.Args), Note: node.Note, Scope: "span",
			})
			out = append(out, flatten(node.Content, blockScope)...)
			out = append(out, ir.Node{Kind: ir.SpanEnd, Name: name, Scope: "span"})
			continue
		}
		kind := ir.Event
		scope := "here"
		spec, ok := vocab.Lookup(name)
		if ok && spec.Kind == vocab.Turn {
			// overlap and gap describe a turn, including the next turn when
			// they sit alone on a scene line. They are not scene state.
			scope = "turn"
		} else if node.Free || (ok && spec.Kind == vocab.State) {
			kind = ir.State
			scope = blockScope
		} else if blockScope == "scene" {
			scope = "scene"
		}
		out = append(out, ir.Node{
			Kind: kind, Name: name, Args: formatArgs(node.Args), Note: node.Note, Scope: scope,
		})
	}
	return out
}

func canonical(name string) string {
	if spec, ok := vocab.Lookup(name); ok {
		return spec.Name
	}
	return name
}

func formatArgs(args []ast.Arg) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		raw := arg.Raw
		if arg.Kind == "quoted" {
			raw = strconv.Quote(arg.Raw)
		}
		if arg.Key != "" {
			parts = append(parts, arg.Key+"="+raw)
			continue
		}
		parts = append(parts, raw)
	}
	return strings.Join(parts, " ")
}
