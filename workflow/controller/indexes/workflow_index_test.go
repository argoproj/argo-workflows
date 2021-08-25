package indexes

import (
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestWorkflowIndexFunc(t *testing.T) {
	// TODO - both cases neede
	obj := &unstructured.Unstructured{}
	wfv1.MustUnmarshal(`
apiVersion: v1
kind: Pod
metadata:
  namespace: my-ns
  labels:
    multi-cluster.argoproj.io/owner-cluster-name: cn
    multi-cluster.argoproj.io/owner-namespace: ns
    multi-cluster.argoproj.io/owner-name: n
`, obj)
	v, err := MetaWorkflowIndexFunc(obj)
	if assert.NoError(t, err) {
		assert.Equal(t, []string{"cn/ns/n"}, v)
	}
}

func TestWorkflowIndexValue(t *testing.T) {
	assert.Equal(t, "my-ns/my-wf", WorkflowIndexValue("my-ns", "my-wf"))
}
