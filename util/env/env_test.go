package env

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLookupEnvDurationOr(t *testing.T) {
	defer func() { _ = os.Unsetenv("FOO") }()
	assert.Equal(t, time.Second, LookupEnvDurationOr("", time.Second), "default value")
	_ = os.Setenv("FOO", "bar")
	assert.Panics(t, func() { LookupEnvDurationOr("FOO", time.Second) }, "bad value")
	_ = os.Setenv("FOO", "1h")
	assert.Equal(t, time.Hour, LookupEnvDurationOr("FOO", time.Second), "env var value")
}

func TestLookupEnvIntOr(t *testing.T) {
	defer func() { _ = os.Unsetenv("FOO") }()
	assert.Equal(t, 1, LookupEnvIntOr("", 1), "default value")
	_ = os.Setenv("FOO", "bar")
	assert.Panics(t, func() { LookupEnvDurationOr("FOO", 1) }, "bad value")
	_ = os.Setenv("FOO", "2")
	assert.Equal(t, 2, LookupEnvIntOr("FOO", 1), "env var value")
}
