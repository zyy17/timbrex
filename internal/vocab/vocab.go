// Package vocab is the standard tag list loaded with every script.
package vocab

import "strings"

// Kind is how a registered tag behaves.
type Kind int

const (
	Event Kind = iota
	State
	Wrap
	Turn
)

// Spec is one registered tag.
type Spec struct {
	Name        string
	Kind        Kind
	Note        bool // bare words are a note
	Content     bool // [name|content] is allowed
	NeedContent bool // [name|content] is required
}

var specs map[string]Spec

func init() {
	specs = map[string]Spec{}
	add(Event, false, false, false,
		"pause", "silence", "duration", "break", "rate",
		"sfx", "audio", "bgm", "ambience", "vc", "cut", "mark")
	add(State, false, false, false, "reset")
	add(Event, true, true, false,
		"laugh", "sigh", "breath", "cough", "gasp",
		"inhale", "exhale", "quick-breath", "pant", "huff", "groan", "grunt", "moan",
		"chuckle", "giggle", "cackle", "snort", "scoff",
		"cry", "sob", "sniffle", "sniff", "whimper", "wail",
		"clear-throat", "sneeze", "hiccup", "yawn", "burp", "snore",
		"swallow", "gulp", "lip-smack", "tongue-click", "kiss", "shush",
		"whistle", "hum", "hiss", "hmm", "uh", "um", "emm")
	add(State, false, true, false,
		"neutral", "happy", "sad", "angry", "fear", "surprise", "disgust", "calm",
		"excited", "tender", "serious", "curious", "sarcastic", "nervous", "tired",
		"bored", "contempt", "confused", "proud", "embarrassed", "hopeful",
		"disappointed", "panicked", "crying",
		"emotion", "speed", "fast", "slow", "pitch", "volume",
		"whisper", "shout", "emph",
		"style", "role", "accent", "oral", "dialect", "emovec")
	// act takes one quoted string. Bare words are the note.
	add(State, true, true, false, "act")
	add(Wrap, false, true, true,
		"pinyin", "ipa", "sub", "say", "arpabet", "jyutping", "romaji",
		"spell", "verbatim", "latex", "lang", "as", "voice")
	add(Turn, false, false, false, "overlap", "gap")
}

func add(kind Kind, note, content, need bool, names ...string) {
	for _, name := range names {
		specs[name] = Spec{
			Name:        name,
			Kind:        kind,
			Note:        note,
			Content:     content,
			NeedContent: need,
		}
	}
}

// Lookup returns the canonical spec for a tag name.
// Matching is case-insensitive, and "-" and "_" are the same character.
func Lookup(name string) (Spec, bool) {
	spec, ok := specs[Canon(name)]
	return spec, ok
}

// Names returns every canonical tag name.
func Names() []string {
	out := make([]string, 0, len(specs))
	for name := range specs {
		out = append(out, name)
	}
	return out
}

// Canon folds a tag name for lookup.
func Canon(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
}
