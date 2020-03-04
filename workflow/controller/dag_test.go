package controller

import (
	"fmt"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test"
)

// TestDagXfail verifies a DAG can fail properly
func TestDagXfail(t *testing.T) {
	wf := test.LoadTestWorkflow("testdata/dag_xfail.yaml")
	woc := newWoc(*wf)
	woc.operate()
	assert.Equal(t, string(wfv1.NodeFailed), string(woc.wf.Status.Phase))
}

// TestDagRetrySucceeded verifies a DAG will be marked Succeeded if retry was successful
func TestDagRetrySucceeded(t *testing.T) {
	wf := test.LoadTestWorkflow("testdata/dag_retry_succeeded.yaml")
	woc := newWoc(*wf)
	woc.operate()
	assert.Equal(t, string(wfv1.NodeSucceeded), string(woc.wf.Status.Phase))
}

// TestDagRetryExhaustedXfail verifies we fail properly when we exhaust our retries
func TestDagRetryExhaustedXfail(t *testing.T) {
	wf := test.LoadTestWorkflow("testdata/dag-exhausted-retries-xfail.yaml")
	woc := newWoc(*wf)
	woc.operate()
	assert.Equal(t, string(wfv1.NodeFailed), string(woc.wf.Status.Phase))
}

// TestDagDisableFailFast test disable fail fast function
func TestDagDisableFailFast(t *testing.T) {
	wf := test.LoadTestWorkflow("testdata/dag-disable-fail-fast.yaml")
	woc := newWoc(*wf)
	woc.operate()
	assert.Equal(t, string(wfv1.NodeFailed), string(woc.wf.Status.Phase))
}

var dynamicSingleDag = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-diamond-
spec:
  entrypoint: diamond
  templates:
  - name: diamond
    dag:
      tasks:
      - name: A
        template: %s
        %s
      - name: TestSingle
        template: succeeded
        depends:
          %s: A

  - name: succeeded
    container:
      image: alpine:3.7
      command: [sh, -c, "exit 0"]

  - name: failed
    container:
      image: alpine:3.7
      command: [sh, -c, "exit 1"]

  - name: skipped
    when: "False"
    container:
      image: alpine:3.7
      command: [sh, -c, "echo Hello"]
`

func TestSingleDependency(t *testing.T) {
	statusMap := map[string]v1.PodPhase{"succeeded": v1.PodSucceeded, "failed": v1.PodFailed}
	for _, status := range []string{"succeeded", "failed", "skipped"} {
		fmt.Printf("\n\n\nCurrent status %s\n\n\n", status)
		controller := newController()
		wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

		// If the status is "skipped" skip the root node.
		var wfString string
		if status == "skipped" {
			wfString = fmt.Sprintf(dynamicSingleDag, status, `when: "False == True"`, status)
		} else {
			wfString = fmt.Sprintf(dynamicSingleDag, status, "", status)
		}
		wf := unmarshalWF(wfString)
		wf, err := wfcset.Create(wf)
		assert.Nil(t, err)
		wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
		assert.Nil(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		woc.operate()
		// Mark the status of the pod according to the test
		if _, ok := statusMap[status]; ok {
			makePodsPhase(t, statusMap[status], controller.kubeclientset, wf.ObjectMeta.Namespace)
		}

		woc.operate()
		found := false
		for _, node := range woc.wf.Status.Nodes {
			if strings.Contains(node.Name, "TestSingle") {
				found = true
				assert.Equal(t, wfv1.NodePending, node.Phase)
			}
		}
		assert.True(t, found)
	}
}
