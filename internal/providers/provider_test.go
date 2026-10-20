package providers_test

import (
	"testing"

	"github.com/zyy17/timbrex/internal/providers"
	_ "github.com/zyy17/timbrex/internal/providers/stub"
)

func TestRegistry(t *testing.T) {
	ids := providers.IDs()
	if len(ids) != 1 || ids[0] != "stub" {
		t.Fatalf("ids = %v", ids)
	}
	p, err := providers.Get("stub")
	if err != nil || p.ID() != "stub" {
		t.Fatalf("get = %v %v", p, err)
	}
	if _, err := providers.Get("missing"); err == nil || err.Error() != `unknown provider "missing"` {
		t.Fatalf("err = %v", err)
	}
}
