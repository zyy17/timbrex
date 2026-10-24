package fish

import (
	"encoding/json"
	"fmt"

	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

// synthesisSnapshot is detached once per invocation and then shared read-only
// across prepared requests. Direct builder calls receive independent snapshots.
type synthesisSnapshot struct {
	value      options.Fish
	dictionary *pronunciationDictionary
}

type pronunciationDictionary struct {
	references []options.FishDictionaryReference
	inline     []options.FishDictionaryInline
}

func (v pronunciationDictionary) MarshalJSON() ([]byte, error) {
	if v.references != nil {
		return json.Marshal(v.references)
	}
	return json.Marshal(v.inline)
}

func synthesisOptions(value options.Value) (synthesisSnapshot, error) {
	var v options.Fish
	switch value := value.(type) {
	case nil:
	case options.Fish:
		v = value
	default:
		return synthesisSnapshot{}, fmt.Errorf("fish received an unsupported synthesis options type")
	}
	if err := v.Validate(); err != nil {
		return synthesisSnapshot{}, err
	}
	s := synthesisSnapshot{value: options.Fish{
		Normalize: snapshot(v.Normalize), NormalizationLanguage: snapshot(v.NormalizationLanguage),
		Temperature: snapshot(v.Temperature), TopP: snapshot(v.TopP),
		ChunkLength: snapshot(v.ChunkLength), MinChunkLength: snapshot(v.MinChunkLength),
		Latency: snapshot(v.Latency), ConditionOnPreviousChunks: snapshot(v.ConditionOnPreviousChunks),
		EarlyStopThreshold: snapshot(v.EarlyStopThreshold),
	}}
	if v.Prosody != nil {
		s.value.Prosody = &options.FishProsody{NormalizeLoudness: snapshot(v.Prosody.NormalizeLoudness)}
	}
	if v.Features != nil {
		features := make([]options.FishFeature, len(*v.Features))
		copy(features, *v.Features)
		s.value.Features = &features
	}
	if dictionaries := v.PronunciationDictionary; dictionaries != nil {
		s.dictionary = &pronunciationDictionary{}
		if dictionaries.References != nil {
			s.dictionary.references = make([]options.FishDictionaryReference, len(*dictionaries.References))
			copy(s.dictionary.references, *dictionaries.References)
		} else {
			s.dictionary.inline = make([]options.FishDictionaryInline, len(*dictionaries.Inline))
			for i, dictionary := range *dictionaries.Inline {
				items := make([]options.FishPronunciationItem, len(dictionary.Items))
				for j, item := range dictionary.Items {
					items[j] = options.FishPronunciationItem{Key: item.Key, Value: item.Value, CaseSensitive: snapshot(item.CaseSensitive)}
				}
				s.dictionary.inline[i] = options.FishDictionaryInline{Items: items}
			}
		}
	}
	return s, nil
}

func buildWithSynthesis(seg providers.Segment, model string, value options.Fish) (request, error) {
	snapshot, err := synthesisOptions(value)
	if err != nil {
		return request{}, err
	}
	return snapshot.build(seg, model)
}

func (s synthesisSnapshot) build(seg providers.Segment, model string) (request, error) {
	r, err := build(seg, model)
	if err != nil {
		return r, err
	}
	value := s.value
	if value.Normalize != nil {
		r.Normalize = *value.Normalize
	}
	if value.Prosody != nil && value.Prosody.NormalizeLoudness != nil {
		r.Prosody.NormalizeLoudness = *value.Prosody.NormalizeLoudness
	}
	r.Language = value.NormalizationLanguage
	r.Temperature = value.Temperature
	r.TopP = value.TopP
	r.ChunkLength = value.ChunkLength
	r.MinChunkLength = value.MinChunkLength
	r.Latency = value.Latency
	r.ConditionOnPreviousChunks = value.ConditionOnPreviousChunks
	r.EarlyStopThreshold = value.EarlyStopThreshold
	r.Features = value.Features
	r.PronunciationDictionary = s.dictionary
	return r, nil
}

func snapshot[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
