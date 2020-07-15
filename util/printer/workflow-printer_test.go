package printer

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestPrintWorkflows(t *testing.T) {
	now := time.Now()
	intOrString := intstr.Parse("my-value")
	workflows := wfv1.Workflows{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wf", Namespace: "my-ns", CreationTimestamp: metav1.Time{Time: now}},
			Spec: wfv1.WorkflowSpec{
				Arguments: wfv1.Arguments{Parameters: []wfv1.Parameter{
					{Name: "my-param", Value: &intOrString},
				}},
				Priority: pointer.Int32Ptr(2),
				Templates: []wfv1.Template{
					{Name: "t0", Container: &corev1.Container{}},
				},
			},
			Status: wfv1.WorkflowStatus{
				Phase:      wfv1.NodeRunning,
				StartedAt:  metav1.Time{Time: now},
				FinishedAt: metav1.Time{Time: now.Add(3 * time.Second)},
				Nodes: wfv1.Nodes{
					"n0": {Phase: wfv1.NodePending, Type: wfv1.NodeTypePod, TemplateName: "t0"},
					"n1": {Phase: wfv1.NodeRunning, Type: wfv1.NodeTypePod, TemplateName: "t0"},
					"n2": {Phase: wfv1.NodeRunning, Type: wfv1.NodeTypePod, TemplateName: "t0"},
					"n3": {Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, TemplateName: "t0"},
					"n4": {Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod, TemplateName: "t0"},
					"n5": {Phase: wfv1.NodeError, Type: wfv1.NodeTypePod, TemplateName: "t0"},
				},
			},
		},
	}
	t.Run("Default", func(t *testing.T) {
		var b bytes.Buffer
		assert.NoError(t, PrintWorkflows(workflows, &b, PrintOpts{}))
		assert.Equal(t, `NAME    STATUS    AGE   DURATION   PRIORITY
my-wf   Running   0s    3s         2
`, b.String())
	})
	t.Run("NoHeader", func(t *testing.T) {
		var b bytes.Buffer
		assert.NoError(t, PrintWorkflows(workflows, &b, PrintOpts{NoHeaders: true}))
		assert.Equal(t, `my-wf   Running   0s   3s   2
`, b.String())
	})
	t.Run("Namespace", func(t *testing.T) {
		var b bytes.Buffer
		assert.NoError(t, PrintWorkflows(workflows, &b, PrintOpts{Namespace: true}))
		assert.Equal(t, `NAMESPACE   NAME    STATUS    AGE   DURATION   PRIORITY
my-ns       my-wf   Running   0s    3s         2
`, b.String())
	})
	t.Run("Wide", func(t *testing.T) {
		var b bytes.Buffer
		assert.NoError(t, PrintWorkflows(workflows, &b, PrintOpts{Output: "wide"}))
		assert.Equal(t, `NAME    STATUS    AGE   DURATION   PRIORITY   P/R/C   PARAMETERS
my-wf   Running   0s    3s         2          1/2/3   my-param=my-value
`, b.String())
	})
	t.Run("Name", func(t *testing.T) {
		var b bytes.Buffer
		assert.NoError(t, PrintWorkflows(workflows, &b, PrintOpts{Output: "name"}))
		assert.Equal(t, `my-wf
`, b.String())
	})
	t.Run("JSON", func(t *testing.T) {
		var b bytes.Buffer
		assert.NoError(t, PrintWorkflows(workflows, &b, PrintOpts{Output: "json"}))
		assert.NotEmpty(t, b.String())
	})
	t.Run("YAML", func(t *testing.T) {
		var b bytes.Buffer
		assert.NoError(t, PrintWorkflows(workflows, &b, PrintOpts{Output: "yaml"}))
		assert.NotEmpty(t, b.String())
	})
}
