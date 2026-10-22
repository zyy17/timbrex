package providers_test

import (
	"testing"

	"github.com/zyy17/timbrex/internal/providers"
)

func TestSpeedUsesParsedArgumentTypes(t *testing.T) {
	for _, arg := range []string{"1.2", "120%"} {
		segs, err := providers.Plan(document(t, "[speed "+arg+"|Hi]"), "openai", "coral")
		if err != nil {
			t.Fatal(err)
		}
		speed, err := providers.Speed(segs[0].Controls[0], 0.25, 4)
		if err != nil || speed != 1.2 {
			t.Fatalf("%q: speed=%g, error=%v", arg, speed, err)
		}
	}
}
