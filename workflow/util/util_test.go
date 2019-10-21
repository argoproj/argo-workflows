package util

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fakeClientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	newWf, err := SubmitWorkflow(nil, wfClientSet, "test-namespace", newWf, &SubmitOpts{DryRun: true})
	assert.Nil(t, err)
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
	assert.Nil(t, err)
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
			assert.Nil(t, err)
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
	merged, _ := PodSpecPatchMerge(&wf, &tmpl)
	var spec v1.PodSpec
	json.Unmarshal([]byte(merged), &spec)
	assert.Equal(t, "1.000", spec.Containers[0].Resources.Limits.Cpu().AsDec().String())
	assert.Equal(t, "104857600", spec.Containers[0].Resources.Limits.Memory().AsDec().String())

	tmpl = wfv1.Template{PodSpecPatch: yamlStr}
	wf = wfv1.Workflow{Spec: wfv1.WorkflowSpec{PodSpecPatch: "{\"containers\":[{\"name\":\"main\", \"resources\":{\"limits\":{\"memory\": \"100Mi\"}}}]}"}}
	merged, _ = PodSpecPatchMerge(&wf, &tmpl)
	json.Unmarshal([]byte(merged), &spec)
	assert.Equal(t, "1.000", spec.Containers[0].Resources.Limits.Cpu().AsDec().String())
	assert.Equal(t, "104857600", spec.Containers[0].Resources.Limits.Memory().AsDec().String())

}
