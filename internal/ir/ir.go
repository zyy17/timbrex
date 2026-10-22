// Package ir is the flat utterance list the synthesizer reads.
//
// The core only stores structure: who speaks, text, events, states, and spans.
// Provider-specific meaning of a tag name lives in the provider, not here.
package ir

// Kind is the role of one node in an utterance.
type Kind int

const (
	Text Kind = iota
	Event
	State
	SpanStart
	SpanEnd
	Raw
)

// Node is one flattened piece of an utterance.
// Scope is here, turn, scene, or span.
type Node struct {
	Kind  Kind
	Name  string
	Args  []Arg
	Note  string
	Text  string
	Scope string
}

type Arg struct {
	Key  string
	Raw  string
	Kind string
}

// Utterance is one speaker turn, scene line, or raw fence after lowering.
type Utterance struct {
	Speaker string
	Nodes   []Node
}

// Document is the whole script after lowering.
type Document struct {
	Provider   string
	Model      string
	Voice      string
	Lang       string
	Voices     map[string]map[string]string
	Utterances []Utterance
}
