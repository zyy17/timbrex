package azure

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/zyy17/timbrex/internal/providers"
	"github.com/zyy17/timbrex/internal/providers/options"
)

func synthesisEndpoint(opt providers.Options) (string, error) {
	var v options.Azure
	switch value := opt.Synthesis.(type) {
	case nil:
	case options.Azure:
		v = value
	default:
		return "", fmt.Errorf("azure received an unsupported synthesis options type")
	}
	if err := v.Validate(); err != nil {
		return "", err
	}
	if v.DeploymentID == nil {
		return opt.Endpoint, nil
	}
	// Validate the query-free base even for callers that bypass configuration.
	// One invocation supports one deployment, not preexisting query parameters.
	u, err := url.Parse(opt.Endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.ForceQuery || strings.Contains(opt.Endpoint, "#") {
		return "", fmt.Errorf("deployment_id requires an explicit HTTP(S) endpoint without credentials, query, or fragment")
	}
	// Custom deployment hosts use .voice., not the standard regional .tts. host.
	// Keep caller-supplied proxy/custom endpoints unchanged; never guess a region.
	// https://learn.microsoft.com/azure/ai-services/speech-service/rest-text-to-speech#custom-voices
	// https://learn.microsoft.com/azure/ai-services/speech-service/sovereign-clouds
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	for _, suffix := range []string{".tts.speech.microsoft.com", ".tts.speech.azure.us", ".tts.speech.azure.cn"} {
		if strings.HasSuffix(host, suffix) {
			return "", fmt.Errorf("deployment_id requires a custom voice endpoint, not a standard regional TTS endpoint")
		}
	}
	u.RawQuery = url.Values{"deploymentId": {*v.DeploymentID}}.Encode()
	return u.String(), nil
}
