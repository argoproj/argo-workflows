package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestTopSort(t *testing.T) {
	sorted, err := TopSort(wfv1.Nodes{
		"root-1": {Children: []string{"child", "missing"}},
		"child":  {},
		"root-2": {},
	})
	if assert.NoError(t, err) {
		// `sorted`  should contain two sub-lists:
		// 1. "root-2"
		// 2. "child", "missing", "root-1"
		// But the order is not-deterministic.
		if sorted[0] == "root-2" {
			assert.Equal(t, []string{"root-2", "child", "missing", "root-1"}, sorted, "we get a sorted list of all nodes")
		} else {
			assert.Equal(t, []string{"child", "missing", "root-1", "root-2"}, sorted, "we get a sorted list of all nodes")
		}
	}
}
