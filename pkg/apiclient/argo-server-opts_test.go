package apiclient

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArgoServerOpts_String(t *testing.T) {
	require.Equal(t, "(url=my-url,path=/my-path,secure=false,insecureSkipVerify=false,http=false)", ArgoServerOpts{URL: "my-url", Path: "/my-path"}.String())
	require.Equal(t, "(url=,path=,secure=true,insecureSkipVerify=false,http=false)", ArgoServerOpts{Secure: true}.String())
	require.Equal(t, "(url=,path=,secure=false,insecureSkipVerify=true,http=false)", ArgoServerOpts{InsecureSkipVerify: true}.String())
	require.Equal(t, "(url=,path=,secure=false,insecureSkipVerify=false,http=true)", ArgoServerOpts{HTTP1: true}.String())
}

func TestArgoServerOpts_GetURL(t *testing.T) {
	require.Equal(t, "http://my-url/my-path", ArgoServerOpts{URL: "my-url", Path: "/my-path"}.GetURL())
	require.Equal(t, "https://my-url/my-path", ArgoServerOpts{URL: "my-url", Path: "/my-path", Secure: true}.GetURL())
}
