package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/check"
	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/lower"
	"github.com/zyy17/timbrex/internal/parse"
	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/vocab"
)

func TestSpecVocabularyMatchesRegistry(t *testing.T) {
	want := vocab.Names()
	slices.Sort(want)
	var emotions []string
	for _, doc := range []struct{ path, header, emotionFamily string }{
		{"docs/spec.md", "| Family | Names |", "Emotions (state)"},
		{"docs/spec.zh-cn.md", "| 类别 | 名称 |", "情绪（state）"},
	} {
		families := specVocabulary(t, doc.path, doc.header)
		var names []string
		for _, family := range families {
			names = append(names, family...)
		}
		slices.Sort(names)
		if !slices.Equal(names, want) {
			t.Errorf("%s: vocabulary = %v, want %v", doc.path, names, want)
		}
		got := families[doc.emotionFamily]
		if len(got) == 0 {
			t.Fatalf("%s: missing emotion vocabulary", doc.path)
		}
		slices.Sort(got)
		if emotions == nil {
			emotions = got
		} else if !slices.Equal(got, emotions) {
			t.Errorf("%s: emotions = %v, want %v", doc.path, got, emotions)
		}
	}
	for _, name := range emotions {
		t.Run(name, func(t *testing.T) {
			spec, ok := vocab.Lookup(name)
			if !ok || spec.Kind != vocab.State || !spec.Content {
				t.Fatalf("emotion %q: spec = %+v, registered = %v", name, spec, ok)
			}
			if direction, err := providers.Direction(ir.Node{Name: name}); err != nil || direction != name {
				t.Fatalf("direction = %q, err = %v", direction, err)
			}
			src := fmt.Sprintf("[emotion scene]\nA [%s] B [emotion inner|C] D [reset %s] E [%s] F [reset emotion] G\n", name, name, name)
			script, err := parse.Parse("emotion.tbx", src)
			if err != nil {
				t.Fatal(err)
			}
			if err := check.Failed(check.Apply(script, true)); err != nil {
				t.Fatal(err)
			}
			segments, err := providers.Plan(lower.Script(script), "openai", "coral")
			if err != nil {
				t.Fatal(err)
			}
			want := []string{"A:emotion scene", "B:" + name, "C:emotion inner", "D:" + name, "E:emotion scene", "F:" + name, "G:emotion scene"}
			if len(segments) != len(want) {
				t.Fatalf("segments = %+v, want %d", segments, len(want))
			}
			for i, seg := range segments {
				text, err := providers.Text(seg.Nodes, nil)
				if err != nil || len(seg.Controls) != 1 {
					t.Fatalf("segment %d: controls = %+v, err = %v", i, seg.Controls, err)
				}
				control := seg.Controls[0].Name
				for _, arg := range seg.Controls[0].Args {
					control += " " + arg.Raw
				}
				if got := strings.TrimSpace(text) + ":" + control; got != want[i] {
					t.Errorf("segment %d = %q, want %q", i, got, want[i])
				}
			}
		})
	}
}

func specVocabulary(t *testing.T, path, header string) map[string][]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, table, ok := strings.Cut(string(data), header+"\n")
	if !ok {
		t.Fatalf("%s: missing vocabulary table", path)
	}
	families := make(map[string][]string)
	for _, line := range strings.Split(table, "\n") {
		if !strings.HasPrefix(line, "|") {
			break
		}
		code := inlineCode.FindAllStringSubmatch(line, -1)
		if len(code) == 0 {
			continue // Table separator.
		}
		if len(code) != 1 {
			t.Fatalf("%s: expected one tag list in %q", path, line)
		}
		family := strings.TrimSpace(strings.Split(line, "|")[1])
		if _, exists := families[family]; exists {
			t.Fatalf("%s: duplicate vocabulary family %q", path, family)
		}
		families[family] = strings.Fields(code[0][1])
	}
	return families
}
