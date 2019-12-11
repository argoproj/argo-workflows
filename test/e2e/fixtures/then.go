package fixtures

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

type Then struct {
	t      *testing.T
	name   string
	client v1alpha1.WorkflowInterface
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
