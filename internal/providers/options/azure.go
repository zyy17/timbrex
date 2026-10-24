package options

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Azure routes synthesis to one existing custom voice deployment.
// A nil DeploymentID preserves the adapter's ordinary endpoint behavior.
type Azure struct {
	DeploymentID *string `yaml:"deployment_id,omitempty"`
}

func (Azure) providerOptions() {}

// Validate treats deployment IDs as opaque strings, not UUIDs or voice names.
func (v Azure) Validate() error {
	// Local input hygiene only; the REST reference specifies no ID alphabet.
	// https://learn.microsoft.com/azure/ai-services/speech-service/rest-text-to-speech#custom-voices
	if v.DeploymentID != nil && (!utf8.ValidString(*v.DeploymentID) || strings.TrimSpace(*v.DeploymentID) == "" || strings.IndexFunc(*v.DeploymentID, unicode.IsControl) >= 0) {
		return fmt.Errorf("deployment_id must be a nonblank UTF-8 string without control characters")
	}
	return nil
}

func decodeAzure(raw any) (Azure, error) {
	var value Azure
	fields, err := synthesisFields(raw)
	if err != nil {
		return value, err
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if key != "deployment_id" {
			return value, fmt.Errorf("unknown synthesis option")
		}
		value.DeploymentID, err = synthesisString[string](fields[key], key)
		if err != nil {
			return value, err
		}
	}
	return value, value.Validate()
}
