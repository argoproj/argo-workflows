package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestDatabaseConfig(t *testing.T) {
	assert.Equal(t, "my-host", DatabaseConfig{Host: "my-host"}.GetHostname())
	assert.Equal(t, "my-host:1234", DatabaseConfig{Host: "my-host", Port: 1234}.GetHostname())
}

func TestSanitize(t *testing.T) {
	c := Config{
		Links: []*wfv1.Link{
			{URL: "javascript:foo"},
			{URL: "javASCRipt: //foo"},
			{URL: "http://foo.bar/?foo=<script>abc</script>bar"},
		},
	}
	c.Sanitize([]string{"http", "https"})
	assert.Equal(t, "", c.Links[0].URL)
	assert.Equal(t, "", c.Links[1].URL)
	assert.Equal(t, "http://foo.bar/?foo=&lt;script&gt;abc&lt;/script&gt;bar", c.Links[2].URL)
}
