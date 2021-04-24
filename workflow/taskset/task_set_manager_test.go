package taskset

import (
	"context"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	wfextv "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestCreateTaskSet(t *testing.T) {
	wfclientset := fakewfclientset.NewSimpleClientset()
	informerFactory := wfextv.NewSharedInformerFactory(wfclientset, 0)
	taskSetInformer := informerFactory.Argoproj().V1alpha1().WorkflowTaskSets()
	queueWorkflowFunc := func ( key string){
	}
	metrics := metrics.New(metrics.ServerConfig{}, metrics.ServerConfig{})
	taskSetMgr := NewWorkflowTaskSetManager(wfclientset.ArgoprojV1alpha1(), taskSetInformer, queueWorkflowFunc, metrics)

	wf := v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name: "test",
			Namespace: "default",
		},
		Spec:       v1alpha1.WorkflowSpec{},
	}
	nodeID := "test-xrgzj"
	tmpl := v1alpha1.Template{
		Name:                         "HTTP",
		HTTP:                         &v1alpha1.HTTP{
			URL:    "http://test.com",
		},
	}
	err := taskSetMgr.CreateTaskSet(context.Background(), &wf, nodeID, tmpl)
	assert.NoError(t, err)
	taskSet, err := wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").Get(context.Background(), "test", v1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, taskSet)
	assert.Len(t, taskSet.Spec.Tasks,1)
	assert.Equal(t, tmpl, taskSet.Spec.Tasks[0].Template)
}
