package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustUnmarshalClusterWorkflow(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("The code did not panic but should have")
		} else {
			assert.Equal(t, "no text to unmarshal", r.(string))
		}
	}()
	_ = MustUnmarshalClusterWorkflow([]byte(""))
	t.Fatalf("MustUnmarshalClusterWorkflow should have panicked and this part should not have been reached.")
}
