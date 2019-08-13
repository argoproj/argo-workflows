package util

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"

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

var workflowManifestWithoutManifest = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-
spec:
  entrypoint: whalesay
  templates:
    - name: whalesay
      resource:
        manifestPath: <PathToResourceManifest>
`

var workflowManifestWithManifest = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-
spec:
  entrypoint: whalesay
  templates:
    - name: whalesay
      resource:
        manifest: |- 
            apiVersion: batch/v1
            kind: Job
            metadata:
                generateName: different-whalesay-
            spec:
                template:
                    metadata:
                        name: different-whalesay
                    spec:
                     containers:
                        - name: different-whalesay
                          image: docker/whalesay:latest
`

var k8sResourceManifest = `
apiVersion: batch/v1
kind: Job
metadata:
  generateName: whalesay-
spec:
  template:
  metadata:
   name: whalesay
  spec:
    containers:
      - name: whalesay
        image: docker/whalesay:latest
`

// TestWritingManifestUsingManifestPath ensures that manifest is filled with the content of the file at manifestPath
func TestWritingManifestUsingManifestPath(t *testing.T) {
	dir, err := ioutil.TempDir("", "testWritingManifestUsingManifestPath")
	if err != nil {
		t.Error("Could not create temporary directory")
	}

	defer os.RemoveAll(dir)

	tmpfn := filepath.Join(dir, "test-resource.yaml")
	if err := ioutil.WriteFile(tmpfn, []byte(k8sResourceManifest), 0666); err != nil {
		t.Error("Could not write to test resource file")
	}

	workflowManifestWithoutManifest = strings.Replace(workflowManifestWithoutManifest, "<PathToResourceManifest>", tmpfn, 1)

	wf := unmarshalWF(workflowManifestWithoutManifest)

	assert.True(t, wf.Spec.Templates[0].Resource.Manifest == "")

	err = MaybeWriteManifestFromManifestPath(wf)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, wf.Spec.Templates[0].Resource.Manifest, k8sResourceManifest)
}

// TestNotOverwritingManifestWithManifestPathWhenNotEmpty ensures that manifest's content is not overwritten
// with the content of the file at manifestPath when it is not empty
func TestNotOverwritingManifestWithManifestPathWhenNotEmpty(t *testing.T) {
	dir, err := ioutil.TempDir("", "testNotOverwritingManifestWithManifestPathWhenNotEmpty")
	if err != nil {
		t.Error("Could not create temporary directory")
	}

	defer os.RemoveAll(dir)

	tmpfn := filepath.Join(dir, "test-resource.yaml")
	if err := ioutil.WriteFile(tmpfn, []byte(k8sResourceManifest), 0666); err != nil {
		t.Error("Could not write to test resource file")
	}

	workflowManifestWithManifest = strings.Replace(workflowManifestWithManifest, "<PathToResourceManifest>", tmpfn, 1)

	wf := unmarshalWF(workflowManifestWithManifest)

	assert.True(t, wf.Spec.Templates[0].Resource.Manifest != "")

	err = MaybeWriteManifestFromManifestPath(wf)
	if err != nil {
		t.Error(err)
	}

	assert.NotEqual(t, wf.Spec.Templates[0].Resource.Manifest, k8sResourceManifest)
}

func unmarshalWF(yamlStr string) *wfv1.Workflow {
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(yamlStr), &wf)
	if err != nil {
		panic(err)
	}
	return &wf
}
