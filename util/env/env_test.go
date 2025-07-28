package env

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestLookupEnvDurationOr(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	assert.Equal(t, time.Second, LookupEnvDurationOr(ctx, "", time.Second), "default value")
	t.Setenv("FOO", "bar")
	assert.Panics(t, func() { LookupEnvDurationOr(ctx, "FOO", time.Second) }, "bad value")
	t.Setenv("FOO", "1h")
	assert.Equal(t, time.Hour, LookupEnvDurationOr(ctx, "FOO", time.Second), "env var value")
	t.Setenv("FOO", "")
	assert.Equal(t, time.Second, LookupEnvDurationOr(ctx, "FOO", time.Second), "empty var value; default value")
}

func TestLookupEnvIntOr(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	assert.Equal(t, 1, LookupEnvIntOr(ctx, "", 1), "default value")
	t.Setenv("FOO", "not-int")
	assert.Panics(t, func() { LookupEnvIntOr(ctx, "FOO", 1) }, "bad value")
	t.Setenv("FOO", "2")
	assert.Equal(t, 2, LookupEnvIntOr(ctx, "FOO", 1), "env var value")
	t.Setenv("FOO", "")
	assert.Equal(t, 1, LookupEnvIntOr(ctx, "FOO", 1), "empty var value; default value")
}

func TestLookupEnvFloatOr(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	assert.InEpsilon(t, 1., LookupEnvFloatOr(ctx, "", 1.), 0.001, "default value")
	t.Setenv("FOO", "not-float")
	assert.Panics(t, func() { LookupEnvFloatOr(ctx, "FOO", 1.) }, "bad value")
	t.Setenv("FOO", "2.0")
	assert.InEpsilon(t, 2., LookupEnvFloatOr(ctx, "FOO", 1.), 0.001, "env var value")
	t.Setenv("FOO", "")
	assert.InEpsilon(t, 1., LookupEnvFloatOr(ctx, "FOO", 1.), 0.001, "empty var value; default value")
}

func TestLookupEnvStringOr(t *testing.T) {
	assert.Equal(t, "a", LookupEnvStringOr("", "a"), "default value")
	t.Setenv("FOO", "b")
	assert.Equal(t, "b", LookupEnvStringOr("FOO", "a"), "env var value")
	t.Setenv("FOO", "")
	assert.Equal(t, "a", LookupEnvStringOr("FOO", "a"), "empty var value; default value")
}
