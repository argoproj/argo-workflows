package validate

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// A Workflow created from a WorkflowTemplate reference must resolve DAG task names from the
// referenced template's templates during submit validation: a Workflow created from a templateRef
// has no inline templates of its own, so it must not inherit an empty template index.
func TestWFTRefSubmitDAGTaskResolution(t *testing.T) {
	wft := &wfv1.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "etl-demo", Namespace: metav1.NamespaceDefault},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "main",
			Templates: []wfv1.Template{
				{Name: "main", DAG: &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{
					{Name: "fetch", Template: "fetch"},
					{Name: "process", Template: "process", Depends: "fetch"},
				}}},
				{Name: "fetch", Container: &corev1.Container{Image: "busybox"}},
				{Name: "process", Container: &corev1.Container{Image: "busybox"}},
			},
		},
	}
	_, err := wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault).Create(logging.TestContext(t.Context()), wft, metav1.CreateOptions{})
	require.NoError(t, err)

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "etl-demo-run-", Namespace: metav1.NamespaceDefault},
		Spec: wfv1.WorkflowSpec{
			WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "etl-demo"},
		},
	}
	err = Workflow(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, wf, nil, Opts{Submit: true})
	require.NoError(t, err)
}
