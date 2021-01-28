package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	apierr "k8s.io/apimachinery/pkg/api/errors"
)

func TestDone(t *testing.T) {
	t.Run("NoError", func(t *testing.T) {
		done, err := Done(nil)
		assert.NoError(t, err)
		assert.True(t, done)
	})
	t.Run("TransientError", func(t *testing.T) {
		done, err := Done(apierr.NewTooManyRequests("", 0))
		assert.NoError(t, err)
		assert.False(t, done)
	})
	t.Run("NonTransientError", func(t *testing.T) {
		done, err := Done(errors.New(""))
		assert.Error(t, err)
		assert.True(t, done)
	})
}
