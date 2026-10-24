// Package providers turns an IR document into an audio file.
//
// Each vendor lives in its own subpackage, registers in init, and follows
// that vendor's documented API. The core does not import vendor SDKs.
package providers

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/zyy17/timbrex/internal/ir"
	"github.com/zyy17/timbrex/internal/provideropts"
)

// Options contains resolved settings. Providers do not load files or environment variables.
type Options struct {
	Out       string
	Model     string
	Voice     string
	APIKey    string
	Endpoint  string
	Synthesis provideropts.Value
	// Concurrency bounds uncommitted segments; zero preserves serial synthesis.
	Concurrency int
}

// Result is one successful synthesis.
type Result struct {
	Path     string
	Provider string
}

// Provider renders a document. ID is the name used by --provider and by
// the provider key in front matter.
type Provider interface {
	ID() string
	Synthesize(ctx context.Context, doc *ir.Document, opt Options) (Result, error)
}

var registry = map[string]Provider{}

// Register adds a provider. The ID must be unique.
// A duplicate id panics. Registration runs from init, which cannot return an error.
func Register(p Provider) {
	if p == nil || p.ID() == "" {
		panic("timbrex: provider is missing an id")
	}
	if _, ok := registry[p.ID()]; ok {
		panic("timbrex: duplicate provider " + strconv.Quote(p.ID()))
	}
	registry[p.ID()] = p
}

// Get returns a registered provider.
func Get(id string) (Provider, error) {
	p, ok := registry[id]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", id)
	}
	return p, nil
}

// IDs lists registered provider names in order.
func IDs() []string {
	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ValidateConcurrency accepts zero for callers using the serial default.
func ValidateConcurrency(n int) error {
	if n < 0 || n > 8 {
		return fmt.Errorf("concurrency must be between 1 and 8 (or 0 for the serial default)")
	}
	return nil
}

// Validate checks invocation options before planning, requests, or output writes.
func (o Options) Validate(id string) error {
	if err := ValidateConcurrency(o.Concurrency); err != nil {
		return err
	}
	return provideropts.Validate(id, o.Synthesis)
}
