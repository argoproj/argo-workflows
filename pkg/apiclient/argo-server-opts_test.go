package apiclient

import (
	"testing"

	"gotest.tools/assert"
)

func TestArgoServerOpts_String(t *testing.T) {
	assert.Equal(t, "(url=my-url,secure=false,insecureSkipVerify=false,http=false)", ArgoServerOpts{URL: "my-url"}.String())
	assert.Equal(t, "(url=,secure=true,insecureSkipVerify=false,http=false)", ArgoServerOpts{Secure: true}.String())
	assert.Equal(t, "(url=,secure=false,insecureSkipVerify=true,http=false)", ArgoServerOpts{InsecureSkipVerify: true}.String())
	assert.Equal(t, "(url=,secure=false,insecureSkipVerify=false,http=true)", ArgoServerOpts{HTTP: true}.String())
}
