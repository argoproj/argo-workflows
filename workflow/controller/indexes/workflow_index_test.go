package indexes

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowIndexFunc(t *testing.T) {

	obj := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"multi-cluster.argoproj.io/owner-cluster":   "cn",
				"multi-cluster.argoproj.io/owner-namespace": "ns",
				"multi-cluster.argoproj.io/owner-name":      "n",
			},
		},
	}
	v, err := MetaWorkflowIndexFunc(obj)
	if assert.NoError(t, err) {
		assert.Equal(t, []string{"ns/n"}, v)
	}
}

func TestWorkflowIndexValue(t *testing.T) {
	assert.Equal(t, "my-ns/my-wf", WorkflowIndexValue("my-ns", "my-wf"))
}
