package indexes

import (
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestWorkflowIndexFunc(t *testing.T) {
	obj := &unstructured.Unstructured{}
	wfv1.MustUnmarshal(`
apiVersion: v1
kind: Pod
metadata:
  namespace: my-ns
  labels:
    workflows.argoproj.io/workflow: my-wf
`, obj)
	v, err := MetaWorkflowIndexFunc(obj)
	if assert.NoError(t, err) {
		assert.Equal(t, []string{"my-ns/my-wf"}, v)
	}
}

func TestMetaNodeIDIndexFunc(t *testing.T) {
	withNodeID := `
apiVersion: v1
kind: Pod
metadata:
  namespace: my-ns
  name: retry-test-p7jzr-whalesay-2308805457
  labels:
    workflows.argoproj.io/workflow: my-wf
  annotations:
    workflows.argoproj.io/node-id: retry-test-p7jzr-2308805457
    workflows.argoproj.io/node-name: 'retry-test-p7jzr[0].steps-outer-step1'
`
	withoutNodeID := `
apiVersion: v1
kind: Pod
metadata:
  namespace: my-ns
  name: retry-test-p7jzr-whalesay-2308805457
  labels:
    workflows.argoproj.io/workflow: my-wf
  annotations:
    workflows.argoproj.io/node-name: 'retry-test-p7jzr[0].steps-outer-step1'
`
	obj := &unstructured.Unstructured{}
	wfv1.MustUnmarshal(withNodeID, obj)
	v, err := MetaNodeIDIndexFunc(obj)
	assert.NoError(t, err)
	assert.Equal(t, []string{"my-ns/retry-test-p7jzr-2308805457"}, v)

	obj = &unstructured.Unstructured{}
	wfv1.MustUnmarshal(withoutNodeID, obj)
	v, err = MetaNodeIDIndexFunc(obj)
	assert.NoError(t, err)
	assert.Equal(t, []string{"my-ns/retry-test-p7jzr-whalesay-2308805457"}, v)
}

func TestWorkflowIndexValue(t *testing.T) {
	assert.Equal(t, "my-ns/my-wf", WorkflowIndexValue("my-ns", "my-wf"))
}
