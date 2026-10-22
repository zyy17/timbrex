package providers_test

import (
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/providers"
)

func TestAsUnknownSpeakerIsRejected(t *testing.T) {
	doc := document(t, "---\ncast:\n  lin: {voices: {stub: tone}}\n---\nlin: [as ghost|Hello]\n")
	_, err := providers.Plan(doc, "stub", "fallback")
	if err == nil || !strings.Contains(err.Error(), "[as ghost]") {
		t.Fatalf("err = %v", err)
	}
}

func TestAsDeclaredWithoutMappingFallsBack(t *testing.T) {
	doc := document(t, "---\ncast:\n  lin: {}\n  chen: {}\n---\nlin: [as chen|Hello]\n")
	segs, err := providers.Plan(doc, "stub", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if segs[0].Voice != "fallback" {
		t.Fatalf("voice=%q", segs[0].Voice)
	}
}

func TestAsAutoSpeakerWithoutTurnIsRejected(t *testing.T) {
	doc := document(t, "---\ncast: auto\n---\nHello [as lin|there]\n")
	_, err := providers.Plan(doc, "stub", "fallback")
	if err == nil || !strings.Contains(err.Error(), "[as lin]") {
		t.Fatalf("err = %v", err)
	}
}

func TestAsAutoSpeakerWithTurnFallsBack(t *testing.T) {
	doc := document(t, "---\ncast: auto\n---\nlin: Hello\nchen: [as lin|there]\n")
	segs, err := providers.Plan(doc, "stub", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 2 || segs[1].Voice != "fallback" {
		t.Fatalf("segs = %+v", segs)
	}
}
