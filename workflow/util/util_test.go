package util

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fakeClientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo/workflow/packer"
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

func TestResumeWorkflowCompressed(t *testing.T) {
	wfIf := fakeClientset.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
	origWf := unmarshalWF(suspendedWf)

	clearFunc := packer.SetMaxWorkflowSize(1156)
	defer clearFunc()
	err := packer.CompressWorkflowIfNeeded(origWf)
	assert.NoError(t, err)

	_, err = wfIf.Create(origWf)
	assert.NoError(t, err)

	err = ResumeWorkflow(wfIf, sqldb.ExplosiveOffloadNodeStatusRepo, "suspend", "")
	assert.NoError(t, err)

	wf, err := wfIf.Get("suspend", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotEmpty(t, wf.Status.CompressedNodes)
}

func TestResumeWorkflowOffloaded(t *testing.T) {
	wfIf := fakeClientset.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
	origWf := unmarshalWF(suspendedWf)

	origNodes := origWf.Status.Nodes

	origWf.Status.Nodes = nil
	origWf.Status.OffloadNodeStatusVersion = "123"

	_, err := wfIf.Create(origWf)
	assert.NoError(t, err)

	offloadNodeStatusRepo := &mocks.OffloadNodeStatusRepo{}
	offloadNodeStatusRepo.On("IsEnabled", mock.Anything).Return(true)
	offloadNodeStatusRepo.On("Get", "4f08d325-dc5a-43a3-9986-259e259e6ea3", "123").Return(origNodes, nil)
	offloadNodeStatusRepo.On("Save", "4f08d325-dc5a-43a3-9986-259e259e6ea3", mock.Anything, mock.Anything).Return("1234", nil)

	err = ResumeWorkflow(wfIf, offloadNodeStatusRepo, "suspend", "")
	assert.NoError(t, err)

	wf, err := wfIf.Get("suspend", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "1234", wf.Status.OffloadNodeStatusVersion)
}

var failedWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-04-10T15:35:59Z"
  generation: 5
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Failed
  name: fail-template
  resourceVersion: "240144"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/workflows/fail-template
  uid: 7e74dbb9-d681-4c22-9bed-a581ec28383f
spec:
  arguments: {}
  entrypoint: fail
  templates:
  - arguments: {}
    inputs: {}
    metadata: {}
    name: fail
    outputs: {}
    steps:
    - - arguments: {}
        name: approve
        template: approve
  - arguments: {}
    container:
      args:
      - exit 1
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: approve
    outputs: {}
status:
  conditions:
  - status: "True"
    type: Completed
  finishedAt: "2020-04-10T15:36:03Z"
  message: child 'fail-template-2878444447' failed
  nodes:
    fail-template:
      children:
      - fail-template-2384955452
      displayName: fail-template
      finishedAt: "2020-04-10T15:36:03Z"
      id: fail-template
      message: child 'fail-template-2878444447' failed
      name: fail-template
      outboundNodes:
      - fail-template-2878444447
      phase: Failed
      startedAt: "2020-04-10T15:35:59Z"
      templateName: fail
      templateScope: local/fail-template
      type: Steps
    fail-template-2384955452:
      boundaryID: fail-template
      children:
      - fail-template-2878444447
      displayName: '[0]'
      finishedAt: "2020-04-10T15:36:03Z"
      id: fail-template-2384955452
      message: child 'fail-template-2878444447' failed
      name: fail-template[0]
      phase: Failed
      startedAt: "2020-04-10T15:35:59Z"
      templateName: fail
      templateScope: local/fail-template
      type: StepGroup
    fail-template-2878444447:
      boundaryID: fail-template
      displayName: approve
      finishedAt: "2020-04-10T15:36:02Z"
      id: fail-template-2878444447
      message: failed with exit code 1
      name: fail-template[0].approve
      outputs:
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: fail-template/fail-template-2878444447/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
      phase: Failed
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2020-04-10T15:35:59Z"
      templateName: approve
      templateScope: local/fail-template
      type: Pod
  phase: Failed
  resourcesDuration:
    cpu: 2
    memory: 0
  startedAt: "2020-04-10T15:35:59Z"

`

func TestRetryWorkflowCompressed(t *testing.T) {
	wfIf := fakeClientset.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
	origWf := unmarshalWF(failedWf)
	kubeCs := fake.NewSimpleClientset()

	_, err := kubeCs.CoreV1().Pods("").Create(&v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "fail-template-2878444447"}})
	assert.NoError(t, err)

	clearFunc := packer.SetMaxWorkflowSize(1684)
	defer clearFunc()
	err = packer.CompressWorkflowIfNeeded(origWf)
	assert.NoError(t, err)

	_, err = wfIf.Create(origWf)
	assert.NoError(t, err)

	clearFunc = packer.SetMaxWorkflowSize(1557)
	defer clearFunc()
	wf, err := RetryWorkflow(kubeCs, sqldb.ExplosiveOffloadNodeStatusRepo, wfIf, origWf)
	assert.NoError(t, err)
	assert.NotEmpty(t, wf.Status.CompressedNodes)
}

func TestRetryWorkflowOffloaded(t *testing.T) {
	wfIf := fakeClientset.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
	origWf := unmarshalWF(failedWf)
	kubeCs := fake.NewSimpleClientset()
	_, err := kubeCs.CoreV1().Pods("").Create(&v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "fail-template-2878444447"}})
	assert.NoError(t, err)

	origNodes := origWf.Status.Nodes

	origWf.Status.Nodes = nil
	origWf.Status.OffloadNodeStatusVersion = "123"

	_, err = wfIf.Create(origWf)
	assert.NoError(t, err)

	offloadNodeStatusRepo := &mocks.OffloadNodeStatusRepo{}
	offloadNodeStatusRepo.On("IsEnabled", mock.Anything).Return(true)
	offloadNodeStatusRepo.On("Get", "7e74dbb9-d681-4c22-9bed-a581ec28383f", "123").Return(origNodes, nil)
	offloadNodeStatusRepo.On("Save", "7e74dbb9-d681-4c22-9bed-a581ec28383f", mock.Anything, mock.Anything).Return("1234", nil)

	_, err = RetryWorkflow(kubeCs, offloadNodeStatusRepo, wfIf, origWf)
	assert.NoError(t, err)

	wf, err := wfIf.Get("fail-template", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "1234", wf.Status.OffloadNodeStatusVersion)
}
