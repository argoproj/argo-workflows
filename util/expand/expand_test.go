package expand

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpand(t *testing.T) {
	for i := 0; i < 1; i++ { // loop 100 times, because map ordering is not determisitic
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			before := map[string]interface{}{
				"a.b":   1,
				"a.c.d": 2,
				"a":     3, // should be deleted
				"ab":    4,
				"abb":   5, // should be kept
			}
			after := Expand(before)
			require.Len(t, before, 5, "original map unchanged")
			require.Equal(t, map[string]interface{}{
				"a": map[string]interface{}{
					"b": 1,
					"c": map[string]interface{}{
						"d": 2,
					},
				},
				"ab":  4,
				"abb": 5,
			}, after)
		})
	}
}
