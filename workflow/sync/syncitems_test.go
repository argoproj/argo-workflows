package sync

import (
	"testing"

	apiv1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/stretchr/testify/assert"
	//	"github.com/stretchr/testify/require"
)

func TestNoDuplicates(t *testing.T) {
	tests := []struct {
		name  string
		items []*syncItem
	}{
		{
			name: "single",
			items: []*syncItem{
				{
					mutex: &v1alpha1.Mutex{
						Name: "alpha",
					},
				},
			},
		},
		{
			name: "two",
			items: []*syncItem{
				{
					mutex: &v1alpha1.Mutex{
						Name: "alpha",
					},
				},
				{
					mutex: &v1alpha1.Mutex{
						Name: "beta",
					},
				},
			},
		},
		{
			name: "different namespace mutex",
			items: []*syncItem{
				{
					mutex: &v1alpha1.Mutex{
						Name:      "alpha",
						Namespace: "foo",
					},
				},
				{
					mutex: &v1alpha1.Mutex{
						Name:      "alpha",
						Namespace: "bar",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkDuplicates(test.items)
			assert.NoError(t, err)
		})
	}
}

func TestExpectDuplicates(t *testing.T) {
	tests := []struct {
		name  string
		items []*syncItem
	}{
		{
			name: "simple duplicate mutex",
			items: []*syncItem{
				{
					mutex: &v1alpha1.Mutex{
						Name: "alpha",
					},
				},
				{
					mutex: &v1alpha1.Mutex{
						Name: "alpha",
					},
				},
			},
		},
		{
			name: "simple duplicate semaphore",
			items: []*syncItem{
				{
					semaphore: &v1alpha1.SemaphoreRef{
						ConfigMapKeyRef: &apiv1.ConfigMapKeySelector{
							LocalObjectReference: apiv1.LocalObjectReference{
								Name: "foo",
							},
							Key: "alpha",
						},
					},
				},
				{
					semaphore: &v1alpha1.SemaphoreRef{
						ConfigMapKeyRef: &apiv1.ConfigMapKeySelector{
							LocalObjectReference: apiv1.LocalObjectReference{
								Name: "foo",
							},
							Key: "alpha",
						},
					},
				},
			},
		},
		{
			name: "another duplicate mutex",
			items: []*syncItem{
				{
					mutex: &v1alpha1.Mutex{
						Name: "alpha",
					},
				},
				{
					mutex: &v1alpha1.Mutex{
						Name: "beta",
					},
				},
				{
					mutex: &v1alpha1.Mutex{
						Name: "alpha",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkDuplicates(test.items)
			assert.Error(t, err)
		})
	}
}
