package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestConfig_GetServiceAccount(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		var c *Config
		ref, err := c.GetServiceAccount(nil)
		assert.NoError(t, err)
		assert.Nil(t, ref)
	})
	t.Run("Empty", func(t *testing.T) {
		_, err := (&Config{}).GetServiceAccount(nil)
		assert.EqualError(t, err, "no RBAC rules match")
	})
	t.Run("DefaultServiceAccount", func(t *testing.T) {
		serviceAccountRef := &corev1.LocalObjectReference{Name: "my-sa"}
		ref, err := (&Config{DefaultServiceAccountRef: serviceAccountRef}).GetServiceAccount(nil)
		if assert.NoError(t, err) {
			assert.Equal(t, serviceAccountRef, ref)
		}
	})
	t.Run("RulesNoMatch", func(t *testing.T) {
		_, err := (&Config{Rules: []Rule{{}}}).GetServiceAccount(nil)
		assert.EqualError(t, err, "no RBAC rules match")
	})
	t.Run("RulesMatch", func(t *testing.T) {
		serviceAccountRef := corev1.LocalObjectReference{Name: "my-sa"}
		ref, err := (&Config{Rules: []Rule{{AnyOf: []string{"my-group"}, ServiceAccountRef: serviceAccountRef}}}).GetServiceAccount([]string{"my-group"})
		if assert.NoError(t, err) {
			assert.Equal(t, &serviceAccountRef, ref)
		}
	})
}
