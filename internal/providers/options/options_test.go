package options

import (
	"reflect"
	"testing"
)

func TestDecodeSelectsProviderSchema(t *testing.T) {
	for _, tc := range []struct {
		id   string
		want Value
	}{
		{"elevenlabs", ElevenLabs{}}, {"fish", Fish{}},
		{"qwen", Qwen{}}, {"minimax", MiniMax{}},
	} {
		got, err := Decode(tc.id, map[string]any{})
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: value=%#v error=%v", tc.id, got, err)
		}
	}
	for _, id := range []string{"stub", "openai", "azure", "gemini", "unknown"} {
		if _, err := Decode(id, map[string]any{}); err == nil {
			t.Errorf("accepted unsupported schema %q", id)
		}
	}
}
