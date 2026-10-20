// Package lex tokenizes a Timbrex script body.
//
// Markup is [name] or [name|content]. "|" is special only inside a tag.
// A line whose first non-space characters are "//" is a comment.
// A line that starts a ``` fence is raw text until the closing fence.
package lex

import (
	"strings"
	"unicode/utf8"
)

// Kind identifies a token.
type Kind int

const (
	EOF Kind = iota
	Newline
	Text
	LBrack
	RBrack
	Pipe
	Comment
	Fence
)

// Token is one lexical unit. Line and Col are 1-based.
// Body is set on a Fence token and holds the raw interior.
type Token struct {
	Kind Kind
	Lit  string
	Body string
	Line int
	Col  int
}

// Error is a lexical failure. The parser adds the file name.
type Error struct {
	Line int
	Col  int
	Msg  string
}

func (e *Error) Error() string {
	return e.Msg
}

// Scan tokenizes src. Newlines are already normalized by the parser.
func Scan(src string) ([]Token, error) {
	s := &scanner{src: src, line: 1, col: 1, bol: true}
	var out []Token
	for {
		tok, err := s.next()
		if err != nil {
			return nil, err
		}
		out = append(out, tok)
		if tok.Kind == EOF {
			return out, nil
		}
	}
}

type scanner struct {
	src   string
	i     int
	line  int
	col   int
	bol   bool
	depth int
}

