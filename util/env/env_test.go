package env

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// TestLookupEnvPositiveMilliseconds covers optional values, positive integer
// milliseconds, and both integer-parsing and duration-conversion overflow.
func TestLookupEnvPositiveMilliseconds(t *testing.T) {
	for _, tt := range []struct {
		name    string
		value   string
		unset   bool
		delay   *time.Duration
		invalid bool
	}{
		{name: "unset", unset: true},
		{name: "empty"},
		{name: "zero", value: "0", invalid: true},
		{name: "one millisecond", value: "1", delay: new(time.Millisecond)},
		{name: "subsecond", value: "250", delay: new(250 * time.Millisecond)},
		{name: "one second", value: "1000", delay: new(time.Second)},
		{name: "multiple seconds", value: "20000", delay: new(20 * time.Second)},
		{name: "maximum", value: "9223372036854", delay: new(9223372036854 * time.Millisecond)},
		{name: "negative", value: "-1", invalid: true},
		{name: "malformed", value: "not-an-integer", invalid: true},
		{name: "whitespace", value: " ", invalid: true},
		{name: "fractional", value: "0.5", invalid: true},
		{name: "duration suffix", value: "250ms", invalid: true},
		{name: "boolean", value: "true", invalid: true},
		{name: "duration overflow", value: "9223372036855", invalid: true},
		{name: "integer overflow", value: "9223372036854775808", invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const key = "TEST_POSITIVE_MILLISECONDS"
			t.Setenv(key, tt.value)
			if tt.unset {
				require.NoError(t, os.Unsetenv(key))
			}
			delay, err := LookupEnvPositiveMilliseconds(key)
			if tt.invalid {
				require.ErrorContains(t, err, key+" must be an integer between 1 and 9223372036854")
				assert.Contains(t, err.Error(), tt.value)
				assert.Nil(t, delay)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.delay, delay)
		})
	}
}

func TestLookupEnvIntInRange(t *testing.T) {
	for _, tt := range []struct {
		name    string
		value   string
		unset   bool
		def     int
		lo      int
		hi      int
		want    int
		invalid bool
	}{
		{name: "unset", unset: true, def: 20, lo: 1, hi: 100, want: 20},
		{name: "empty", def: 20, lo: 1, hi: 100, want: 20},
		{name: "lower bound", value: "1", lo: 1, hi: 100, want: 1},
		{name: "within range", value: "20", lo: 1, hi: 100, want: 20},
		{name: "upper bound", value: "100", lo: 1, hi: 100, want: 100},
		{name: "zero", value: "0", lo: 1, hi: 100, invalid: true},
		{name: "negative", value: "-1", lo: 1, hi: 100, invalid: true},
		{name: "above range", value: "101", lo: 1, hi: 100, invalid: true},
		{name: "suffix", value: "20%", lo: 1, hi: 100, invalid: true},
		{name: "fractional", value: "0.2", lo: 1, hi: 100, invalid: true},
		{name: "leading whitespace", value: " 20", lo: 1, hi: 100, invalid: true},
		{name: "trailing whitespace", value: "20 ", lo: 1, hi: 100, invalid: true},
		{name: "malformed", value: "invalid", lo: 1, hi: 100, invalid: true},
		{name: "overflow", value: "9223372036854775808", lo: 1, hi: 100, invalid: true},
		{name: "negative range", value: "-5", lo: -10, hi: -1, want: -5},
		{name: "zero in range", value: "0", lo: -1, hi: 1, want: 0},
		{name: "single value range", value: "5", lo: 5, hi: 5, want: 5},
		{name: "below custom range", value: "4", lo: 5, hi: 10, invalid: true},
		{name: "above custom range", value: "11", lo: 5, hi: 10, invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const key = "TEST_INT_IN_RANGE"
			t.Setenv(key, tt.value)
			if tt.unset {
				require.NoError(t, os.Unsetenv(key))
			}
			got, err := LookupEnvIntInRange(key, tt.def, tt.lo, tt.hi)
			if tt.invalid {
				require.ErrorContains(t, err, fmt.Sprintf("%s must be an integer between %d and %d", key, tt.lo, tt.hi))
				assert.Contains(t, err.Error(), fmt.Sprintf("%q", tt.value))
				assert.Zero(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

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
