package parse

import "fmt"

// Error is a parse failure with a 1-based source location.
type Error struct {
	File string
	Line int
	Col  int
	Code string
	Msg  string
}

func (e *Error) Error() string {
	msg := e.Msg
	if e.Code != "" {
		msg = "error[" + e.Code + "]: " + e.Msg
	}
	if e.File == "" {
		return fmt.Sprintf("%d:%d: %s", e.Line, e.Col, msg)
	}
	return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Col, msg)
}