func (s *scanner) next() (Token, error) {
	if s.i >= len(s.src) {
		return Token{Kind: EOF, Line: s.line, Col: s.col}, nil
	}
	if tok, ok := s.tryComment(); ok {
		return tok, nil
	}
	if s.atFence() {
		return s.fence()
	}
	// A quoted argument is opaque. "|" "[" "]" inside it are not markup.
	if s.depth > 0 && s.peek() == '"' {
		return s.quoted()
	}
	switch s.peek() {
	case '\n':
		return s.emit(Newline, "\n", 1), nil
	case '[':
		if s.peekAt(1) == '[' {
			return s.text(), nil
		}
		s.depth++
		return s.emit(LBrack, "[", 1), nil
	case ']':
		if s.peekAt(1) == ']' {
			return s.text(), nil
		}
		if s.depth > 0 {
			s.depth--
		}
		return s.emit(RBrack, "]", 1), nil
	case '|':
		if s.depth > 0 {
			return s.emit(Pipe, "|", 1), nil
		}
		return s.text(), nil
	case '\\':
		n := s.peekAt(1)
		if escapable(n) {
			line, col := s.line, s.col
			s.skip(2)
			return Token{Kind: Text, Lit: string(n), Line: line, Col: col}, nil
		}
		// A lone backslash is literal. text() stops before "\", so this byte
		// must be consumed here or Scan spins on the same position.
		return s.emit(Text, `\`, 1), nil
	default:
		return s.text(), nil
	}
}

func (s *scanner) atFence() bool {
	if !s.bol || s.depth > 0 {
		return false
	}
	i := 0
	for s.peekAt(i) == ' ' || s.peekAt(i) == '\t' {
		i++
	}
	return s.peekAt(i) == '`' && s.peekAt(i+1) == '`' && s.peekAt(i+2) == '`'
}

func (s *scanner) fence() (Token, error) {
	line, col := s.line, s.col
	for s.peek() == ' ' || s.peek() == '\t' {
		s.advance()
	}
	s.skip(3)
	infoStart := s.i
	for s.i < len(s.src) && s.peek() != '\n' {
		s.advance()
	}
	info := strings.TrimSpace(s.src[infoStart:s.i])
	if info == "" {
		return Token{}, &Error{Line: line, Col: col, Msg: "fence needs an info string"}
	}
	if s.peek() == '\n' {
		s.advance()
	}
	bodyStart := s.i
	for s.i < len(s.src) {
		if s.closingFence() {
			body := s.src[bodyStart:s.i]
			s.eatFenceLine()
			return Token{Kind: Fence, Lit: info, Body: body, Line: line, Col: col}, nil
		}
		for s.i < len(s.src) && s.peek() != '\n' {
			s.advance()
		}
		if s.peek() == '\n' {
			s.advance()
		}
	}
	return Token{}, &Error{Line: line, Col: col, Msg: "unclosed fence"}
}

func (s *scanner) closingFence() bool {
	i := 0
	for s.peekAt(i) == ' ' || s.peekAt(i) == '\t' {
		i++
	}
	if s.peekAt(i) != '`' || s.peekAt(i+1) != '`' || s.peekAt(i+2) != '`' {
		return false
	}
	i += 3
	for {
		r := s.peekAt(i)
		if r == 0 || r == '\n' {
			return true
		}
		if r != ' ' && r != '\t' {
			return false
		}
		i++
	}
}

func (s *scanner) eatFenceLine() {
	for s.i < len(s.src) && s.peek() != '\n' {
		s.advance()
	}
	if s.peek() == '\n' {
		s.advance()
	}
}

func (s *scanner) tryComment() (Token, bool) {
	if !s.bol || s.depth > 0 {
		return Token{}, false
	}
	i := 0
	for {
		r := s.peekAt(i)
		if r != ' ' && r != '\t' {
			break
		}
		i++
	}
	if s.peekAt(i) != '/' || s.peekAt(i+1) != '/' {
		return Token{}, false
	}
	line, col := s.line, s.col
	s.skip(i + 2)
	start := s.i
	for s.i < len(s.src) && s.peek() != '\n' {
		s.advance()
	}
	return Token{Kind: Comment, Lit: s.src[start:s.i], Line: line, Col: col}, true
}

func (s *scanner) text() Token {
	line, col := s.line, s.col
	var b strings.Builder
	for s.i < len(s.src) {
		r := s.peek()
		if r == '\n' || (r == '|' && s.depth > 0) {
			break
		}
		if r == '[' {
			if s.peekAt(1) == '[' {
				b.WriteByte('[')
				s.skip(2)
				continue
			}
			break
		}
		if r == ']' {
			if s.peekAt(1) == ']' {
				b.WriteByte(']')
				s.skip(2)
				continue
			}
			break
		}
		if r == '\\' {
			break
		}
		if s.depth > 0 && r == '"' {
			break
		}
		b.WriteRune(r)
		s.advance()
	}
	return Token{Kind: Text, Lit: b.String(), Line: line, Col: col}
}

// quoted reads one double-quoted string inside a tag, including its quotes.
// \" and \\ stay in the literal so the argument parser can unescape them.
func (s *scanner) quoted() (Token, error) {
	line, col := s.line, s.col
	var b strings.Builder
	b.WriteByte('"')
	s.advance()
	for s.i < len(s.src) {
		r := s.peek()
		if r == '"' {
			b.WriteByte('"')
			s.advance()
			return Token{Kind: Text, Lit: b.String(), Line: line, Col: col}, nil
		}
		if r == '\\' {
			n := s.peekAt(1)
			if n == '"' || n == '\\' {
				b.WriteByte('\\')
				b.WriteRune(n)
				s.skip(2)
				continue
			}
		}
		b.WriteRune(r)
		s.advance()
	}
	return Token{}, &Error{Line: line, Col: col, Msg: "unclosed quote"}
}

func (s *scanner) emit(kind Kind, lit string, n int) Token {
	line, col := s.line, s.col
	s.skip(n)
	return Token{Kind: kind, Lit: lit, Line: line, Col: col}
}

func (s *scanner) skip(n int) {
	for range n {
		s.advance()
	}
}

func (s *scanner) advance() {
	r, w := utf8.DecodeRuneInString(s.src[s.i:])
	s.i += w
	if r == '\n' {
		s.line++
		s.col = 1
		s.bol = true
		return
	}
	s.col++
	s.bol = false
}

func (s *scanner) peek() rune {
	return s.peekAt(0)
}

func (s *scanner) peekAt(n int) rune {
	i := s.i
	for range n {
		if i >= len(s.src) {
			return 0
		}
		_, w := utf8.DecodeRuneInString(s.src[i:])
		i += w
	}
	if i >= len(s.src) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(s.src[i:])
	return r
}

func escapable(r rune) bool {
	switch r {
	case '[', ']', '|', '\\', ':':
		return true
	default:
		return false
	}
}
