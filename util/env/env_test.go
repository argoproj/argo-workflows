package env

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLookupEnvDurationOr(t *testing.T) {
	require.Equal(t, time.Second, LookupEnvDurationOr("", time.Second), "default value")
	t.Setenv("FOO", "bar")
	require.Panics(t, func() { LookupEnvDurationOr("FOO", time.Second) }, "bad value")
	t.Setenv("FOO", "1h")
	require.Equal(t, time.Hour, LookupEnvDurationOr("FOO", time.Second), "env var value")
	t.Setenv("FOO", "")
	require.Equal(t, time.Second, LookupEnvDurationOr("FOO", time.Second), "empty var value; default value")
}

func TestLookupEnvIntOr(t *testing.T) {
	require.Equal(t, 1, LookupEnvIntOr("", 1), "default value")
	t.Setenv("FOO", "not-int")
	require.Panics(t, func() { LookupEnvIntOr("FOO", 1) }, "bad value")
	t.Setenv("FOO", "2")
	require.Equal(t, 2, LookupEnvIntOr("FOO", 1), "env var value")
	t.Setenv("FOO", "")
	require.Equal(t, 1, LookupEnvIntOr("FOO", 1), "empty var value; default value")
}

func TestLookupEnvFloatOr(t *testing.T) {
	require.InEpsilon(t, 1., LookupEnvFloatOr("", 1.), 0.001, "default value")
	t.Setenv("FOO", "not-float")
	require.Panics(t, func() { LookupEnvFloatOr("FOO", 1.) }, "bad value")
	t.Setenv("FOO", "2.0")
	require.InEpsilon(t, 2., LookupEnvFloatOr("FOO", 1.), 0.001, "env var value")
	t.Setenv("FOO", "")
	require.InEpsilon(t, 1., LookupEnvFloatOr("FOO", 1.), 0.001, "empty var value; default value")
}

func TestLookupEnvStringOr(t *testing.T) {
	require.Equal(t, "a", LookupEnvStringOr("", "a"), "default value")
	t.Setenv("FOO", "b")
	require.Equal(t, "b", LookupEnvStringOr("FOO", "a"), "env var value")
	t.Setenv("FOO", "")
	require.Equal(t, "a", LookupEnvStringOr("FOO", "a"), "empty var value; default value")
}
