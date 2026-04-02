package wait

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestExponentialBackoff2(t *testing.T) {
	t.Run("NoError", func(t *testing.T) {
		err := Backoff(wait.Backoff{Steps: 1}, func() (bool, error) {
			return true, nil
		})
		require.NoError(t, err)
	})
	t.Run("Error", func(t *testing.T) {
		err := Backoff(wait.Backoff{Steps: 1}, func() (bool, error) {
			return true, errors.New("foo")
		})
		require.EqualError(t, err, "foo")
	})
	t.Run("Timeout", func(t *testing.T) {
		err := Backoff(wait.Backoff{Steps: 1}, func() (bool, error) {
			return false, nil
		})
		assert.Equal(t, err, wait.ErrorInterrupted(err))
	})
	t.Run("TimeoutError", func(t *testing.T) {
		err := Backoff(wait.Backoff{Steps: 1}, func() (bool, error) {
			return false, errors.New("foo")
		})
		require.EqualError(t, err, "timed out waiting for the condition: foo")
	})
}
