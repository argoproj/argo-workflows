package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestDatabaseConfig(t *testing.T) {
	assert.Equal(t, "my-host", DatabaseConfig{Host: "my-host"}.GetHostname())
	assert.Equal(t, "my-host:1234", DatabaseConfig{Host: "my-host", Port: 1234}.GetHostname())
}

func TestDBConfigConnectionTimeout(t *testing.T) {
	// Defaults to 5s when unset.
	assert.Equal(t, 5*time.Second, DBConfig{}.ConnectionTimeout())
	// Honors an explicit value.
	assert.Equal(t, 12*time.Second, DBConfig{ConnectionTimeoutSeconds: 12}.ConnectionTimeout())
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		c   Config
		err string
	}{
		{Config{Links: []*wfv1.Link{{URL: "javascript:foo"}}}, "protocol javascript is not allowed"},
		{Config{Links: []*wfv1.Link{{URL: "javASCRipt: //foo"}}}, "protocol javascript is not allowed"},
		{Config{Links: []*wfv1.Link{{URL: "http://foo.bar/?foo=<script>abc</script>bar"}}}, ""},
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

func TestDBRetryConfig(t *testing.T) {
	defaults := (*DBRetryConfig)(nil).Backoff()
	assert.Equal(t, 5, defaults.Steps)
	assert.Equal(t, 10*time.Millisecond, defaults.Duration)
	assert.InEpsilon(t, 2.0, defaults.Factor, 0.001)
	assert.InEpsilon(t, 0.5, defaults.Jitter, 0.001)
	assert.Equal(t, 600*time.Millisecond, defaults.Cap)

	// an unset field keeps its default rather than becoming zero
	partial := (&DBRetryConfig{Steps: 10}).Backoff()
	assert.Equal(t, 10, partial.Steps)
	assert.Equal(t, 10*time.Millisecond, partial.Duration)
	assert.InEpsilon(t, 2.0, partial.Factor, 0.001)

	assert.True(t, (*DBRetryConfig)(nil).RequeueEnabled())
	assert.True(t, (&DBRetryConfig{Steps: 10}).RequeueEnabled())
	assert.False(t, (&DBRetryConfig{Requeue: new(false)}).RequeueEnabled())
}
