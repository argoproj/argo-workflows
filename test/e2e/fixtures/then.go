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
	given *Given
}

func (t *Then) Expect(block func(t *testing.T, wf *wfv1.WorkflowStatus)) *Then {
	fmt.Printf("checking expectation %s", t.wf().Name)

	wf, err := t.client().Get(t.wf().Name, metav1.GetOptions{})
	if err != nil {
		t.t().Fatal(err)
	}
	bytes, err := yaml.Marshal(wf.Status)
	fmt.Println(string(bytes))
	block(t.t(), &wf.Status)
	return t
}

func (t *Then) client() v1alpha1.WorkflowInterface {
	return t.given.client()
}

func (t *Then) wf() *wfv1.Workflow {
	return t.given.wf
}

func (t *Then) t() *testing.T {
	return t.given.t()
}
