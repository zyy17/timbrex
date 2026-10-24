package cli

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func TestAzureConfiguredDeploymentConcurrency(t *testing.T) {
	vendor := synthesisVendor(t, "azure")
	baseline, baselineWAV := runConcurrentContract(t, vendor, 0, "")
	for _, workers := range []int{0, 1, 2, 4, 8} {
		t.Run(fmt.Sprint(workers), func(t *testing.T) {
			requests, wav := runConcurrentContract(t, vendor, workers, `providers: {azure: {synthesis: {deployment_id: 'opaque / ? & % + # 部署'}}}`)
			if !bytes.Equal(wav, baselineWAV) {
				t.Fatal("deployment routing changed ordered audio or local pauses")
			}
			for key, request := range requests {
				if request.Query != "deploymentId=opaque+%2F+%3F+%26+%25+%2B+%23+%E9%83%A8%E7%BD%B2" {
					t.Fatal("configured deployment ID was not encoded exactly once")
				}
				request.Query = ""
				if !reflect.DeepEqual(request, baseline[key]) {
					t.Fatal("deployment option changed request path, SSML, headers, or method")
				}
			}
		})
	}
}
