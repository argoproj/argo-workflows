package v1alpha1

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStringToDuration(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    time.Duration
		wantErr string
	}{
		{
			name:  "maximum whole seconds",
			value: "9223372036",
			want:  time.Duration(9223372036) * time.Second,
		},
		{
			name:  "minimum whole seconds",
			value: "-9223372036",
			want:  time.Duration(-9223372036) * time.Second,
		},
		{
			name:    "whole seconds above maximum",
			value:   "9223372037",
			wantErr: "whole seconds overflow time.Duration",
		},
		{
			name:    "whole seconds below minimum",
			value:   "-9223372037",
			wantErr: "whole seconds overflow time.Duration",
		},
		{
			name:  "duration syntax fallback",
			value: "1.5s",
			want:  1500 * time.Millisecond,
		},
		{
			name:  "maximum duration string round trip",
			value: time.Duration(math.MaxInt64).String(),
			want:  time.Duration(math.MaxInt64),
		},
		{
			name:  "minimum duration string round trip",
			value: time.Duration(math.MinInt64).String(),
			want:  time.Duration(math.MinInt64),
		},
		{
			name:    "invalid duration",
			value:   "not-a-duration",
			wantErr: "unable to parse not-a-duration as a duration",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseStringToDuration(test.value)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}
