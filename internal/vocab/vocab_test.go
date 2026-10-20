package vocab

import "testing"

func TestLookupFoldsCaseAndDash(t *testing.T) {
	spec, ok := Lookup("Clear_Throat")
	if !ok || spec.Name != "clear-throat" || spec.Kind != Event {
		t.Fatalf("spec = %+v ok=%v", spec, ok)
	}
	if _, ok := Lookup("not-a-tag"); ok {
		t.Fatal("unknown tag looked up")
	}
}

func TestActAllowsNote(t *testing.T) {
	spec, ok := Lookup("act")
	if !ok || !spec.Note || spec.Kind != State {
		t.Fatalf("act = %+v", spec)
	}
}

func TestWrapNeedsContent(t *testing.T) {
	spec, ok := Lookup("pinyin")
	if !ok || !spec.NeedContent || !spec.Content {
		t.Fatalf("pinyin = %+v", spec)
	}
}
