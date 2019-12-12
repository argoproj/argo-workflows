package fixtures

import (
	"bufio"
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

type Then struct {
	t      *testing.T
	name   string
	client v1alpha1.WorkflowInterface
	kubeClient kubernetes.Interface
}

func (t *Then) Expect(block func(t *testing.T, wf *wfv1.WorkflowStatus)) *Then {
	fmt.Printf("checking expectation %s", t.name)
	wf, err := t.client.Get(t.name, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	bytes, err := yaml.Marshal(wf.Status)
	if err != nil {
		t.t.Fatal(err)
	}
	fmt.Println(string(bytes))
	block(t.t, &wf.Status)
	return t
}

func (t *Then) Logs() {
	wf, err := t.client.Get(t.name, metav1.GetOptions{})
	if err != nil {
		t.t.Fatal(err)
	}
	for _, status := range wf.Status.Nodes {
		pods := t.kubeClient.CoreV1().Pods(wf.Namespace)
		podName := status.ID
		pod, err := pods.Get(podName, metav1.GetOptions{})
		if err != nil {
			t.t.Fatal(err)
		}
		for _, container := range pod.Status.ContainerStatuses {
			fmt.Printf("=== %s/%s/%v", podName, container.Name, container.State)
			stream, err := pods.GetLogs(podName, &v1.PodLogOptions{Container: container.Name,}).Stream()
			if err != nil {
				t.t.Fatal(err)
			}
			scanner := bufio.NewScanner(stream)
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Println(line)
			}
		}
	}
}
