package util

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestMergeParameters(t *testing.T) {
	one := "one"
	two := "two"
	param1 := []wfv1.Parameter{
		{
			Name:  "p1",
			Value: &one,
		},
		{
			Name: "p2",
		},
	}
	param2 := []wfv1.Parameter{
		{
			Name:  "p1",
			Value: &two,
		},
		{
			Name: "p3",
		},
	}
	t.Run("MergeParameter-1", func(t *testing.T) {
		result := MergeParameters(param1, param2)
		assert.Equal(t, len(result), 3)
		for _, item := range result {
			if item.Name == "p1" {
				assert.Equal(t, "one", *item.Value)
			}
		}
	})
	t.Run("MergeParameter-2", func(t *testing.T) {
		result := MergeParameters(param2, param1)
		assert.Equal(t, len(result), 3)
		for _, item := range result {
			if item.Name == "p1" {
				assert.Equal(t, "two", *item.Value)
			}
		}
	})

}
