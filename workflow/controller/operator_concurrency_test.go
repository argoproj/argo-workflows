package controller

import (
	"strconv"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	argoErr "github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/sync"
)

const configMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  workflow: "2"
  template: "1"
`
const wfWithSemaphore = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata: 
  name: hello-world
  namespace: default
spec: 
  entrypoint: whalesay
  templates: 
    - 
      synchronization: 
        semaphore: 
          configMapKeyRef: 
            key: template
            name: my-config
      container: 
        args: 
          - "hello world"
        command: 
          - cowsay
        image: "docker/whalesay:latest"
      name: whalesay
`

func GetSyncLimitFunc(kube kubernetes.Interface) func(string) (int, error) {
	syncLimitConfig := func(lockName string) (int, error) {
		items := strings.Split(lockName, "/")
		if len(items) < 4 {
			return 0, argoErr.New(argoErr.CodeBadRequest, "Invalid Config Map Key")
		}

		configMap, err := kube.CoreV1().ConfigMaps(items[0]).Get(items[2], metav1.GetOptions{})

		if err != nil {
			return 0, err
		}

		value, found := configMap.Data[items[3]]

		if !found {
			return 0, argoErr.New(argoErr.CodeBadRequest, "Invalid Sync configuration Key")
		}
		return strconv.Atoi(value)
	}
	return syncLimitConfig
}

func TestGetNodeType(t *testing.T) {
	t.Run("getNodeType", func(t *testing.T) {
		assert.Equal(t, wfv1.NodeTypePod, getNodeType(wfv1.TemplateTypeScript))
		assert.Equal(t, wfv1.NodeTypePod, getNodeType(wfv1.TemplateTypeContainer))
		assert.Equal(t, wfv1.NodeTypePod, getNodeType(wfv1.TemplateTypeResource))
		assert.NotEqual(t, wfv1.NodeTypePod, getNodeType(wfv1.TemplateTypeSteps))
		assert.NotEqual(t, wfv1.NodeTypePod, getNodeType(wfv1.TemplateTypeDAG))
		assert.NotEqual(t, wfv1.NodeTypePod, getNodeType(wfv1.TemplateTypeSuspend))
	})
}

func TestSemaphoreTmplLevel(t *testing.T) {
	_, controller := newController()
	controller.concurrencyMgr = sync.NewLockManager(GetSyncLimitFunc(controller.kubeclientset), func(key string) {
	})
	var cm v1.ConfigMap
	err := yaml.Unmarshal([]byte(configMap), &cm)
	assert.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(&cm)
	assert.NoError(t, err)
	t.Run("TmplLevelAcquireAndRelease", func(t *testing.T) {
		wf := unmarshalWF(wfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf)
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		// acquired the lock
		woc.operate()
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc.wf.Status.Synchronization.Semaphore.Holding))

		// Try to Acquire the lock, But lock is not available
		wf_Two := wf.DeepCopy()
		wf_Two.Name = "two"
		wf_Two, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf_Two)
		assert.NoError(t, err)
		woc_two := newWorkflowOperationCtx(wf_Two, controller)
		// Try Acquire the lock
		woc_two.operate()
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Check Node status
		err = woc_two.podReconciliation()
		assert.NoError(t, err)
		for _, node := range woc_two.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}

		// Release the lock
		woc.operate()
		assert.NotNil(t, woc.wf.Status.Synchronization)
		assert.NotNil(t, woc.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 0, len(woc.wf.Status.Synchronization.Semaphore.Holding[0].Holders))

		// Try to acquired the lock
		woc_two.operate()
		assert.NotNil(t, woc_two.wf.Status.Synchronization)
		assert.NotNil(t, woc_two.wf.Status.Synchronization.Semaphore)
		assert.Equal(t, 1, len(woc_two.wf.Status.Synchronization.Semaphore.Holding))

	})
}

func TestSemaphoreWithOutConfigMap(t *testing.T) {
	_, controller := newController()

	controller.concurrencyMgr = sync.NewLockManager(GetSyncLimitFunc(controller.kubeclientset), func(key string) {
	})

	t.Run("SemaphoreRefWithOutConfigMap", func(t *testing.T) {
		wf := unmarshalWF(wfWithSemaphore)
		wf.Name = "one"
		wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(wf)
		assert.NoError(t, err)
		woc := newWorkflowOperationCtx(wf, controller)
		err = woc.podReconciliation()
		assert.NoError(t, err)
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Acquire the lock
		woc.operate()
		assert.Nil(t, woc.wf.Status.Synchronization)
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodeError, node.Phase)
		}

	})
}
