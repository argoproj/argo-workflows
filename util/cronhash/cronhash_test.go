package cronhash

import (
	"fmt"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpand(t *testing.T) {
	tests := []struct {
		schedule string
		expected string
	}{
		// schedules without a `H` are untouched, whitespace and all
		{"* * * * *", "* * * * *"},
		{"0 0 * * MON-FRI", "0 0 * * MON-FRI"},
		{"@daily", "@daily"},
		{"@every 1h", "@every 1h"},
		{"CRON_TZ=Asia/Tokyo 0 9 * * *", "CRON_TZ=Asia/Tokyo 0 9 * * *"},
		{"0  0 * * *", "0  0 * * *"},
		// `THU` contains a `H`, but not as a token
		{"0 0 * * THU", "0 0 * * THU"},
		// a single `H` resolves to one value of the field's range
		{"H * * * *", "9 * * * *"},
		{"H H * * *", "9 6 * * *"},
		{"H H H H H", "9 6 20 5 3"},
		// `H` is hashed per position, so `H,H` does not resolve to the same value twice
		{"H,H * * * *", "9,30 * * * *"},
		// a step keeps the interval and only offsets the first run
		{"H/15 * * * *", "9-59/15 * * * *"},
		{"H/90 * * * *", "9-59/90 * * * *"},
		// an explicit range limits which values `H` resolves to
		{"H(20-30) * * * *", "26 * * * *"},
		{"H(0-4)/2 * * * *", "1-4/2 * * * *"},
		{"0 H(9-17) * * *", "0 9 * * *"},
		// an explicit range may go beyond the range a bare `H` resolves within, up to what cron
		// itself accepts
		{"0 0 H(29-31) * *", "0 0 31 * *"},
		// `H` mixes with regular values
		{"H,30 H(9-17) * * MON-FRI", "9,30 9 * * MON-FRI"},
		// the timezone prefix is not a field
		{"CRON_TZ=Asia/Tokyo H H * * *", "CRON_TZ=Asia/Tokyo 9 6 * * *"},
		{"TZ=Asia/Tokyo H H * * *", "TZ=Asia/Tokyo 9 6 * * *"},
		// expressions we cannot parse are left to the cron parser
		{"* * * *", "* * * *"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.schedule, func(t *testing.T) {
			expanded, err := Expand(tt.schedule, "argo", "my-cron-wf")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, expanded)
		})
	}
}

func TestExpandIsStable(t *testing.T) {
	first, err := Expand("H H(0-7) * * *", "argo", "my-cron-wf")
	require.NoError(t, err)
	second, err := Expand("H H(0-7) * * *", "argo", "my-cron-wf")
	require.NoError(t, err)
	assert.Equal(t, first, second)

	otherName, err := Expand("H H(0-7) * * *", "argo", "other-cron-wf")
	require.NoError(t, err)
	assert.NotEqual(t, first, otherName)

	otherNamespace, err := Expand("H H(0-7) * * *", "other", "my-cron-wf")
	require.NoError(t, err)
	assert.NotEqual(t, first, otherNamespace)
}

// TestExpandStaysInRange checks that the expanded schedules are valid and within the field ranges,
// whatever the name of the CronWorkflow is
func TestExpandStaysInRange(t *testing.T) {
	for i := range 200 {
		name := fmt.Sprintf("my-cron-wf-%d", i)
		for _, schedule := range []string{"H H H H H", "H/7 H/5 H/3 H/2 H/4", "H(10-20) H(1-5) H(2-6) H(3-9) H(1-4)"} {
			expanded, err := Expand(schedule, "argo", name)
			require.NoError(t, err)
			parsed, err := cron.ParseStandard(expanded)
			require.NoErrorf(t, err, "%q expanded to %q", schedule, expanded)
			// a schedule which never fires would return the zero time
			assert.NotZerof(t, parsed.Next(time.Now()), "%q expanded to %q", schedule, expanded)
		}
	}
}

func TestExpandMalformed(t *testing.T) {
	tests := []struct {
		schedule string
		errorMsg string
	}{
		{"H(0-30 * * * *", `"H(0-30" is malformed: missing ` + "`)`"},
		{"H(30-10) * * * *", `"H(30-10)" is malformed: range "30-10" is inverted`},
		{"H(0-90) * * * *", `"H(0-90)" is malformed: range "0-90" is outside of 0-59, the range of the minute field`},
		{"* H(0-30) * * *", `"H(0-30)" is malformed: range "0-30" is outside of 0-23, the range of the hour field`},
		{"* * H(0-31) * *", `"H(0-31)" is malformed: range "0-31" is outside of 1-31, the range of the day of month field`},
		// 7 is not a second Sunday here, the cron parser this feeds only accepts 0-6
		{"* * * * H(0-7)", `"H(0-7)" is malformed: range "0-7" is outside of 0-6, the range of the day of week field`},
		{"H(a-b) * * * *", `"H(a-b)" is malformed: expected a range like ` + "`1-5`" + `, got "a-b"`},
		{"H/0 * * * *", `"H/0" is malformed: step must be greater than 0`},
		{"H15 * * * *", `"H15" is malformed: expected `},
	}
	for _, tt := range tests {
		t.Run(tt.schedule, func(t *testing.T) {
			_, err := Expand(tt.schedule, "argo", "my-cron-wf")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestExpandAll(t *testing.T) {
	expanded, err := ExpandAll([]string{"H * * * *", "0 0 * * *"}, "argo", "my-cron-wf")
	require.NoError(t, err)
	assert.Equal(t, []string{"9 * * * *", "0 0 * * *"}, expanded)

	_, err = ExpandAll([]string{"0 0 * * *", "H/0 * * * *"}, "argo", "my-cron-wf")
	require.Error(t, err)
}
