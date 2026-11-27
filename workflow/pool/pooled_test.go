package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestPooled(t *testing.T) {
	for _, tt := range []struct {
		name string
		tmpl wfv1.Template
		want bool
	}{
		{"container with pool", wfv1.Template{Pool: &wfv1.Pool{}, Container: &apiv1.Container{}}, true},
		{"script with pool", wfv1.Template{Pool: &wfv1.Pool{}, Script: &wfv1.ScriptTemplate{}}, true},
		{"container without pool", wfv1.Template{Container: &apiv1.Container{}}, false},
		{"steps with pool", wfv1.Template{Pool: &wfv1.Pool{}, Steps: []wfv1.ParallelSteps{}}, false},
		{"resource with pool", wfv1.Template{Pool: &wfv1.Pool{}, Resource: &wfv1.ResourceTemplate{}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Pooled(&tt.tmpl))
		})
	}
}
