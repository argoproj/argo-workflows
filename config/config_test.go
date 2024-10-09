package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestDatabaseConfig(t *testing.T) {
	assert.Equal(t, "my-host", DatabaseConfig{Host: "my-host"}.GetHostname())
	assert.Equal(t, "my-host:1234", DatabaseConfig{Host: "my-host", Port: 1234}.GetHostname())
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		c   Config
		err string
	}{
		{Config{Links: []*wfv1.Link{{URL: "javascript:foo"}}}, "protocol javascript is not allowed"},
		{Config{Links: []*wfv1.Link{{URL: "javASCRipt: //foo"}}}, "protocol javascript is not allowed"},
		{Config{Links: []*wfv1.Link{{URL: "http://foo.bar/?foo=<script>abc</script>bar"}}}, ""},
		{Config{Links: []*wfv1.Link{{URL: "/my-namespace"}}}, ""},
		{Config{Links: []*wfv1.Link{{URL: "?namespace=argo-events&phase=Failed&phase=Error&limit=50"}}}, ""},
	}
	for _, tt := range tests {
		err := tt.c.Sanitize([]string{"http", "https"})
		if tt.err != "" {
			require.EqualError(t, err, tt.err)
		} else {
			require.NoError(t, err)
		}
	}
}
