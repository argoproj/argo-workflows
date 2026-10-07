package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitPluginNames(t *testing.T) {
	tests := map[string]struct {
		raw  string
		want []string
	}{
		"empty":               {raw: "", want: nil},
		"single":              {raw: "s3-driver", want: []string{"s3-driver"}},
		"multi with spaces":   {raw: "s3-driver, gcs-driver ,  oci ", want: []string{"s3-driver", "gcs-driver", "oci"}},
		"skips blank entries": {raw: "s3,,gcs, ,oci", want: []string{"s3", "gcs", "oci"}},
		"trailing comma":      {raw: "s3-driver,", want: []string{"s3-driver"}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, SplitPluginNames(tc.raw))
		})
	}
}
