// Package parse turns a Timbrex script into an AST.
//
// A script is plain text, with optional YAML between opening and closing
// "---" lines, line comments, speaker turns, and [tag] markup.
package parse

import (
	"errors"
	"strings"

	"github.com/zyy17/timbrex/internal/ast"
	"github.com/zyy17/timbrex/internal/lex"
)

// Parse reads src as a Timbrex script. file is used only in error text.
func Parse(file, src string) (*ast.Script, error) {
	src = normalize(src)
	front, body, err := splitFront(file, src)
	if err != nil {
		return nil, err
	}
	off := lineOffset(src, body)
	toks, err := lex.Scan(body)
	if err != nil {
		return nil, shiftLexErr(file, err, off)
	}
	shiftLines(toks, off)
	p := &parser{file: file, front: front, toks: toks}
	blocks, err := p.blocks()
	if err != nil {
		return nil, err
	}
	return &ast.Script{File: file, Front: front, Blocks: blocks}, nil
}

func shiftLexErr(file string, err error, off int) error {
	var le *lex.Error
	if errors.As(err, &le) {
		return &Error{File: file, Line: le.Line + off, Col: le.Col, Msg: le.Msg}
	}
	return err
}

func normalize(src string) string {
	src = strings.TrimPrefix(src, "\uFEFF")
	src = strings.ReplaceAll(src, "\r\n", "\n")
	return strings.ReplaceAll(src, "\r", "\n")
}

func lineOffset(src, body string) int {
	if !strings.HasSuffix(src, body) {
		return 0
	}
	head := src[:len(src)-len(body)]
	return strings.Count(head, "\n")
}

func shiftLines(toks []lex.Token, offset int) {
	if offset == 0 {
		return
	}
	for i := range toks {
		if toks[i].Kind == lex.EOF {
			continue
		}
		toks[i].Line += offset
	}
}

type parser struct {
	file  string
	front ast.Front
	toks  []lex.Token
	i     int
}

func (p *parser) blocks() ([]ast.Block, error) {
	var blocks []ast.Block
	for p.peek().Kind != lex.EOF {
		p.skipBlanks()
		if p.peek().Kind == lex.EOF {
			break
		}
		if p.peek().Kind == lex.Comment {
			tok := p.next()
			blocks = append(blocks, ast.Block{
				Kind:    ast.Comment,
				Comment: strings.TrimSpace(tok.Lit),
				Line:    tok.Line,
				Col:     tok.Col,
			})
			p.skipOneNL()
			continue
		}
		if p.peek().Kind == lex.Fence {
			tok := p.next()
			if !fenceInfoOK(tok.Lit) {
				return nil, &Error{
					File: p.file,
					Line: tok.Line,
					Col:  tok.Col,
					Msg:  "fence info must start with an identifier",
				}
			}
			blocks = append(blocks, ast.Block{
				Kind: ast.Fence,
				Info: tok.Lit,
				Raw:  tok.Body,
				Line: tok.Line,
				Col:  tok.Col,
			})
			continue
		}
		if p.isScene(p.lookLine()) {
			line := p.readLine()
			block, err := buildScene(p.file, line)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, *block)
			continue
		}
		block, err := p.paragraph()
		if err != nil {
			return nil, err
		}
		if block != nil {
			blocks = append(blocks, *block)
		}
	}
	return blocks, nil
}

