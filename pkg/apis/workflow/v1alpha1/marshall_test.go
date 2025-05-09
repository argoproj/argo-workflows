package v1alpha1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustUnmarshalClusterWorkflowTemplate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("The code did not panic but should have")
		} else {
			assert.Equal(t, fmt.Errorf("no text to unmarshal"), r)
		}
	}()
	_ = MustUnmarshalClusterWorkflowTemplate([]byte(""))
	t.Fatalf("MustUnmarshalClusterWorkflowTemplate should have panicked and this part should not have been reached.")
}
