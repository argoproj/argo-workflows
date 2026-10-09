package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

func TestPersistConfigGetOperationTimeout(t *testing.T) {
	intPtr := func(v int32) *int32 { return &v }

	t.Run("defaults to 30s when unset", func(t *testing.T) {
		assert.Equal(t, 30*time.Second, PersistConfig{}.GetOperationTimeout())
	})

	t.Run("honors an explicit positive value", func(t *testing.T) {
		assert.Equal(t, 90*time.Second, PersistConfig{OperationTimeoutSeconds: intPtr(90)}.GetOperationTimeout())
	})

	t.Run("falls back to default when zero", func(t *testing.T) {
		assert.Equal(t, 30*time.Second, PersistConfig{OperationTimeoutSeconds: intPtr(0)}.GetOperationTimeout())
	})

	t.Run("falls back to default when negative", func(t *testing.T) {
		assert.Equal(t, 30*time.Second, PersistConfig{OperationTimeoutSeconds: intPtr(-5)}.GetOperationTimeout())
	})
}

func TestPersistConfigGetTemplateOffloadMinSize(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	t.Run("defaults to 256KiB when unset", func(t *testing.T) {
		assert.Equal(t, DefaultTemplateOffloadMinSize, PersistConfig{}.GetTemplateOffloadMinSize())
		assert.Equal(t, 262144, PersistConfig{}.GetTemplateOffloadMinSize())
	})

	t.Run("honors an explicit positive value", func(t *testing.T) {
		assert.Equal(t, 1024, PersistConfig{TemplateOffloadMinSize: intPtr(1024)}.GetTemplateOffloadMinSize())
	})

	t.Run("falls back to default when zero", func(t *testing.T) {
		assert.Equal(t, DefaultTemplateOffloadMinSize, PersistConfig{TemplateOffloadMinSize: intPtr(0)}.GetTemplateOffloadMinSize())
	})

	t.Run("falls back to default when negative", func(t *testing.T) {
		assert.Equal(t, DefaultTemplateOffloadMinSize, PersistConfig{TemplateOffloadMinSize: intPtr(-1)}.GetTemplateOffloadMinSize())
	})
}

func TestPersistConfigValidate(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	t.Run("accepts offload with unset min size", func(t *testing.T) {
		assert.NoError(t, PersistConfig{TemplateOffload: true}.Validate())
	})
	t.Run("accepts offload with positive min size", func(t *testing.T) {
		assert.NoError(t, PersistConfig{TemplateOffload: true, TemplateOffloadMinSize: intPtr(1024)}.Validate())
	})
	t.Run("rejects offload with zero min size", func(t *testing.T) {
		require.EqualError(t, PersistConfig{TemplateOffload: true, TemplateOffloadMinSize: intPtr(0)}.Validate(),
			"templateOffloadMinSize must be greater than 0 when templateOffLoad is enabled")
	})
	t.Run("rejects offload with negative min size", func(t *testing.T) {
		assert.Error(t, PersistConfig{TemplateOffload: true, TemplateOffloadMinSize: intPtr(-1)}.Validate())
	})
	t.Run("allows zero min size when offload disabled", func(t *testing.T) {
		assert.NoError(t, PersistConfig{TemplateOffloadMinSize: intPtr(0)}.Validate())
	})
}

func TestPersistConfigGetTemplateHydrationFallbackSettings(t *testing.T) {
	durationPtr := func(d time.Duration) *metav1.Duration { return &metav1.Duration{Duration: d} }
	intPtr := func(v int) *int { return &v }

	t.Run("defaults when unset", func(t *testing.T) {
		cfg := PersistConfig{}
		assert.Equal(t, 15, cfg.GetTemplateHydrationFallbackRetries())
		assert.Equal(t, 500*time.Millisecond, cfg.GetTemplateHydrationFallbackBackoff())
		assert.Equal(t, 120*time.Second, cfg.GetTemplateHydrationFallbackBackoffMax())
	})
	t.Run("honors explicit values", func(t *testing.T) {
		cfg := PersistConfig{
			TemplateHydrationFallbackRetries:    intPtr(3),
			TemplateHydrationFallbackBackoff:    durationPtr(time.Second),
			TemplateHydrationFallbackBackoffMax: durationPtr(10 * time.Second),
		}
		assert.Equal(t, 3, cfg.GetTemplateHydrationFallbackRetries())
		assert.Equal(t, time.Second, cfg.GetTemplateHydrationFallbackBackoff())
		assert.Equal(t, 10*time.Second, cfg.GetTemplateHydrationFallbackBackoffMax())
	})
	t.Run("clamps negative values to zero", func(t *testing.T) {
		cfg := PersistConfig{
			TemplateHydrationFallbackRetries:    intPtr(-2),
			TemplateHydrationFallbackBackoff:    durationPtr(-time.Second),
			TemplateHydrationFallbackBackoffMax: durationPtr(-time.Minute),
		}
		assert.Equal(t, 0, cfg.GetTemplateHydrationFallbackRetries())
		assert.Equal(t, time.Duration(0), cfg.GetTemplateHydrationFallbackBackoff())
		assert.Equal(t, time.Duration(0), cfg.GetTemplateHydrationFallbackBackoffMax())
	})
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