func (p *parser) paragraph() (*ast.Block, error) {
	var lines [][]lex.Token
	for {
		kind := p.peek().Kind
		if kind == lex.EOF || kind == lex.Comment || kind == lex.Fence {
			break
		}
		if kind == lex.Newline {
			p.next()
			if len(lines) > 0 {
				break
			}
			continue
		}
		look := p.lookLine()
		if len(lines) > 0 && (p.isScene(look) || p.lineStartsTurn(look)) {
			break
		}
		line := p.readLine()
		if blankLine(line) {
			if len(lines) > 0 {
				break
			}
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return nil, nil
	}
	return buildParagraph(p.file, p.front, lines)
}

func (p *parser) lookLine() []lex.Token {
	var line []lex.Token
	for i := p.i; i < len(p.toks); i++ {
		tok := p.toks[i]
		if tok.Kind == lex.Newline || tok.Kind == lex.EOF || tok.Kind == lex.Fence {
			break
		}
		line = append(line, tok)
	}
	return line
}

func (p *parser) lineStartsTurn(line []lex.Token) bool {
	if len(line) > 0 && line[0].Depth > 0 {
		return false
	}
	speaker, _, _, _ := speakerPrefix(line)
	return speakerAllowed(p.front, speaker)
}

func (p *parser) isScene(line []lex.Token) bool {
	if len(line) > 0 && line[0].Depth > 0 {
		return false
	}
	if blankLine(line) || p.lineStartsTurn(line) {
		return false
	}
	nodes, err := parseStream(p.file, line, false, lex.Token{})
	if err != nil || len(nodes) == 0 {
		return false
	}
	sawTag := false
	for _, node := range nodes {
		switch node.Kind {
		case ast.Tag:
			if node.Span {
				return false
			}
			sawTag = true
		case ast.Text:
			if strings.TrimSpace(node.Text) != "" {
				return false
			}
		default:
			return false
		}
	}
	return sawTag
}

func buildScene(file string, line []lex.Token) (*ast.Block, error) {
	nodes, err := parseStream(file, line, false, lex.Token{})
	if err != nil {
		return nil, err
	}
	var kept []ast.Node
	for _, node := range nodes {
		if node.Kind == ast.Text && strings.TrimSpace(node.Text) == "" {
			continue
		}
		kept = append(kept, node)
	}
	lineNo, col := 1, 1
	if len(line) > 0 {
		lineNo, col = line[0].Line, line[0].Col
	}
	return &ast.Block{Kind: ast.Scene, Nodes: kept, Line: lineNo, Col: col}, nil
}

func (p *parser) readLine() []lex.Token {
	var line []lex.Token
	for {
		tok := p.peek()
		if tok.Kind == lex.EOF || tok.Kind == lex.Newline {
			if tok.Kind == lex.Newline {
				p.next()
			}
			return line
		}
		line = append(line, p.next())
	}
}

func (p *parser) skipBlanks() {
	for {
		tok := p.peek()
		if tok.Kind == lex.Newline {
			p.next()
			continue
		}
		if tok.Kind == lex.Text && strings.TrimSpace(tok.Lit) == "" && p.peekAhead(1).Kind == lex.Newline {
			p.next()
			p.next()
			continue
		}
		return
	}
}

func (p *parser) skipOneNL() {
	if p.peek().Kind == lex.Newline {
		p.next()
	}
}

func (p *parser) peek() lex.Token {
	if p.i >= len(p.toks) {
		return lex.Token{Kind: lex.EOF}
	}
	return p.toks[p.i]
}

func (p *parser) peekAhead(n int) lex.Token {
	i := p.i + n
	if i >= len(p.toks) {
		return lex.Token{Kind: lex.EOF}
	}
	return p.toks[i]
}

func (p *parser) next() lex.Token {
	tok := p.peek()
	if p.i < len(p.toks) {
		p.i++
	}
	return tok
}

// fenceInfoOK matches fence_info = ident , { WS , ( ident | kv ) } at the start.
// The first word must be an ASCII identifier. Later words are not checked here.
func fenceInfoOK(info string) bool {
	info = strings.TrimSpace(info)
	if info == "" {
		return false
	}
	field := info
	if i := strings.IndexAny(info, " \t"); i >= 0 {
		field = info[:i]
	}
	return field != "" && identLen(field) == len(field)
}

func blankLine(line []lex.Token) bool {
	if len(line) == 0 {
		return true
	}
	for _, tok := range line {
		if tok.Kind != lex.Text || strings.TrimSpace(tok.Lit) != "" {
			return false
		}
	}
	return true
}
