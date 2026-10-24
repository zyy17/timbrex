package options

import (
	"strings"
	"testing"
)

func TestAzureDeploymentID(t *testing.T) {
	if err := (Azure{}).Validate(); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"opaque-id", "d", "not a UUID", "  keep spaces  ", "voice / & ? % + #", "部署-α"} {
		if err := (Azure{DeploymentID: &id}).Validate(); err != nil {
			t.Fatalf("opaque ID rejected: %v", err)
		}
	}
	for _, id := range []string{"", " ", "\u2003", "secret-value\n", "secret-value\t", "secret-value\x00", "secret-value\x7f", "secret-value\u0085", "secret-value\xff"} {
		if err := (Azure{DeploymentID: &id}).Validate(); err == nil || strings.Contains(err.Error(), "secret-value") {
			t.Fatalf("invalid ID accepted or leaked: %v", err)
		}
	}
}
