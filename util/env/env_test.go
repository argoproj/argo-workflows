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
	_ = os.Setenv("FOO", "")
	assert.Equal(t, time.Second, LookupEnvDurationOr("FOO", time.Second), "empty var value; default value")
}

func TestLookupEnvIntOr(t *testing.T) {
	defer func() { _ = os.Unsetenv("FOO") }()
	assert.Equal(t, 1, LookupEnvIntOr("", 1), "default value")
	_ = os.Setenv("FOO", "not-int")
	assert.Panics(t, func() { LookupEnvIntOr("FOO", 1) }, "bad value")
	_ = os.Setenv("FOO", "2")
	assert.Equal(t, 2, LookupEnvIntOr("FOO", 1), "env var value")
	_ = os.Setenv("FOO", "")
	assert.Equal(t, 1, LookupEnvIntOr("FOO", 1), "empty var value; default value")
}

func TestLookupEnvFloatOr(t *testing.T) {
	defer func() { _ = os.Unsetenv("FOO") }()
	assert.Equal(t, 1., LookupEnvFloatOr("", 1.), "default value")
	_ = os.Setenv("FOO", "not-float")
	assert.Panics(t, func() { LookupEnvFloatOr("FOO", 1.) }, "bad value")
	_ = os.Setenv("FOO", "2.0")
	assert.Equal(t, 2., LookupEnvFloatOr("FOO", 1.), "env var value")
	_ = os.Setenv("FOO", "")
	assert.Equal(t, 1., LookupEnvFloatOr("FOO", 1.), "empty var value; default value")
}

func TestLookupEnvStringOr(t *testing.T) {
	defer func() { _ = os.Unsetenv("FOO") }()
	assert.Equal(t, "a", LookupEnvStringOr("", "a"), "default value")
	_ = os.Setenv("FOO", "b")
	assert.Equal(t, "b", LookupEnvStringOr("FOO", "a"), "env var value")
	_ = os.Setenv("FOO", "")
	assert.Equal(t, "a", LookupEnvStringOr("FOO", "a"), "empty var value; default value")
}
