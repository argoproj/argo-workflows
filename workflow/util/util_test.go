package util

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestResubmitWorkflowWithOnExit ensures we do not carry over the onExit node even if successful
func TestResubmitWorkflowWithOnExit(t *testing.T) {
	wfName := "test-wf"
	onExitName := wfName + ".onExit"
	wf := wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-wf",
		},
		Status: wfv1.WorkflowStatus{
			Phase: wfv1.NodeFailed,
			Nodes: map[string]wfv1.NodeStatus{},
		},
	}
	onExitID := wf.NodeID(onExitName)
	wf.Status.Nodes[onExitID] = wfv1.NodeStatus{
		Name:  onExitName,
		Phase: wfv1.NodeSucceeded,
	}
	newWF, err := FormulateResubmitWorkflow(&wf, true)
	assert.Nil(t, err)
	newWFOnExitName := newWF.ObjectMeta.Name + ".onExit"
	newWFOneExitID := newWF.NodeID(newWFOnExitName)
	_, ok := newWF.Status.Nodes[newWFOneExitID]
	assert.False(t, ok)
}

// TestReadFromPath ensures we can read the content of a file correctly using the ReadFromUrlOrPath function
func TestReadFromPath(t *testing.T) {
	content := []byte("test file's content")
	dir, err := ioutil.TempDir("", "testReadFromUrlOrPath")
	if err != nil {
		t.Error("Could not create temporary directory")
	}

	defer os.RemoveAll(dir)

	tmpfn := filepath.Join(dir, "test_file.yaml")
	if err := ioutil.WriteFile(tmpfn, content, 0666); err != nil {
		t.Error("Could not write to temporary file")
	}
	body, err := ReadFromUrlOrPath(tmpfn)
	assert.Nil(t, err)
	assert.Equal(t, body, content)
}
