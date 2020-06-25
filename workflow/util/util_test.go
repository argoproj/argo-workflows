package util

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fakeClientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	hydratorfake "github.com/argoproj/argo/workflow/hydrator/fake"
)

// TestSubmitDryRun
func TestSubmitDryRun(t *testing.T) {

	workflowName := "test-dry-run"
	workflowYaml := `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
    name: ` + workflowName + `
spec:
    entrypoint: whalesay
    templates:
    - name: whalesay
      container: 
        image: docker/whalesay:latest
        command: [cowsay]
        args: ["hello world"]
`
	wf := unmarshalWF(workflowYaml)
	newWf := wf.DeepCopy()
	wfClientSet := fakeClientset.NewSimpleClientset()
	newWf, err := SubmitWorkflow(nil, wfClientSet, "test-namespace", newWf, &wfv1.SubmitOpts{DryRun: true})
	assert.NoError(t, err)
	assert.Equal(t, wf.Spec, newWf.Spec)
	assert.Equal(t, wf.Status, newWf.Status)
}

// TestResubmitWorkflowWithOnExit ensures we do not carry over the onExit node even if successful
func TestResubmitWorkflowWithOnExit(t *testing.T) {
	wfName := "test-wf"
	onExitName := wfName + ".onExit"
	wf := wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: wfName,
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
	assert.NoError(t, err)
	newWFOnExitName := newWF.ObjectMeta.Name + ".onExit"
	newWFOneExitID := newWF.NodeID(newWFOnExitName)
	_, ok := newWF.Status.Nodes[newWFOneExitID]
	assert.False(t, ok)
}

// TestReadFromSingleorMultiplePath ensures we can read the content of a single file or multiple files correctly using the ReadFromFilePathsOrUrls function
func TestReadFromSingleorMultiplePath(t *testing.T) {
	tests := map[string]struct {
		fileNames []string
		contents  []string
	}{
		"singleFile": {
			fileNames: []string{"singleFile"},
			contents:  []string{"test file's content"},
		},
		"multipleFiles": {
			fileNames: []string{"file1", "file2"},
			contents:  []string{"file1 content", "file2 content"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", name)
			if err != nil {
				t.Error("Could not create temporary directory")
			}
			defer os.RemoveAll(dir)
			var filePaths []string
			for i := range tc.fileNames {
				content := []byte(tc.contents[i])
				tmpfn := filepath.Join(dir, tc.fileNames[i])
				filePaths = append(filePaths, tmpfn)
				err := ioutil.WriteFile(tmpfn, content, 0666)
				if err != nil {
					t.Error("Could not write to temporary file")
				}
			}
			body, err := ReadFromFilePathsOrUrls(filePaths...)
			assert.Equal(t, len(body), len(filePaths))
			assert.NoError(t, err)
			for i := range body {
				assert.Equal(t, body[i], []byte(tc.contents[i]))
			}
		})
	}
}

// TestReadFromSingleorMultiplePathErrorHandling ensures that an error is returned if there is any error while reading files or urls
func TestReadFromSingleorMultiplePathErrorHandling(t *testing.T) {
	tests := map[string]struct {
		fileNames []string
		contents  []string
		exists    []bool
	}{
		"nonExistingFile": {
			fileNames: []string{"nonExistingFile"},
			contents:  []string{"this content should not exist"},
			exists:    []bool{false},
		},
		"multipleNonExistingFiles": {
			fileNames: []string{"file1", "file2"},
			contents:  []string{"this content should not exist", "this content should not exist"},
			exists:    []bool{false, false},
		},
		"mixedExistingAndNonExistingFiles": {
			fileNames: []string{"file1", "file2", "file3", "file4"},
			contents:  []string{"actual file content", "", "", "actual file content 2"},
			exists:    []bool{true, false, false, true},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", name)
			if err != nil {
				t.Error("Could not create temporary directory")
			}
			defer os.RemoveAll(dir)
			var filePaths []string
			for i := range tc.fileNames {
				content := []byte(tc.contents[i])
				tmpfn := filepath.Join(dir, tc.fileNames[i])
				filePaths = append(filePaths, tmpfn)
				if tc.exists[i] {
					err := ioutil.WriteFile(tmpfn, content, 0666)
					if err != nil {
						t.Error("Could not write to temporary file")
					}
				}
			}
			body, err := ReadFromFilePathsOrUrls(filePaths...)
			assert.NotNil(t, err)
			assert.Equal(t, len(body), 0)
		})
	}
}

func unmarshalWF(yamlStr string) *wfv1.Workflow {
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(yamlStr), &wf)
	if err != nil {
		panic(err)
	}
	return &wf
}

var yamlStr = `
containers:
  - name: main
    resources:
      limits:
        cpu: 1000m
`

