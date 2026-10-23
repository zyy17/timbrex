package parse

import (
	"strings"
	"testing"
)

func TestFrontMatterFenceInsideScalar(t *testing.T) {
	for _, tc := range []struct{ style, want string }{{"|", "---\n"}, {">-", "---"}} {
		t.Run(tc.style, func(t *testing.T) {
			metadata := "cast:\n  lin:\n    name: " + tc.style + "\n      ---\n    voices: {openai: coral}\nstrict_tags: true\n"
			src := "--- \t\n" + metadata + "---\t \nlin: Hi\n"
			for _, source := range []string{src, strings.ReplaceAll(src, "\n", "\r\n")} {
				script, err := Parse("scalar.tbx", source)
				if err != nil {
					t.Fatal(err)
				}
				member := script.Front.Cast["lin"]
				if member.Name != tc.want || member.Voices["openai"] != "coral" {
					t.Fatalf("cast = %+v", member)
				}
				if script.Front.StrictTags == nil || !*script.Front.StrictTags {
					t.Fatal("strict_tags was lost")
				}
				if len(script.Blocks) != 1 || script.Blocks[0].Speaker != "lin" || script.Blocks[0].Line != 9 {
					t.Fatalf("body = %+v", script.Blocks)
				}
			}
		})
	}
}

func TestFrontMatterClosingFenceAtEOF(t *testing.T) {
	for _, src := range []string{"---\n---", "---\ncast: auto\n--- \t"} {
		script, err := Parse("eof.tbx", src)
		if err != nil || len(script.Blocks) != 0 {
			t.Fatalf("%q: script=%+v err=%v", src, script, err)
		}
	}
}

func TestIndentedFenceDoesNotCloseFrontMatter(t *testing.T) {
	if _, err := Parse("open.tbx", "---\nvoice: |\n  ---\n"); err == nil || !strings.Contains(err.Error(), "unclosed front matter") {
		t.Fatalf("err = %v", err)
	}
	script, err := Parse("body.tbx", " ---\nHello")
	if err != nil || len(script.Blocks) != 1 || !strings.Contains(script.Blocks[0].Nodes[0].Text, "---") {
		t.Fatalf("indented opening: %+v %v", script, err)
	}
}

func TestCastUsesFrontMatterAnchor(t *testing.T) {
	for _, cast := range []string{
		"lin: {voices: {openai: *default_voice}}",
		"lin: {name: *default_voice}",
	} {
		src := "---\nvoice: &default_voice coral\ncast:\n  " + cast + "\n---\nlin: Hi"
		script, err := Parse("anchor.tbx", src)
		if err != nil {
			t.Fatal(err)
		}
		member := script.Front.Cast["lin"]
		if member.Name != "coral" && member.Voices["openai"] != "coral" {
			t.Fatalf("cast = %+v", member)
		}
	}
}

func TestCastMergeKeysAndAliases(t *testing.T) {
	src := "---\ncast:\n  first: &first {name: First, voices: {openai: coral}}\n  second: &second {name: Second}\n  lin: {<<: [*first, *second], name: Lin}\n  chen: *first\n---\nlin: Hi"
	script, err := Parse("merge.tbx", src)
	if err != nil {
		t.Fatal(err)
	}
	if got := script.Front.Cast["lin"]; got.Name != "Lin" || got.Voices["openai"] != "coral" {
		t.Fatalf("lin = %+v", got)
	}
	if got := script.Front.Cast["chen"]; got.Name != "First" || got.Voices["openai"] != "coral" {
		t.Fatalf("chen = %+v", got)
	}
}

func TestCastAliasesKeepUnknownFieldsStrict(t *testing.T) {
	for _, cast := range []string{
		"lin: {voics: {openai: coral}}",
		"lin: &member {voics: {openai: coral}}\n  chen: *member",
		"lin: {<<: &member {voics: {openai: coral}}}",
		"lin: {\"<<\": {name: Lin}}",
		"lin: {name: First, name: Second}",
		"lin: &member {<<: *member}",
	} {
		if _, err := Parse("invalid.tbx", "---\ncast:\n  "+cast+"\n---\nHi"); err == nil {
			t.Fatalf("accepted %q", cast)
		}
	}
}

func TestFrontMatterVersionRequiresInteger(t *testing.T) {
	for _, version := range []string{"1.9", "1.0", "1e0", "0.9", ".nan", ".inf", "true", "\"1\"", "[]", "{}", "0", "2"} {
		if _, err := Parse("version.tbx", "---\ntimbrex: "+version+"\n---\nHi"); err == nil {
			t.Fatalf("accepted version %s", version)
		}
	}
	for _, version := range []string{"1", "+1", "0x1", "0o1", "!!int '1'"} {
		script, err := Parse("version.tbx", "---\ntimbrex: "+version+"\n---\nHi")
		if err != nil || script.Front.Timbrex != 1 {
			t.Fatalf("version %s: %+v %v", version, script, err)
		}
	}
	script, err := Parse("version-alias.tbx", "---\nvoice: &version 1\ntimbrex: *version\n---\nHi")
	if err != nil || script.Front.Timbrex != 1 {
		t.Fatalf("integer alias: %+v %v", script, err)
	}
}
