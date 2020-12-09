// +build e2emc

package e2e

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/test/e2e/fixtures"
)

type AgentSuite struct {
	fixtures.E2ESuite
}

func (s *AgentSuite) TestAgent() {
	t := s.T()
	config, err := clientcmd.BuildConfigFromFlags("", "../../cmd/agent/testdata/kubeconfig")
	assert.NoError(t, err)

	clientset, err := kubernetes.NewForConfig(config)
	assert.NoError(t, err)

	pods := clientset.CoreV1().Pods("argo")

	testUID := "test." + strconv.FormatInt(time.Now().Unix(), 10)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-pod",
			Labels:       map[string]string{"testUID": testUID},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "main", Image: "argoproj/argosay:v2"},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	var name string

	listOptions := metav1.ListOptions{LabelSelector: "testUID=" + testUID}

	t.Run("Create", func(t *testing.T) {
		pod, err := pods.Create(pod)
		if assert.NoError(t, err) {
			assert.NotNil(t, pod)
		}
	})
	t.Run("List", func(t *testing.T) {
		podList, err := pods.List(listOptions)
		if assert.NoError(t, err) {
			assert.Len(t, podList.Items, 1)
			name = podList.Items[0].Name
			assert.NotEmpty(t, name)
		}
	})
	t.Run("Get", func(t *testing.T) {
		pod, err = pods.Get(name, metav1.GetOptions{})
		if assert.NoError(t, err) && assert.NotNil(t, pod) {
			assert.Equal(t, name, pod.Name)
		}
	})
	t.Run("Update", func(t *testing.T) {
		pod, err := pods.Update(pod)
		if assert.NoError(t, err) && assert.NotNil(t, pod) {
			assert.Equal(t, name, pod.Name)
		}
	})
	t.Run("Patch", func(t *testing.T) {
		pod, err := pods.Patch(pod.Name, types.MergePatchType, []byte(`{"metadata": {"annotations": {"patched": "true"}}}`))
		if assert.NoError(t, err) && assert.NotNil(t, pod) {
			assert.NotEmpty(t, pod.Annotations["patched"])
		}
	})
	t.Run("Watch", func(t *testing.T) {
		w, err := pods.Watch(listOptions)
		if assert.NoError(t, err) && assert.NotNil(t, w) {
			defer w.Stop()
		loop:
			for event := range w.ResultChan() {
				switch event.Type {
				case watch.Modified:
					break loop
				default:
					if !assert.NotEqual(t, watch.Error, event.Type) {
						break loop
					}
				}
			}
			println("done")
		}
	})
	t.Run("Delete", func(t *testing.T) {
		err := pods.Delete(name, &metav1.DeleteOptions{})
		assert.NoError(t, err)
	})
	t.Run("DeleteCollection", func(t *testing.T) {
		err := pods.DeleteCollection(&metav1.DeleteOptions{}, listOptions)
		assert.NoError(t, err)
	})
}

func TestAgentSuite(t *testing.T) {
	suite.Run(t, new(AgentSuite))
}
