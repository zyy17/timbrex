// Package ast is the script tree produced by the parser.
package ast

// BlockKind distinguishes narration, a speaker turn, a comment, a scene line, and a fence.
type BlockKind int

const (
	Narration BlockKind = iota
	Turn
	Comment
	Scene
	Fence
)

// Block is one paragraph, comment, scene line, or raw fence.
type Block struct {
	Kind    BlockKind
	Speaker string
	Comment string
	Info    string // fence info, such as "ssml azure"
	Raw     string // fence body, unparsed
	Nodes   []Node
	Line    int
	Col     int
}

// NodeKind distinguishes literal text from a tag.
type NodeKind int

const (
	Text NodeKind = iota
	Tag
)

// Arg is one tag argument. Key is empty for a positional value.
// Kind is number, duration, percent, semitone, decibel, quoted, or word.
type Arg struct {
	Key  string
	Raw  string
	Kind string
}

// Node is a piece of a paragraph.
// A span tag has Span set. Head is true when the body starts with an ident or ns:ident.
// Raw is the tag body before a pipe. Check fills Free, Note, and the canonical Name.
type Node struct {
	Kind    NodeKind
	Text    string
	Head    bool
	Raw     string
	Name    string
	Args    []Arg
	Note    string
	Free    bool
	Span    bool
	Content []Node
	Line    int
	Col     int
}

// Cast is one declared speaker.
type Cast struct {
	Name string
}

// Front is the YAML front matter.
// Cast is nil when the script did not declare one.
// CastAuto is the cast: auto form.
// StrictTags is nil when the key is absent.
// Narrator replaces the default narration speaker when set.
type Front struct {
	Timbrex    int
	Provider   string
	Lang       string
	StrictTags *bool
	Narrator   string
	CastAuto   bool
	Cast       map[string]Cast
}

// Script is a parsed .tbx file.
type Script struct {
	File   string
	Front  Front
	Blocks []Block
}
