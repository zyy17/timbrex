package check

import (
	"strings"
	"unicode/utf8"

	"github.com/zyy17/timbrex/internal/ast"
)

// A trie of colon-terminated segments handles colons within display names
// without scanning the cast or allocating prefixes for each paragraph.
type displayNameIndex struct {
	next      map[string]*displayNameIndex
	id        string
	name      string
	runes     int
	maxPrefix int
}

func newDisplayNameIndex(cast map[string]ast.Cast) *displayNameIndex {
	root := &displayNameIndex{}
	for id, member := range cast {
		name := strings.TrimSpace(member.Name)
		if name == "" {
			continue
		}
		for _, colon := range []string{":", "："} {
			rest := name + colon
			root.maxPrefix = max(root.maxPrefix, len(rest))
			node := root
			for rest != "" {
				var prefix string
				prefix, rest = splitDisplayPrefix(rest)
				if node.next == nil {
					node.next = make(map[string]*displayNameIndex)
				}
				if node.next[prefix] == nil {
					node.next[prefix] = &displayNameIndex{}
				}
				node = node.next[prefix]
			}
			if node.name == "" || id < node.id {
				node.id, node.name, node.runes = id, name, utf8.RuneCountInString(name)
			}
		}
	}
	return root
}

func (names *displayNameIndex) match(text string) (id, name, prefix string) {
	var best *displayNameIndex
	consumed, prefixEnd := 0, 0
	for rest, node := text[:min(len(text), names.maxPrefix)], names; rest != "" && node.next != nil; {
		part, tail := splitDisplayPrefix(rest)
		node = node.next[part]
		if node == nil {
			break
		}
		consumed += len(part)
		if node.name != "" && (best == nil || node.runes > best.runes || (node.runes == best.runes && node.id < best.id)) {
			best, prefixEnd = node, consumed
		}
		rest = tail
	}
	if best == nil {
		return "", "", ""
	}
	return best.id, best.name, text[:prefixEnd]
}

func splitDisplayPrefix(text string) (prefix, rest string) {
	i := strings.IndexAny(text, ":：")
	if i < 0 {
		return "", ""
	}
	end := i + 1
	if text[i] != ':' {
		end = i + len("：")
	}
	return text[:end], text[end:]
}
