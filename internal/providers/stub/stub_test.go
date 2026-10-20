package stub

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/providers"
)

func TestWritesWAV(t *testing.T) {
	p, err := providers.Get("stub")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.wav")
	doc := &ir.Document{Utterances: []ir.Utterance{{
		Speaker: "lin",
		Nodes:   []ir.Node{{Kind: ir.Text, Text: "Hello"}},
	}}}
	res, err := p.Synthesize(context.Background(), doc, providers.Options{Out: out})
	if err != nil {
		t.Fatal(err)
	}
	if res.Provider != "stub" || res.Path != out {
		t.Fatalf("result = %+v", res)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 44 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		t.Fatalf("not a wav: %d bytes", len(b))
	}
}
