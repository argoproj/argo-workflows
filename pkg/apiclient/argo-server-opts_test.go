package apiclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgoServerOpts_String(t *testing.T) {
	tests := []struct {
		name     string
		opts     ArgoServerOpts
		expected string
	}{
		{
			name:     "should format options with URL and RootPath",
			opts:     ArgoServerOpts{URL: "argo.example.com", RootPath: "/my-path"},
			expected: "(url=argo.example.com,rootPath=/my-path,baseHRef=,secure=false,insecureSkipVerify=false,http=false)",
		},
		{
			name:     "should format secure option",
			opts:     ArgoServerOpts{Secure: true},
			expected: "(url=,rootPath=,baseHRef=,secure=true,insecureSkipVerify=false,http=false)",
		},
		{
			name:     "should format insecure skip verify option",
			opts:     ArgoServerOpts{InsecureSkipVerify: true},
			expected: "(url=,rootPath=,baseHRef=,secure=false,insecureSkipVerify=true,http=false)",
		},
		{
			name:     "should format HTTP1 option",
			opts:     ArgoServerOpts{HTTP1: true},
			expected: "(url=,rootPath=,baseHRef=,secure=false,insecureSkipVerify=false,http=true)",
		},
		{
			name:     "should format all options together",
			opts:     ArgoServerOpts{URL: "argo.example.com", RootPath: "/api", BaseHRef: "/ui", Secure: true, InsecureSkipVerify: true, HTTP1: true},
			expected: "(url=argo.example.com,rootPath=/api,baseHRef=/ui,secure=true,insecureSkipVerify=true,http=true)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.opts.String())
		})
	}
}

func TestArgoServerOpts_GetURL(t *testing.T) {
	tests := []struct {
		name     string
		opts     ArgoServerOpts
		expected string
	}{
		{
			name:     "should construct HTTP URL with root path",
			opts:     ArgoServerOpts{URL: "argo.example.com", RootPath: "/api/v1"},
			expected: "http://argo.example.com/api/v1",
		},
		{
			name:     "should construct HTTPS URL with root path when secure",
			opts:     ArgoServerOpts{URL: "argo.example.com", RootPath: "/api/v1", Secure: true},
			expected: "https://argo.example.com/api/v1",
		},
		{
			name:     "should construct HTTP URL without root path when empty",
			opts:     ArgoServerOpts{URL: "argo.example.com"},
			expected: "http://argo.example.com",
		},
		{
			name:     "should construct HTTP URL without root path when explicitly empty",
			opts:     ArgoServerOpts{URL: "argo.example.com", RootPath: ""},
			expected: "http://argo.example.com",
		},
		{
			name:     "should use only root path for API URL when both BaseHRef and RootPath are present",
			opts:     ArgoServerOpts{URL: "argo.example.com", RootPath: "/api/v1", BaseHRef: "/argo"},
			expected: "http://argo.example.com/api/v1",
		},
		{
			name:     "should construct URL without path when only BaseHRef is present",
			opts:     ArgoServerOpts{URL: "argo.example.com", BaseHRef: "/argo"},
			expected: "http://argo.example.com",
		},
		{
			name:     "should handle complex root path",
			opts:     ArgoServerOpts{URL: "argo.example.com", RootPath: "/workflows/api/v1", Secure: false},
			expected: "http://argo.example.com/workflows/api/v1",
		},
		{
			name:     "should handle HTTPS with complex configuration",
			opts:     ArgoServerOpts{URL: "argo.example.com", RootPath: "/workflows", Secure: true, InsecureSkipVerify: true},
			expected: "https://argo.example.com/workflows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.opts.GetURL())
		})
	}
}
