package providers

import (
	"context"
	"fmt"

	"github.com/zyy17/timbrex/internal/ir"
)

// Prepare builds every speech request before rendering can make any network calls.
// Requests retain segment indices; local pauses have zero-value requests.
func Prepare[T any](ctx context.Context, doc *ir.Document, id, voice string, build func(Segment) (T, error)) ([]Segment, []T, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	segments, err := plan(ctx, doc, id, voice)
	if err != nil {
		return nil, nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	requests := make([]T, len(segments))
	for i, seg := range segments {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if seg.Pause > 0 {
			continue
		}
		requests[i], err = build(seg)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", id, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	return segments, requests, nil
}
