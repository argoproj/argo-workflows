package indexes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	testutil "github.com/argoproj/argo/test/util"
)

func TestWorkflowIndexFunc(t *testing.T) {
	obj := &unstructured.Unstructured{}
	testutil.MustUnmarshallYAML(`
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

func TestWorkflowIndexValue(t *testing.T) {
	assert.Equal(t, "my-ns/my-wf", WorkflowIndexValue("my-ns", "my-wf"))
}
