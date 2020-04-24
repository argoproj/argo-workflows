package util

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestMergeParameters(t *testing.T){
	one := "one"
	two := "two"
	param1 :=[]wfv1.Parameter{
		{
			Name: "p1",
			Value: &one,
		},
		{
			Name: "p2",

		},
	}
	param2 :=[]wfv1.Parameter{
		{
			Name: "p1",
			Value: &two,
		},
		{
			Name: "p3",
		},
	}
	t.Run("MergeParameter-1", func(t *testing.T){
		result :=MergeParameters(param1, param2)
		assert.Equal(t, len(result), 3)
		for _, item := range result{
			if item.Name == "p1"{
				assert.Equal(t, "one",*item.Value)
			}
		}
	})
	t.Run("MergeParameter-2", func(t *testing.T){
		result :=MergeParameters(param2, param1)
		assert.Equal(t, len(result), 3)
		for _, item := range result{
			if item.Name == "p1"{
				assert.Equal(t, "two",*item.Value)
			}
		}
	})

}

func TestMergeVolume(t *testing.T){
	vol1 :=[]v1.Volume{
		{
			Name: "p1",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "one",
				},
			},
		},
		{
			Name: "p2",

		},
	}
	vol2 :=[]v1.Volume{
		{
			Name: "p1",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "two",
				},
			},
		},
		{
			Name: "p3",
		},
	}
	t.Run("MergeVolume-1", func(t *testing.T){
		result :=MergeVolume(vol1, vol2)
		assert.Equal(t, len(result), 3)
		for _, item := range result{
			if item.Name == "p1"{
				assert.Equal(t, "one",item.HostPath.Path)
			}
		}
	})
	t.Run("MergeVolume-2", func(t *testing.T){
		result :=MergeVolume(vol2, vol1)
		assert.Equal(t, len(result), 3)
		for _, item := range result{
			if item.Name == "p1"{
				assert.Equal(t, "two",item.HostPath.Path)
			}
		}
	})

}