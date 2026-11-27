package executor

import (
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestReleaseReserve(t *testing.T) {
	maxSeconds := math.MaxInt64 / int64(time.Second)
	for _, tc := range []struct {
		name    string
		value   string
		unset   bool
		want    time.Duration
		wantErr bool
	}{
		{name: "Unset", unset: true, wantErr: true},
		{name: "Empty", value: "", wantErr: true},
		{name: "NotANumber", value: "ten", wantErr: true},
		{name: "NotWhole", value: "1.5", wantErr: true},
		{name: "Zero", value: "0", wantErr: true},
		{name: "Negative", value: "-1", wantErr: true},
		{name: "Overflows", value: strconv.FormatInt(maxSeconds+1, 10), wantErr: true},
		{name: "One", value: "1", want: time.Second},
		{name: "Seconds", value: "30", want: 30 * time.Second},
		{name: "Largest", value: strconv.FormatInt(maxSeconds, 10), want: time.Duration(maxSeconds) * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(common.EnvVarReleaseReserveSeconds, tc.value)
			if tc.unset {
				require.NoError(t, os.Unsetenv(common.EnvVarReleaseReserveSeconds))
			}
			got, err := ReleaseReserve()
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), common.EnvVarReleaseReserveSeconds)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
