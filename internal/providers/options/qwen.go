package options

import (
	"fmt"
	"maps"
	"slices"
	"unicode/utf8"
)

// Qwen overrides the corresponding input fields on each Qwen-Audio-TTS request.
type Qwen struct {
	Seed   *int64      `yaml:"seed,omitempty"`
	HotFix *QwenHotFix `yaml:"hot_fix,omitempty"`
}

// QwenHotFix preserves the distinction between omitted and explicitly empty lists.
type QwenHotFix struct {
	Pronunciation *[]QwenTextPair `yaml:"pronunciation,omitempty"`
	Replace       *[]QwenTextPair `yaml:"replace,omitempty"`
}

// QwenTextPair becomes a single-entry object inside an ordered hot_fix array.
type QwenTextPair struct {
	Text        string `yaml:"text"`
	Replacement string `yaml:"replacement"`
}

func (Qwen) providerOptions() {}

// Validate checks input constraints before request preparation.
func (v Qwen) Validate() error {
	// https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api
	if v.Seed != nil && (*v.Seed < 0 || *v.Seed > 65535) {
		return fmt.Errorf("seed must be an integer between 0 and 65535")
	}
	if v.HotFix == nil {
		return nil
	}
	for _, field := range []struct {
		name  string
		pairs *[]QwenTextPair
	}{
		{"pronunciation", v.HotFix.Pronunciation},
		{"replace", v.HotFix.Replace},
	} {
		if field.pairs == nil {
			continue
		}
		for i, pair := range *field.pairs {
			// Local policy: a rule must identify source text. The API does not
			// document empty-key behavior or duplicate/overlap precedence.
			if pair.Text == "" {
				return fmt.Errorf("hot_fix.%s[%d].text must not be empty", field.name, i)
			}
			// Local lossless-JSON policy: encoding/json replaces invalid UTF-8.
			if !utf8.ValidString(pair.Text) {
				return fmt.Errorf("hot_fix.%s[%d].text must be valid UTF-8", field.name, i)
			}
			if !utf8.ValidString(pair.Replacement) {
				return fmt.Errorf("hot_fix.%s[%d].replacement must be valid UTF-8", field.name, i)
			}
		}
	}
	return nil
}

func decodeQwen(raw any) (Qwen, error) {
	var value Qwen
	fields, err := synthesisFields(raw)
	if err != nil {
		return value, err
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		switch key {
		case "seed":
			value.Seed, err = synthesisInteger(fields[key], key)
		case "hot_fix":
			value.HotFix, err = decodeQwenHotFix(fields[key])
		default:
			return value, fmt.Errorf("unknown synthesis option")
		}
		if err != nil {
			return value, err
		}
	}
	return value, value.Validate()
}

func decodeQwenHotFix(raw any) (*QwenHotFix, error) {
	fields, err := synthesisFields(raw)
	if err != nil {
		return nil, fmt.Errorf("hot_fix: %w", err)
	}
	value := &QwenHotFix{}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		switch key {
		case "pronunciation":
			value.Pronunciation, err = decodeQwenTextPairs(fields[key])
		case "replace":
			value.Replace, err = decodeQwenTextPairs(fields[key])
		default:
			return nil, fmt.Errorf("unknown hot_fix option")
		}
		if err != nil {
			return nil, fmt.Errorf("hot_fix.%s: %w", key, err)
		}
	}
	return value, nil
}

func decodeQwenTextPairs(raw any) (*[]QwenTextPair, error) {
	items, err := synthesisList(raw)
	if err != nil {
		return nil, err
	}
	result := make([]QwenTextPair, 0, len(items))
	for _, item := range items {
		fields, err := synthesisFields(item)
		if err != nil {
			return nil, err
		}
		for _, key := range slices.Sorted(maps.Keys(fields)) {
			if key != "text" && key != "replacement" {
				return nil, fmt.Errorf("unknown text pair option")
			}
		}
		text, err := synthesisString[string](fields["text"], "text")
		if err != nil {
			return nil, err
		}
		replacement, err := synthesisString[string](fields["replacement"], "replacement")
		if err != nil {
			return nil, err
		}
		result = append(result, QwenTextPair{Text: *text, Replacement: *replacement})
	}
	return &result, nil
}
