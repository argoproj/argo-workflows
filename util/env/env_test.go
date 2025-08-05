package env

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLookupEnvDurationOr(t *testing.T) {
	assert.Equal(t, time.Second, LookupEnvDurationOr("", time.Second), "default value")
	t.Setenv("FOO", "bar")
	assert.Panics(t, func() { LookupEnvDurationOr("FOO", time.Second) }, "bad value")
	t.Setenv("FOO", "1h")
	assert.Equal(t, time.Hour, LookupEnvDurationOr("FOO", time.Second), "env var value")
	t.Setenv("FOO", "")
	assert.Equal(t, time.Second, LookupEnvDurationOr("FOO", time.Second), "empty var value; default value")
}

func TestLookupEnvIntOr(t *testing.T) {
	assert.Equal(t, 1, LookupEnvIntOr("", 1), "default value")
	t.Setenv("FOO", "not-int")
	assert.Panics(t, func() { LookupEnvIntOr("FOO", 1) }, "bad value")
	t.Setenv("FOO", "2")
	assert.Equal(t, 2, LookupEnvIntOr("FOO", 1), "env var value")
	t.Setenv("FOO", "")
	assert.Equal(t, 1, LookupEnvIntOr("FOO", 1), "empty var value; default value")
}

func TestLookupEnvFloatOr(t *testing.T) {
	assert.InEpsilon(t, 1., LookupEnvFloatOr("", 1.), 0.001, "default value")
	t.Setenv("FOO", "not-float")
	assert.Panics(t, func() { LookupEnvFloatOr("FOO", 1.) }, "bad value")
	t.Setenv("FOO", "2.0")
	assert.InEpsilon(t, 2., LookupEnvFloatOr("FOO", 1.), 0.001, "env var value")
	t.Setenv("FOO", "")
	assert.InEpsilon(t, 1., LookupEnvFloatOr("FOO", 1.), 0.001, "empty var value; default value")
}

func TestLookupEnvStringOr(t *testing.T) {
	assert.Equal(t, "a", LookupEnvStringOr("", "a"), "default value")
	t.Setenv("FOO", "b")
	assert.Equal(t, "b", LookupEnvStringOr("FOO", "a"), "env var value")
	t.Setenv("FOO", "")
	assert.Equal(t, "a", LookupEnvStringOr("FOO", "a"), "empty var value; default value")
}
