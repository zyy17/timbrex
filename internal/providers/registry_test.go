package providers

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/ir"
)

func TestRegisterDuplicatePanics(t *testing.T) {
	const id = "dup-test"
	delete(registry, id)
	t.Cleanup(func() { delete(registry, id) })
	p := idProvider{id: id}
	Register(p)
	defer func() {
		rec := recover()
		if rec == nil || !strings.Contains(fmt.Sprint(rec), `duplicate provider "dup-test"`) {
			t.Fatalf("panic = %v", rec)
		}
	}()
	Register(p)
}

func TestRegisterEmptyIDPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	Register(idProvider{})
}

type idProvider struct{ id string }

func (p idProvider) ID() string { return p.id }

func (p idProvider) Synthesize(context.Context, *ir.Document, Options) (Result, error) {
	return Result{}, nil
}
