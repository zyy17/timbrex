package providers

import (
	"fmt"

	"github.com/zyy17/timbrex/internal/ir"
)

// Prepare builds every speech request before rendering can make any network calls.
// Requests retain segment indices; local pauses have zero-value requests.
func Prepare[T any](doc *ir.Document, id, voice string, build func(Segment) (T, error)) ([]Segment, []T, error) {
	segments, err := Plan(doc, id, voice)
	if err != nil {
		return nil, nil, err
	}
	requests := make([]T, len(segments))
	for i, seg := range segments {
		if seg.Pause > 0 {
			continue
		}
		requests[i], err = build(seg)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", id, err)
		}
	}
	return segments, requests, nil
}
