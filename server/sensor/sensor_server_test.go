package sensor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
)

func Test_unToStruct(t *testing.T) {
	item, err := util.ToUnstructured(&wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
	})
	assert.NoError(t, err)
	s, err := unstructuredToStruct(*item)
	if assert.NoError(t, err) {
		assert.NotNil(t, s)
		assert.Equal(t, "my-wf", s.Fields["metadata"].GetStructValue().Fields["name"])
	}
}
