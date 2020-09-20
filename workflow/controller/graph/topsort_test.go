package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestTopSort(t *testing.T) {
	sorted, err := TopSort(wfv1.Nodes{
		"root":  wfv1.NodeStatus{Children: []string{"child"}},
		"child": wfv1.NodeStatus{},
	}, "root")
	if assert.NoError(t, err) {
		assert.Equal(t, []string{"child", "root"}, sorted)
	}
}
