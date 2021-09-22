package expand

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpand(t *testing.T) {
	for i := 0; i < 100; i++ { // loop 100 times, because map ordering is not determisitic
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			before := map[string]interface{}{
				"a.b": 1,
				"a":   2,
				"ab":  3,
			}
			after := Expand(before)
			assert.Len(t, before, 3, "original map unchanged")
			assert.Equal(t, map[string]interface{}{
				"a": map[string]interface{}{
					"b": 1,
				},
				"ab": 3,
			}, after)
		})
	}
}
