package template

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEval(t *testing.T) {
	y, err := Eval(&wfv1.WorkflowSpec{Entrypoint: `ƛx == "foo" ? "bar": "x"`}, map[string]string{"x": "foo"})
	assert.NoError(t, err)
	assert.Equal(t, &wfv1.WorkflowSpec{Entrypoint: "bar"}, y)
}
