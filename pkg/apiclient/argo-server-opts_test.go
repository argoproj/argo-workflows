package apiclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgoServerOpts_String(t *testing.T) {
	assert.Equal(t, "(url=my-url,path=/my-path,secure=false,insecureSkipVerify=false,http=false)", ArgoServerOpts{URL: "my-url", Path: "/my-path"}.String())
	assert.Equal(t, "(url=,path=,secure=true,insecureSkipVerify=false,http=false)", ArgoServerOpts{Secure: true}.String())
	assert.Equal(t, "(url=,path=,secure=false,insecureSkipVerify=true,http=false)", ArgoServerOpts{InsecureSkipVerify: true}.String())
	assert.Equal(t, "(url=,path=,secure=false,insecureSkipVerify=false,http=true)", ArgoServerOpts{HTTP1: true}.String())
}

func TestArgoServerOpts_GetURL(t *testing.T) {
	assert.Equal(t, "http://my-url/my-path", ArgoServerOpts{URL: "my-url", Path: "/my-path"}.GetURL())
	assert.Equal(t, "https://my-url/my-path", ArgoServerOpts{URL: "my-url", Path: "/my-path", Secure: true}.GetURL())
}