func TestPodSpecPatchMerge(t *testing.T) {
	tmpl := wfv1.Template{PodSpecPatch: "{\"containers\":[{\"name\":\"main\", \"resources\":{\"limits\":{\"cpu\": \"1000m\"}}}]}"}
	wf := wfv1.Workflow{Spec: wfv1.WorkflowSpec{PodSpecPatch: "{\"containers\":[{\"name\":\"main\", \"resources\":{\"limits\":{\"memory\": \"100Mi\"}}}]}"}}
	merged, err := PodSpecPatchMerge(&wf, &tmpl)
	assert.NoError(t, err)
	var spec v1.PodSpec
	err = json.Unmarshal([]byte(merged), &spec)
	assert.NoError(t, err)
	assert.Equal(t, "1.000", spec.Containers[0].Resources.Limits.Cpu().AsDec().String())
	assert.Equal(t, "104857600", spec.Containers[0].Resources.Limits.Memory().AsDec().String())

	tmpl = wfv1.Template{PodSpecPatch: yamlStr}
	wf = wfv1.Workflow{Spec: wfv1.WorkflowSpec{PodSpecPatch: "{\"containers\":[{\"name\":\"main\", \"resources\":{\"limits\":{\"memory\": \"100Mi\"}}}]}"}}
	merged, err = PodSpecPatchMerge(&wf, &tmpl)
	assert.NoError(t, err)
	err = json.Unmarshal([]byte(merged), &spec)
	assert.NoError(t, err)
	assert.Equal(t, "1.000", spec.Containers[0].Resources.Limits.Cpu().AsDec().String())
	assert.Equal(t, "104857600", spec.Containers[0].Resources.Limits.Memory().AsDec().String())
}

var suspendedWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-04-10T15:21:23Z"
  name: suspend
  generation: 2
  labels:
    workflows.argoproj.io/phase: Running
  resourceVersion: "238969"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/workflows/suspend
  uid: 4f08d325-dc5a-43a3-9986-259e259e6ea3
spec:
  arguments: {}
  entrypoint: suspend
  templates:
  - arguments: {}
    inputs: {}
    metadata: {}
    name: suspend
    outputs: {}
    steps:
    - - arguments: {}
        name: approve
        template: approve
  - arguments: {}
    inputs: {}
    metadata: {}
    name: approve
    outputs: {}
    suspend: {}
status:
  finishedAt: null
  nodes:
    suspend-template-xjsg2:
      children:
      - suspend-template-xjsg2-4125372399
      displayName: suspend-template-xjsg2
      finishedAt: null
      id: suspend-template-xjsg2
      name: suspend-template-xjsg2
      phase: Running
      startedAt: "2020-04-10T15:21:23Z"
      templateName: suspend
      templateScope: local/suspend-template-xjsg2
      type: Steps
    suspend-template-xjsg2-1771269240:
      boundaryID: suspend-template-xjsg2
      displayName: approve
      finishedAt: null
      id: suspend-template-xjsg2-1771269240
      name: suspend-template-xjsg2[0].approve
      phase: Running
      startedAt: "2020-04-10T15:21:23Z"
      templateName: approve
      templateScope: local/suspend-template-xjsg2
      type: Suspend
    suspend-template-xjsg2-4125372399:
      boundaryID: suspend-template-xjsg2
      children:
      - suspend-template-xjsg2-1771269240
      displayName: '[0]'
      finishedAt: null
      id: suspend-template-xjsg2-4125372399
      name: suspend-template-xjsg2[0]
      phase: Running
      startedAt: "2020-04-10T15:21:23Z"
      templateName: suspend
      templateScope: local/suspend-template-xjsg2
      type: StepGroup
  phase: Running
  startedAt: "2020-04-10T15:21:23Z"
`

func TestResumeWorkflowByNodeName(t *testing.T) {
	wfIf := fakeClientset.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
	origWf := unmarshalWF(suspendedWf)

	_, err := wfIf.Create(origWf)
	assert.NoError(t, err)

	//will return error as displayName does not match any nodes
	err = ResumeWorkflow(wfIf, hydratorfake.Noop, "suspend", "displayName=nonexistant")
	assert.Error(t, err)

	//displayName didn't match suspend node so should still be running
	wf, err := wfIf.Get("suspend", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByDisplayName("approve").Phase)

	err = ResumeWorkflow(wfIf, hydratorfake.Noop, "suspend", "displayName=approve")
	assert.NoError(t, err)

	//displayName matched node so has succeeded
	wf, err = wfIf.Get("suspend", metav1.GetOptions{})
	if assert.NoError(t, err) {
		assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByDisplayName("approve").Phase)
	}
}

func TestStopWorkflowByNodeName(t *testing.T) {
	wfIf := fakeClientset.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
	origWf := unmarshalWF(suspendedWf)

	_, err := wfIf.Create(origWf)
	assert.NoError(t, err)

	//will return error as displayName does not match any nodes
	err = StopWorkflow(wfIf, hydratorfake.Noop, "suspend", "displayName=nonexistant", "error occurred")
	assert.Error(t, err)

	//displayName didn't match suspend node so should still be running
	wf, err := wfIf.Get("suspend", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByDisplayName("approve").Phase)

	err = StopWorkflow(wfIf, hydratorfake.Noop, "suspend", "displayName=approve", "error occurred")
	assert.NoError(t, err)

	//displayName matched node so has succeeded
	wf, err = wfIf.Get("suspend", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, wf.Status.Nodes.FindByDisplayName("approve").Phase)
}
