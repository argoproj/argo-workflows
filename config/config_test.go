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

func TestAgentConfig_SetDefaults(t *testing.T) {
	t.Run("CreatePod defaults to true when not set", func(t *testing.T) {
		ac := &AgentConfig{}
		ac.SetDefaults()
		assert.True(t, *ac.CreatePod, "CreatePod should default to true")
	})

	t.Run("DeleteAfterCompletion defaults to true for per-workflow agent", func(t *testing.T) {
		ac := &AgentConfig{RunMultipleWorkflow: false}
		ac.SetDefaults()
		assert.True(t, *ac.DeleteAfterCompletion, "DeleteAfterCompletion should default to true for per-workflow agent")
	})

	t.Run("DeleteAfterCompletion not set for multi-workflow agent", func(t *testing.T) {
		ac := &AgentConfig{RunMultipleWorkflow: true}
		ac.SetDefaults()
		assert.Nil(t, ac.DeleteAfterCompletion, "DeleteAfterCompletion should not be set for multi-workflow agent")
	})

	t.Run("Does not override existing values", func(t *testing.T) {
		createPod := false
		deleteAfter := false
		ac := &AgentConfig{
			CreatePod:             &createPod,
			DeleteAfterCompletion: &deleteAfter,
		}
		ac.SetDefaults()
		assert.False(t, *ac.CreatePod, "CreatePod should not be overridden")
		assert.False(t, *ac.DeleteAfterCompletion, "DeleteAfterCompletion should not be overridden")
	})
}

func TestAgentConfig_ShouldCreatePod(t *testing.T) {
	t.Run("Returns true when nil config", func(t *testing.T) {
		var ac *AgentConfig
		assert.True(t, ac.ShouldCreatePod(), "Should default to true for nil config")
	})

	t.Run("Returns true when CreatePod is nil", func(t *testing.T) {
		ac := &AgentConfig{}
		assert.True(t, ac.ShouldCreatePod(), "Should default to true when CreatePod is nil")
	})

	t.Run("Returns false when CreatePod is false", func(t *testing.T) {
		createPod := false
		ac := &AgentConfig{CreatePod: &createPod}
		assert.False(t, ac.ShouldCreatePod(), "Should return false when CreatePod is false")
	})

	t.Run("Returns true when CreatePod is true", func(t *testing.T) {
		createPod := true
		ac := &AgentConfig{CreatePod: &createPod}
		assert.True(t, ac.ShouldCreatePod(), "Should return true when CreatePod is true")
	})
}

func TestAgentConfig_ShouldDeleteAfterCompletion(t *testing.T) {
	t.Run("Returns true when nil config", func(t *testing.T) {
		var ac *AgentConfig
		assert.True(t, ac.ShouldDeleteAfterCompletion(), "Should default to true for nil config")
	})

	t.Run("Returns true when DeleteAfterCompletion is nil", func(t *testing.T) {
		ac := &AgentConfig{}
		assert.True(t, ac.ShouldDeleteAfterCompletion(), "Should default to true when DeleteAfterCompletion is nil")
	})

	t.Run("Returns false when DeleteAfterCompletion is false", func(t *testing.T) {
		deleteAfter := false
		ac := &AgentConfig{DeleteAfterCompletion: &deleteAfter}
		assert.False(t, ac.ShouldDeleteAfterCompletion(), "Should return false when DeleteAfterCompletion is false")
	})

	t.Run("Returns true when DeleteAfterCompletion is true", func(t *testing.T) {
		deleteAfter := true
		ac := &AgentConfig{DeleteAfterCompletion: &deleteAfter}
		assert.True(t, ac.ShouldDeleteAfterCompletion(), "Should return true when DeleteAfterCompletion is true")
	})
}
