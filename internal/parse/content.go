package parse

import (
	"strings"

	"github.com/zyy17/timbrex/internal/ast"
	"github.com/zyy17/timbrex/internal/lex"
)

func buildParagraph(file string, front ast.Front, lines [][]lex.Token) (*ast.Block, error) {
	speaker, first, posLine, posCol := speakerPrefix(lines[0])
	// A speaker prefix counts only when cast lists that id, or cast is auto.
	// Any other "ident:" line is narration, including when cast is omitted.
	if speakerAllowed(front, speaker) {
		lines[0] = first
	} else {
		speaker = ""
	}
	stream := joinLines(lines)
	nodes, err := parseStream(file, stream, false, lex.Token{})
	if err != nil {
		return nil, err
	}
	block := &ast.Block{Nodes: nodes, Line: posLine, Col: posCol}
	if speaker != "" {
		block.Kind = ast.Turn
		block.Speaker = speaker
		return block, nil
	}
	block.Kind = ast.Narration
	return block, nil
}

func joinLines(lines [][]lex.Token) []lex.Token {
	var stream []lex.Token
	for i, line := range lines {
		if i > 0 && len(line) > 0 {
			stream = append(stream, lex.Token{Kind: lex.Text, Lit: "\n", Line: line[0].Line, Col: line[0].Col})
		}
		stream = append(stream, line...)
	}
	return stream
}

func speakerPrefix(line []lex.Token) (string, []lex.Token, int, int) {
	lineNo, col := 1, 1
	if len(line) > 0 {
		lineNo, col = line[0].Line, line[0].Col
	}
	if len(line) == 0 || line[0].Kind != lex.Text {
		return "", line, lineNo, col
	}
	n := identLen(line[0].Lit)
	lit := line[0].Lit
	if n == 0 || n >= len(lit) || lit[n] != ':' {
		return "", line, lineNo, col
	}
	out := append([]lex.Token(nil), line...)
	out[0].Lit = strings.TrimLeft(lit[n+1:], " \t")
	if out[0].Lit == "" {
		out = out[1:]
	}
	return lit[:n], out, lineNo, col
}

func speakerAllowed(front ast.Front, id string) bool {
	if id == "" {
		return false
	}
	if front.CastAuto {
		return true
	}
	if front.Cast == nil {
		return false
	}
	_, ok := front.Cast[id]
	return ok
}

func identLen(s string) int {
	for i := range len(s) {
		c := s[i]
		if i == 0 {
			if !isLetter(c) {
				return 0
			}
			continue
		}
		if !isLetter(c) && !isDigit(c) && c != '_' && c != '-' && c != '.' {
			return i
		}
	}
	return len(s)
}

func isLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

type cursor struct {
	file string
	toks []lex.Token
	i    int
}

func parseStream(file string, toks []lex.Token, span bool, open lex.Token) ([]ast.Node, error) {
	c := &cursor{file: file, toks: toks}
	return c.content(span, open)
}

func (c *cursor) content(span bool, open lex.Token) ([]ast.Node, error) {
	var nodes []ast.Node
	var b strings.Builder
	line, col := 1, 1
	flush := func() {
		if b.Len() == 0 {
			return
		}
		nodes = append(nodes, ast.Node{Kind: ast.Text, Text: b.String(), Line: line, Col: col})
		b.Reset()
	}
	for c.i < len(c.toks) {
		tok := c.toks[c.i]
		switch tok.Kind {
		case lex.RBrack:
			if span {
				flush()
				return nodes, nil
			}
			c.i++
			c.write(&b, &line, &col, "]", tok)
		case lex.LBrack:
			flush()
			tag, err := c.tag()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, *tag)
		case lex.Pipe:
			c.i++
			c.write(&b, &line, &col, "|", tok)
		case lex.Text:
			c.i++
			c.write(&b, &line, &col, tok.Lit, tok)
		default:
			c.i++
		}
	}
	if span {
		return nil, unclosed(c.file, open)
	}
	flush()
	return nodes, nil
}

func (c *cursor) write(b *strings.Builder, line, col *int, lit string, tok lex.Token) {
	if b.Len() == 0 {
		*line, *col = tok.Line, tok.Col
	}
	b.WriteString(lit)
}

func (c *cursor) tag() (*ast.Node, error) {
	open := c.toks[c.i]
	c.i++
	var body strings.Builder
	for c.i < len(c.toks) {
		tok := c.toks[c.i]
		switch tok.Kind {
		case lex.Pipe:
			c.i++
			content, err := c.content(true, open)
			if err != nil {
				return nil, err
			}
			if c.i >= len(c.toks) || c.toks[c.i].Kind != lex.RBrack {
				return nil, unclosed(c.file, open)
			}
			c.i++
			return c.doneTag(open, body.String(), content, true)
		case lex.RBrack:
			c.i++
			return c.doneTag(open, body.String(), nil, false)
		case lex.LBrack:
			return nil, &Error{File: c.file, Line: tok.Line, Col: tok.Col, Msg: "unescaped [ inside tag"}
		case lex.Text:
			body.WriteString(tok.Lit)
			c.i++
		default:
			return nil, unclosed(c.file, open)
		}
	}
	return nil, unclosed(c.file, open)
}

func unclosed(file string, open lex.Token) error {
	return &Error{File: file, Line: open.Line, Col: open.Col, Code: "V012", Msg: "unclosed tag"}
}

func (c *cursor) doneTag(open lex.Token, body string, content []ast.Node, span bool) (*ast.Node, error) {
	raw := strings.TrimSpace(body)
	if raw == "" {
		return nil, &Error{File: c.file, Line: open.Line, Col: open.Col, Msg: "empty tag"}
	}
	node := &ast.Node{
		Kind:    ast.Tag,
		Raw:     raw,
		Span:    span,
		Content: content,
		Line:    open.Line,
		Col:     open.Col,
	}
	head, rest, ok := splitHead(raw)
	if !ok {
		node.Name = raw
		return node, nil
	}
	args, err := scanArgs(rest)
	if err != nil {
		return nil, &Error{File: c.file, Line: open.Line, Col: open.Col, Msg: err.Error()}
	}
	node.Head = true
	node.Name = head
	node.Args = args
	return node, nil
}

func splitHead(body string) (string, string, bool) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", "", false
	}
	head := body
	rest := ""
	for i := range len(body) {
		if body[i] == ' ' || body[i] == '\t' {
			head = body[:i]
			rest = strings.TrimSpace(body[i:])
			break
		}
	}
	if !isHead(head) {
		return "", "", false
	}
	return head, rest, true
}

func isHead(s string) bool {
	left, right, ok := strings.Cut(s, ":")
	if !ok {
		return identLen(s) == len(s) && s != ""
	}
	return identLen(left) == len(left) && left != "" && identLen(right) == len(right) && right != ""
}
