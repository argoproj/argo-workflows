package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

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

func TestHotWorkersConfigDefaults(t *testing.T) {
	c := &Config{}
	require.NoError(t, parseConfigMap(&apiv1.ConfigMap{}, c))
	require.Nil(t, c.HotWorkers)
	assert.False(t, c.HotWorkers.IsEnabled())
	assert.Equal(t, int64(300), c.HotWorkers.GetLeaseSlackSeconds())
	assert.Equal(t, int64(600), c.HotWorkers.GetDefaultRetentionSeconds())
	assert.Equal(t, int64(30), c.HotWorkers.GetTerminationGracePeriodSeconds())
	assert.Equal(t, int64(10), c.HotWorkers.GetReleaseReserveSeconds())
	require.NoError(t, c.HotWorkers.Validate())

	c = &Config{}
	require.NoError(t, parseConfigMap(&apiv1.ConfigMap{Data: map[string]string{"hotWorkers": "enabled: true\nleaseSlackSeconds: 60\n"}}, c))
	assert.True(t, c.HotWorkers.IsEnabled())
	assert.Equal(t, int64(60), c.HotWorkers.GetLeaseSlackSeconds())
	assert.Equal(t, int64(600), c.HotWorkers.GetDefaultRetentionSeconds(), "unset values keep their defaults")
	require.NoError(t, c.HotWorkers.Validate())
}

func TestHotWorkersConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  HotWorkersConfig
		wantErr string
	}{
		{name: "custom values", config: HotWorkersConfig{LeaseSlackSeconds: new(int64(1)), DefaultRetentionSeconds: new(int64(1)), TerminationGracePeriodSeconds: new(int64(60)), ReleaseReserveSeconds: new(int64(59))}},
		{name: "zero lease slack", config: HotWorkersConfig{LeaseSlackSeconds: new(int64(0))}, wantErr: "hotWorkers.leaseSlackSeconds must be positive, got 0"},
		{name: "negative retention", config: HotWorkersConfig{DefaultRetentionSeconds: new(int64(-1))}, wantErr: "hotWorkers.defaultRetentionSeconds must be positive, got -1"},
		{name: "zero grace period", config: HotWorkersConfig{TerminationGracePeriodSeconds: new(int64(0))}, wantErr: "hotWorkers.terminationGracePeriodSeconds must be positive, got 0"},
		{name: "zero release reserve", config: HotWorkersConfig{ReleaseReserveSeconds: new(int64(0))}, wantErr: "hotWorkers.releaseReserveSeconds must be positive, got 0"},
		{name: "release reserve below the minimum", config: HotWorkersConfig{ReleaseReserveSeconds: new(int64(2))}, wantErr: "hotWorkers.releaseReserveSeconds (2) must be at least 3, to leave time for main's SIGKILL escalation, the worker noticing main has stopped, and the claim's deletion"},
		{name: "release reserve at the minimum", config: HotWorkersConfig{TerminationGracePeriodSeconds: new(int64(4)), ReleaseReserveSeconds: new(int64(3))}},
		{name: "release reserve equal to grace period", config: HotWorkersConfig{TerminationGracePeriodSeconds: new(int64(20)), ReleaseReserveSeconds: new(int64(20))}, wantErr: "hotWorkers.releaseReserveSeconds (20) must be less than hotWorkers.terminationGracePeriodSeconds (20)"},
		{name: "release reserve above the default grace period", config: HotWorkersConfig{ReleaseReserveSeconds: new(int64(45))}, wantErr: "hotWorkers.releaseReserveSeconds (45) must be less than hotWorkers.terminationGracePeriodSeconds (30)"},
		{name: "grace period below the default release reserve", config: HotWorkersConfig{TerminationGracePeriodSeconds: new(int64(5))}, wantErr: "hotWorkers.releaseReserveSeconds (10) must be less than hotWorkers.terminationGracePeriodSeconds (5)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}
