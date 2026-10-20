// Package check applies vocabulary rules to a parsed script.
//
// The parser stays structural. Strict mode, closing tags, and note words
// are decided here so a batch run can reject an unregistered tag.
// A near miss in a single script is warning W040. Non-ASCII acting text is info I050.
package check

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/zyy17/timbrex/internal/ast"
	"github.com/zyy17/timbrex/internal/vocab"
)

// Diag is one check finding.
type Diag struct {
	File     string
	Line     int
	Col      int
	Severity string
	Code     string
	Msg      string
}

func (d Diag) Error() string {
	loc := fmt.Sprintf("%s:%d:%d", d.File, d.Line, d.Col)
	if d.Code == "" {
		return loc + ": " + d.Severity + ": " + d.Msg
	}
	return loc + ": " + d.Severity + "[" + d.Code + "]: " + d.Msg
}

// Apply resolves tags in place. strict reports V041 for implicit free-form acting.
// Explicit [act "..."] stays valid in both modes.
//
// No provider namespace is loaded in M0, so a head like ns:name is free-form
// acting. Bare words on a note:allow tag become the note; other value tokens
// stay arguments. Per-tag parameter schemas are not applied yet.
func Apply(script *ast.Script, strict bool) []Diag {
	c := &checker{file: script.File, strict: strict}
	for i := range script.Blocks {
		block := &script.Blocks[i]
		switch block.Kind {
		case ast.Comment, ast.Fence:
			continue
		case ast.Scene:
			c.walk(block.Nodes, true, true)
		case ast.Turn:
			c.walk(block.Nodes, true, false)
		default:
			// Narration is not a turn, so overlap and gap are out of place.
			c.walk(block.Nodes, false, false)
		}
	}
	return c.diags
}

// Failed returns an error when diags contain an error severity.
func Failed(diags []Diag) error {
	for _, d := range diags {
		if d.Severity == "error" {
			return errors.New("check failed")
		}
	}
	return nil
}

type checker struct {
	file   string
	strict bool
	diags  []Diag
}

func (c *checker) walk(nodes []ast.Node, atStart, scene bool) {
	for i := range nodes {
		n := &nodes[i]
		if n.Kind != ast.Tag {
			if n.Kind == ast.Text && strings.TrimSpace(n.Text) != "" {
				atStart = false
			}
			continue
		}
		c.one(n, atStart, scene)
		if atStart && turnTag(n) {
			continue
		}
		atStart = false
	}
}

func turnTag(n *ast.Node) bool {
	spec, ok := vocab.Lookup(n.Name)
	return ok && !n.Free && spec.Kind == vocab.Turn && !n.Span
}

func (c *checker) one(n *ast.Node, atStart, scene bool) {
	closed := c.resolve(n)
	// Accepted acting only. A strict-mode V041 already explains the tag.
	if !closed && n.Name == "act" && (!n.Free || !c.strict) && hasNonASCII(actingText(n)) {
		c.diag(n, "info", "I050", "non-ASCII free-form acting")
	}
	if n.Span {
		c.walk(n.Content, false, false)
	}
	if closed || n.Free {
		return
	}
	spec, ok := vocab.Lookup(n.Name)
	if !ok {
		return
	}
	if spec.NeedContent && !n.Span {
		c.error(n, "", `tag "`+spec.Name+`" needs content`)
	}
	if n.Span && !spec.Content {
		c.error(n, "", `tag "`+spec.Name+`" does not take content`)
	}
	if spec.Kind == vocab.Turn && !scene && !atStart {
		c.error(n, "", `tag "`+spec.Name+`" must be at the start of a turn`)
	}
}

// resolve reports V013 for a closing tag and returns true so later rules
// do not also treat [/pinyin] as a pinyin tag that needs content.
func (c *checker) resolve(n *ast.Node) bool {
	raw := strings.TrimSpace(n.Raw)
	if name, ok := closing(raw); ok {
		c.error(n, "V013", `[/`+name+`] is not valid; spans use [`+name+`|...]`)
		n.Name = name
		return true
	}
	if !n.Head {
		c.free(n, raw)
		return false
	}
	spec, ok := vocab.Lookup(n.Name)
	if !ok {
		c.free(n, raw)
		return false
	}
	n.Name = spec.Name
	n.Free = false
	if spec.Note {
		n.Args, n.Note = splitNote(n.Args)
	}
	return false
}

func (c *checker) free(n *ast.Node, raw string) {
	n.Free = true
	n.Name = "act"
	n.Note = raw
	n.Args = nil
	if !c.strict {
		// One unknown word within distance 2. A phrase is free-form with no warning.
		if s := suggest(raw); s != "" {
			c.diag(n, "warning", "W040", "unknown ["+raw+"] used as free-form acting; did you mean ["+s+"]?")
		}
		return
	}
	msg := `unregistered tag "` + raw + `"; for free-form acting write [act "` + raw + `"]`
	if s := suggest(raw); s != "" {
		msg = `unregistered tag "` + raw + `"; did you mean [` + s + `]? for free-form acting write [act "` + raw + `"]`
	}
	c.error(n, "V041", msg)
}

func splitNote(args []ast.Arg) ([]ast.Arg, string) {
	var kept []ast.Arg
	var notes []string
	for _, arg := range args {
		if arg.Key == "" && arg.Kind == "word" {
			notes = append(notes, arg.Raw)
			continue
		}
		kept = append(kept, arg)
	}
	return kept, strings.Join(notes, " ")
}

func (c *checker) error(n *ast.Node, code, msg string) {
	c.diag(n, "error", code, msg)
}

func (c *checker) diag(n *ast.Node, severity, code, msg string) {
	c.diags = append(c.diags, Diag{
		File:     c.file,
		Line:     n.Line,
		Col:      n.Col,
		Severity: severity,
		Code:     code,
		Msg:      msg,
	})
}

func actingText(n *ast.Node) string {
	if n.Free {
		return n.Note
	}
	text := n.Note
	for _, arg := range n.Args {
		if arg.Kind != "quoted" {
			continue
		}
		if text != "" {
			text += " "
		}
		text += arg.Raw
	}
	return text
}

func hasNonASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

func closing(raw string) (string, bool) {
	if !strings.HasPrefix(raw, "/") {
		return "", false
	}
	name := strings.TrimSpace(strings.TrimPrefix(raw, "/"))
	if name == "" || strings.ContainsAny(name, " \t") {
		return "", false
	}
	for i := range len(name) {
		c := name[i]
		if i == 0 && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			return "", false
		}
		if i > 0 && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' && c != '-' && c != '.' {
			return "", false
		}
	}
	return name, true
}

// suggest returns a registered name within Levenshtein distance 2.
// Phrases are not matched; only a single unknown word is.
// [laff] is distance 3 from [laugh], so it is reported with no suggestion.
func suggest(raw string) string {
	if strings.ContainsAny(raw, " \t") || strings.Contains(raw, ":") {
		return ""
	}
	name := vocab.Canon(raw)
	best := ""
	bestD := 3
	for _, cand := range vocab.Names() {
		d := dist(name, cand)
		if d < bestD || (d == bestD && (best == "" || cand < best)) {
			best = cand
			bestD = d
		}
	}
	if bestD <= 2 {
		return best
	}
	return ""
}

func dist(a, b string) int {
	if len(a) < len(b) {
		a, b = b, a
	}
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
