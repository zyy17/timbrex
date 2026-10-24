package config

import (
	"strings"
	"testing"

	"github.com/zyy17/timbrex/internal/providers/options"
)

func TestAzureSynthesisConfiguration(t *testing.T) {
	const endpoint = "https://eastus.voice.speech.microsoft.com/cognitiveservices/v1"
	const id = "opaque / ? & % + # 部署"
	c, err := loadTest(t, `providers:
  azure:
    api_key: FAKE-SECRET-DO-NOT-LOG
    endpoint: `+endpoint+`
    region: westus
    voice: MyCustomVoice
    synthesis:
      deployment_id: '`+id+`'
`)
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.Resolve("azure", "", "")
	if err != nil || p.Endpoint != endpoint || p.Region != "westus" || p.Voice != "MyCustomVoice" {
		t.Fatalf("custom endpoint or voice changed: %v", err)
	}
	value, ok := p.Synthesis.(options.Azure)
	if !ok || value.DeploymentID == nil || *value.DeploymentID != id {
		t.Fatalf("deployment options changed: %T", p.Synthesis)
	}
	redacted, err := c.RedactedYAML()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(redacted), "FAKE-SECRET-DO-NOT-LOG") || !strings.Contains(string(redacted), "api_key: REDACTED") || !strings.Contains(string(redacted), "deployment_id:") {
		t.Fatal("redaction leaked a key or lost deployment routing")
	}
	roundTrip, err := loadTest(t, string(redacted))
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip.Providers["azure"].Synthesis.(options.Azure)
	if got.DeploymentID == nil || *got.DeploymentID != id || c.Providers["azure"].Endpoint != endpoint || c.Providers["azure"].APIKey != "FAKE-SECRET-DO-NOT-LOG" {
		t.Fatal("round trip or redaction changed configuration")
	}
}

func TestAzureSynthesisPresencePreservesDefaultEndpoint(t *testing.T) {
	for _, body := range []string{"", "synthesis: {}"} {
		c, err := loadTest(t, "providers:\n  azure:\n    api_key: test-key\n    region: eastus\n    "+body+"\n")
		if err != nil {
			t.Fatal(err)
		}
		p, err := c.Resolve("azure", "", "")
		if err != nil || p.Endpoint != "https://eastus.tts.speech.microsoft.com/cognitiveservices/v1" {
			t.Fatalf("ordinary endpoint changed: %v", err)
		}
		if body == "" {
			if p.Synthesis != nil {
				t.Fatal("omission created synthesis options")
			}
		} else if value, ok := p.Synthesis.(options.Azure); !ok || value.DeploymentID != nil {
			t.Fatal("empty options created a deployment override")
		}
	}
}

func TestAzureDeploymentRequiresExplicitEndpoint(t *testing.T) {
	for _, region := range []string{"", "eastus"} {
		c, err := loadTest(t, "providers:\n  azure:\n    api_key: test-key\n    region: '"+region+"'\n    synthesis: {deployment_id: secret-value}\n")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Resolve("azure", "", ""); err == nil || !strings.Contains(err.Error(), "providers.azure.endpoint") || strings.Contains(err.Error(), "secret-value") {
			t.Fatalf("missing custom endpoint accepted or leaked ID: %v", err)
		}
		if _, err := c.Resolve("stub", "", ""); err != nil {
			t.Fatalf("unselected deployment required an endpoint: %v", err)
		}
	}
	const endpoint = "https://westeurope.voice.speech.microsoft.com/cognitiveservices/v1"
	c, err := loadTest(t, "providers: {azure: {api_key: test-key, region: eastus, synthesis: {deployment_id: existing}}}", "AZURE_SPEECH_ENDPOINT="+endpoint)
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.Resolve("azure", "", "")
	if err != nil || p.Endpoint != endpoint {
		t.Fatalf("explicit environment endpoint ignored: %v", err)
	}
}

func TestRejectInvalidAzureSynthesisConfiguration(t *testing.T) {
	for _, value := range []string{
		"null", "[]", "true", "secret-value",
		"{deployment_id: null}", "{deployment_id: 1}", "{deployment_id: false}",
		"{deployment_id: []}", "{deployment_id: {secret-value: true}}",
		"{deployment_id: ''}", "{deployment_id: ' '}", "{deployment_id: \"secret-value\\n\"}",
		"{deployment_id: \"secret-value\\0\"}", "{deployment_id: \"secret-value\\u0085\"}",
		"{deploymentId: secret-value}", "{secret-value: 1}", "{deployment_id: a, deployment_id: b}",
		"{deployment_id.secret-value: x}",
	} {
		t.Run(value, func(t *testing.T) {
			_, err := loadTest(t, "provider: stub\nproviders: {azure: {synthesis: "+value+"}}")
			if err == nil || strings.Contains(err.Error(), "secret-value") {
				t.Fatalf("invalid unselected Azure options accepted or leaked: %v", err)
			}
		})
	}
	for _, endpoint := range []string{
		"https://example.com?deploymentId=secret-value",
		"https://example.com?deploymentId=one&deploymentId=secret-value",
		"https://example.com?key=secret-value",
		"https://user:secret-value@example.com/speech",
		"https://example.com/speech#secret-value",
	} {
		_, err := loadTest(t, "providers: {azure: {endpoint: '"+endpoint+"', synthesis: {deployment_id: existing}}}")
		if err == nil || strings.Contains(err.Error(), "secret-value") {
			t.Fatalf("endpoint restrictions weakened or value leaked: %v", err)
		}
	}
}
