package parse

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/zyy17/timbrex/internal/ast"
)

// scanArgs reads the value grammar from the text after a tag head.
// A bare word, a quoted string, a number with an optional suffix, or key=value.
func scanArgs(s string) ([]ast.Arg, error) {
	var args []ast.Arg
	i := 0
	for {
		i = skipSpace(s, i)
		if i >= len(s) {
			return args, nil
		}
		arg, next, err := scanArg(s, i)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		i = next
	}
}

func scanArg(s string, i int) (ast.Arg, int, error) {
	if s[i] == '"' {
		text, next, err := scanQuoted(s, i)
		if err != nil {
			return ast.Arg{}, 0, err
		}
		return ast.Arg{Raw: text, Kind: "quoted"}, next, nil
	}
	if n := identLen(s[i:]); n > 0 && i+n < len(s) && s[i+n] == '=' {
		key := s[i : i+n]
		val, next, err := scanValue(s, i+n+1)
		if err != nil {
			return ast.Arg{}, 0, err
		}
		val.Key = key
		return val, next, nil
	}
	return scanValue(s, i)
}

func scanValue(s string, i int) (ast.Arg, int, error) {
	if i >= len(s) {
		return ast.Arg{}, 0, fmt.Errorf("malformed tag argument")
	}
	if s[i] == '"' {
		text, next, err := scanQuoted(s, i)
		if err != nil {
			return ast.Arg{}, 0, err
		}
		return ast.Arg{Raw: text, Kind: "quoted"}, next, nil
	}
	if raw, kind, next, ok := scanNumber(s, i); ok {
		return ast.Arg{Raw: raw, Kind: kind}, next, nil
	}
	raw, next := scanWord(s, i)
	if raw == "" {
		return ast.Arg{}, 0, fmt.Errorf("malformed tag argument")
	}
	return ast.Arg{Raw: raw, Kind: "word"}, next, nil
}

func scanQuoted(s string, i int) (string, int, error) {
	i++
	var b strings.Builder
	for i < len(s) {
		if s[i] == '"' {
			return b.String(), i + 1, nil
		}
		if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
			b.WriteByte(s[i+1])
			i += 2
			continue
		}
		r, w := utf8.DecodeRuneInString(s[i:])
		b.WriteRune(r)
		i += w
	}
	return "", 0, fmt.Errorf("unclosed quote")
}

func scanNumber(s string, i int) (string, string, int, bool) {
	start := i
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	if i >= len(s) || !isDigit(s[i]) {
		return "", "", 0, false
	}
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	if i < len(s) && s[i] == '.' {
		if i+1 < len(s) && isDigit(s[i+1]) {
			i++
			for i < len(s) && isDigit(s[i]) {
				i++
			}
		}
	}
	kind := "number"
	// Longer suffixes win. "st" and "ms" must be tested before "s".
	// A suffix counts only at a token boundary, so "10stone" stays one word.
	switch {
	case hasSuffix(s, i, "ms"):
		i += 2
		kind = "duration"
	case hasSuffix(s, i, "min"):
		i += 3
		kind = "duration"
	case hasSuffix(s, i, "st"):
		i += 2
		kind = "semitone"
	case hasSuffix(s, i, "dB"):
		i += 2
		kind = "decibel"
	case hasSuffix(s, i, "%"):
		i++
		kind = "percent"
	case hasSuffix(s, i, "s"):
		i++
		kind = "duration"
	default:
		if i < len(s) && isValueCont(s[i]) {
			return "", "", 0, false
		}
	}
	return s[start:i], kind, i, true
}

func hasSuffix(s string, i int, suf string) bool {
	end := i + len(suf)
	if end > len(s) || s[i:end] != suf {
		return false
	}
	return end == len(s) || !isValueCont(s[end])
}

func isValueCont(c byte) bool {
	return isLetter(c) || isDigit(c) || c == '_' || c == '-' || c == '.'
}

func scanWord(s string, i int) (string, int) {
	start := i
	for i < len(s) {
		switch s[i] {
		case ' ', '\t', '[', ']', '|', '=', '"':
			return s[start:i], i
		}
		_, w := utf8.DecodeRuneInString(s[i:])
		i += w
	}
	return s[start:i], i
}

func skipSpace(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}
