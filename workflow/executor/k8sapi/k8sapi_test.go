package k8sapi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_backoffOver30s(t *testing.T) {
	x := backoffOver30s
	assert.Equal(t, 1*time.Second, x.Step())
	assert.Equal(t, 2*time.Second, x.Step())
	assert.Equal(t, 4*time.Second, x.Step())
	assert.Equal(t, 8*time.Second, x.Step())
	assert.Equal(t, 16*time.Second, x.Step())
	assert.Equal(t, 32*time.Second, x.Step())
	assert.Equal(t, 64*time.Second, x.Step())
}
