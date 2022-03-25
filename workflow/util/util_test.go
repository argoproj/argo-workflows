package util

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	hydratorfake "github.com/argoproj/argo-workflows/v3/workflow/hydrator/fake"
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
	wf := wfv1.MustUnmarshalWorkflow(workflowYaml)
	newWf := wf.DeepCopy()
	wfClientSet := argofake.NewSimpleClientset()
	ctx := context.Background()
	newWf, err := SubmitWorkflow(ctx, nil, wfClientSet, "test-namespace", newWf, &wfv1.SubmitOpts{DryRun: true})
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
			Phase: wfv1.WorkflowFailed,
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
			dir := t.TempDir()
			var filePaths []string
			for i := range tc.fileNames {
				content := []byte(tc.contents[i])
				tmpfn := filepath.Join(dir, tc.fileNames[i])
				filePaths = append(filePaths, tmpfn)
				err := ioutil.WriteFile(tmpfn, content, 0o600)
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
			dir := t.TempDir()
			var filePaths []string
			for i := range tc.fileNames {
				content := []byte(tc.contents[i])
				tmpfn := filepath.Join(dir, tc.fileNames[i])
				filePaths = append(filePaths, tmpfn)
				if tc.exists[i] {
					err := ioutil.WriteFile(tmpfn, content, 0o600)
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
	wfv1.MustUnmarshal([]byte(merged), &spec)
	assert.Equal(t, "1.000", spec.Containers[0].Resources.Limits.Cpu().AsDec().String())
	assert.Equal(t, "104857600", spec.Containers[0].Resources.Limits.Memory().AsDec().String())

	tmpl = wfv1.Template{PodSpecPatch: yamlStr}
	wf = wfv1.Workflow{Spec: wfv1.WorkflowSpec{PodSpecPatch: "{\"containers\":[{\"name\":\"main\", \"resources\":{\"limits\":{\"memory\": \"100Mi\"}}}]}"}}
	merged, err = PodSpecPatchMerge(&wf, &tmpl)
	assert.NoError(t, err)
	wfv1.MustUnmarshal([]byte(merged), &spec)
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
  
  entrypoint: suspend
  templates:
  - 
    inputs: {}
    metadata: {}
    name: suspend
    outputs: {}
    steps:
    - - 
        name: approve
        template: approve
  - 
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
	wfIf := argofake.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
	origWf := wfv1.MustUnmarshalWorkflow(suspendedWf)

	ctx := context.Background()
	_, err := wfIf.Create(ctx, origWf, metav1.CreateOptions{})
	assert.NoError(t, err)

	// will return error as displayName does not match any nodes
	err = ResumeWorkflow(ctx, wfIf, hydratorfake.Noop, "suspend", "displayName=nonexistant")
	assert.Error(t, err)

	// displayName didn't match suspend node so should still be running
	wf, err := wfIf.Get(ctx, "suspend", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByDisplayName("approve").Phase)

	err = ResumeWorkflow(ctx, wfIf, hydratorfake.Noop, "suspend", "displayName=approve")
	assert.NoError(t, err)

	// displayName matched node so has succeeded
	wf, err = wfIf.Get(ctx, "suspend", metav1.GetOptions{})
	if assert.NoError(t, err) {
		assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByDisplayName("approve").Phase)
	}
}

func TestStopWorkflowByNodeName(t *testing.T) {
	wfIf := argofake.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
	origWf := wfv1.MustUnmarshalWorkflow(suspendedWf)

	ctx := context.Background()
	_, err := wfIf.Create(ctx, origWf, metav1.CreateOptions{})
	assert.NoError(t, err)

	// will return error as displayName does not match any nodes
	err = StopWorkflow(ctx, wfIf, hydratorfake.Noop, "suspend", "displayName=nonexistant", "error occurred")
	assert.Error(t, err)

	// displayName didn't match suspend node so should still be running
	wf, err := wfIf.Get(ctx, "suspend", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByDisplayName("approve").Phase)

	err = StopWorkflow(ctx, wfIf, hydratorfake.Noop, "suspend", "displayName=approve", "error occurred")
	assert.NoError(t, err)

	// displayName matched node so has succeeded
	wf, err = wfIf.Get(ctx, "suspend", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, wf.Status.Nodes.FindByDisplayName("approve").Phase)
}

// Regression test for #6478
func TestAddParamToGlobalScopeValueNil(t *testing.T) {
	paramValue := wfv1.AnyString("test")
	wf := wfv1.Workflow{
		Status: wfv1.WorkflowStatus{
			Outputs: &wfv1.Outputs{
				Parameters: []wfv1.Parameter{
					{
						Name:       "test",
						Value:      &paramValue,
						GlobalName: "global_output_param",
					},
				},
			},
		},
	}

	p := AddParamToGlobalScope(&wf, nil, wfv1.Parameter{
		Name:       "test",
		Value:      nil,
		GlobalName: "test",
	})
	assert.False(t, p)
}

var susWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: suspend-template
spec:
  
  entrypoint: suspend
  templates:
  - 
    inputs: {}
    metadata: {}
    name: suspend
    outputs: {}
    steps:
    - - 
        name: approve
        template: approve
    - - arguments:
          parameters:
          - name: message
            value: '{{steps.approve.outputs.parameters.message}}'
        name: release
        template: whalesay
  - 
    inputs: {}
    metadata: {}
    name: approve
    outputs:
      parameters:
      - name: message
        valueFrom:
          supplied: {}
    suspend: {}
  - 
    container:
      args:
      - '{{inputs.parameters.message}}'
      command:
      - cowsay
      image: docker/whalesay
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: message
    metadata: {}
    name: whalesay
    outputs: {}
status:
  finishedAt: null
  nodes:
    suspend-template-kgfn7:
      children:
      - suspend-template-kgfn7-1405476480
      displayName: suspend-template-kgfn7
      finishedAt: null
      id: suspend-template-kgfn7
      name: suspend-template-kgfn7
      phase: Running
      startedAt: "2020-06-25T18:01:56Z"
      templateName: suspend
      templateScope: local/suspend-template-kgfn7
      type: Steps
    suspend-template-kgfn7-1405476480:
      boundaryID: suspend-template-kgfn7
      children:
      - suspend-template-kgfn7-2667278707
      displayName: '[0]'
      finishedAt: null
      id: suspend-template-kgfn7-1405476480
      name: suspend-template-kgfn7[0]
      phase: Running
      startedAt: "2020-06-25T18:01:56Z"
      templateName: suspend
      templateScope: local/suspend-template-kgfn7
      type: StepGroup
    suspend-template-kgfn7-2667278707:
      boundaryID: suspend-template-kgfn7
      displayName: approve
      finishedAt: null
      id: suspend-template-kgfn7-2667278707
      name: suspend-template-kgfn7[0].approve
      outputs:
        parameters:
        - name: message
          valueFrom:
            supplied: {}
        - name: message2
          globalName: message-global-param
          valueFrom:
            supplied: {}
      phase: Running
      startedAt: "2020-06-25T18:01:56Z"
      templateName: approve
      templateScope: local/suspend-template-kgfn7
      type: Suspend
  phase: Running
  startedAt: "2020-06-25T18:01:56Z"
`

func TestUpdateSuspendedNode(t *testing.T) {
	wfIf := argofake.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
	origWf := wfv1.MustUnmarshalWorkflow(susWorkflow)

	ctx := context.Background()
	_, err := wfIf.Create(ctx, origWf, metav1.CreateOptions{})
	if assert.NoError(t, err) {
		err = updateSuspendedNode(ctx, wfIf, hydratorfake.Noop, "does-not-exist", "displayName=approve", SetOperationValues{OutputParameters: map[string]string{"message": "Hello World"}})
		assert.EqualError(t, err, "workflows.argoproj.io \"does-not-exist\" not found")
		err = updateSuspendedNode(ctx, wfIf, hydratorfake.Noop, "suspend-template", "displayName=does-not-exists", SetOperationValues{OutputParameters: map[string]string{"message": "Hello World"}})
		assert.EqualError(t, err, "currently, set only targets suspend nodes: no suspend nodes matching nodeFieldSelector: displayName=does-not-exists")
		err = updateSuspendedNode(ctx, wfIf, hydratorfake.Noop, "suspend-template", "displayName=approve", SetOperationValues{OutputParameters: map[string]string{"does-not-exist": "Hello World"}})
		assert.EqualError(t, err, "node is not expecting output parameter 'does-not-exist'")
		err = updateSuspendedNode(ctx, wfIf, hydratorfake.Noop, "suspend-template", "displayName=approve", SetOperationValues{OutputParameters: map[string]string{"message": "Hello World"}})
		assert.NoError(t, err)
		err = updateSuspendedNode(ctx, wfIf, hydratorfake.Noop, "suspend-template", "name=suspend-template-kgfn7[0].approve", SetOperationValues{OutputParameters: map[string]string{"message2": "Hello World 2"}})
		assert.NoError(t, err)

		//make sure global variable was updated
		wf, err := wfIf.Get(ctx, "suspend-template", metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, "Hello World 2", wf.Status.Outputs.Parameters[0].Value.String())
	}

	noSpaceWf := wfv1.MustUnmarshalWorkflow(susWorkflow)
	noSpaceWf.Name = "suspend-template-no-outputs"
	node := noSpaceWf.Status.Nodes["suspend-template-kgfn7-2667278707"]
	node.Outputs = nil
	noSpaceWf.Status.Nodes["suspend-template-kgfn7-2667278707"] = node
	_, err = wfIf.Create(ctx, noSpaceWf, metav1.CreateOptions{})
	if assert.NoError(t, err) {
		err = updateSuspendedNode(ctx, wfIf, hydratorfake.Noop, "suspend-template-no-outputs", "displayName=approve", SetOperationValues{OutputParameters: map[string]string{"message": "Hello World"}})
		assert.EqualError(t, err, "cannot set output parameters because node is not expecting any raw parameters")
	}
}

func TestSelectorMatchesNode(t *testing.T) {
	tests := map[string]struct {
		selector string
		outcome  bool
	}{
		"idFound": {
			selector: "id=123",
			outcome:  true,
		},
		"idNotFound": {
			selector: "id=321",
			outcome:  false,
		},
		"nameFound": {
			selector: "name=failed-node",
			outcome:  true,
		},
		"phaseFound": {
			selector: "phase=Failed",
			outcome:  true,
		},
		"randomNotFound": {
			selector: "foo=Failed",
			outcome:  false,
		},
		"templateFound": {
			selector: "templateRef.name=templateName",
			outcome:  true,
		},
		"inputFound": {
			selector: "inputs.parameters.myparam.value=abc",
			outcome:  true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			node := wfv1.NodeStatus{ID: "123", Name: "failed-node", Phase: wfv1.NodeFailed, TemplateRef: &wfv1.TemplateRef{
				Name:     "templateName",
				Template: "template",
			},
				Inputs: &wfv1.Inputs{
					Parameters: []wfv1.Parameter{
						{
							Name:  "myparam",
							Value: wfv1.AnyStringPtr("abc"),
						},
					},
				},
			}
			selector, err := fields.ParseSelector(tc.selector)
			assert.NoError(t, err)
			if tc.outcome {
				assert.True(t, SelectorMatchesNode(selector, node))
			} else {
				assert.False(t, SelectorMatchesNode(selector, node))
			}
		})
	}
}

func TestGetNodeType(t *testing.T) {
	t.Run("getNodeType", func(t *testing.T) {
		assert.Equal(t, wfv1.NodeTypePod, GetNodeType(&wfv1.Template{Script: &wfv1.ScriptTemplate{}}))
		assert.Equal(t, wfv1.NodeTypePod, GetNodeType(&wfv1.Template{Container: &v1.Container{}}))
		assert.Equal(t, wfv1.NodeTypePod, GetNodeType(&wfv1.Template{Resource: &wfv1.ResourceTemplate{}}))
		assert.NotEqual(t, wfv1.NodeTypePod, GetNodeType(&wfv1.Template{Steps: []wfv1.ParallelSteps{}}))
		assert.NotEqual(t, wfv1.NodeTypePod, GetNodeType(&wfv1.Template{DAG: &wfv1.DAGTemplate{}}))
		assert.NotEqual(t, wfv1.NodeTypePod, GetNodeType(&wfv1.Template{Suspend: &wfv1.SuspendTemplate{}}))
	})
}

func TestApplySubmitOpts(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		assert.NoError(t, ApplySubmitOpts(&wfv1.Workflow{}, nil))
	})
	t.Run("InvalidLabels", func(t *testing.T) {
		assert.Error(t, ApplySubmitOpts(&wfv1.Workflow{}, &wfv1.SubmitOpts{Labels: "a"}))
	})
	t.Run("Labels", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		err := ApplySubmitOpts(wf, &wfv1.SubmitOpts{Labels: "a=1,b=1"})
		assert.NoError(t, err)
		assert.Len(t, wf.GetLabels(), 2)
	})
	t.Run("MergeLabels", func(t *testing.T) {
		wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"a": "0", "b": "0"}}}
		err := ApplySubmitOpts(wf, &wfv1.SubmitOpts{Labels: "a=1"})
		assert.NoError(t, err)
		if assert.Len(t, wf.GetLabels(), 2) {
			assert.Equal(t, "1", wf.GetLabels()["a"])
			assert.Equal(t, "0", wf.GetLabels()["b"])
		}
	})
	t.Run("InvalidParameters", func(t *testing.T) {
		assert.Error(t, ApplySubmitOpts(&wfv1.Workflow{}, &wfv1.SubmitOpts{Parameters: []string{"a"}}))
	})
	t.Run("Parameters", func(t *testing.T) {
		wf := &wfv1.Workflow{
			Spec: wfv1.WorkflowSpec{
				Arguments: wfv1.Arguments{
					Parameters: []wfv1.Parameter{{Name: "a", Value: wfv1.AnyStringPtr("0")}},
				},
			},
		}
		err := ApplySubmitOpts(wf, &wfv1.SubmitOpts{Parameters: []string{"a=81861780812"}})
		assert.NoError(t, err)
		parameters := wf.Spec.Arguments.Parameters
		if assert.Len(t, parameters, 1) {
			assert.Equal(t, "a", parameters[0].Name)
			assert.Equal(t, "81861780812", parameters[0].Value.String())
		}
	})
	t.Run("PodPriorityClassName", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		err := ApplySubmitOpts(wf, &wfv1.SubmitOpts{PodPriorityClassName: "abc"})
		assert.NoError(t, err)
		assert.Equal(t, "abc", wf.Spec.PodPriorityClassName)
	})
}

func TestReadParametersFile(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	defer func() { _ = os.Remove(file.Name()) }()
	err = ioutil.WriteFile(file.Name(), []byte(`a: 81861780812`), 0o600)
	assert.NoError(t, err)
	opts := &wfv1.SubmitOpts{}
	err = ReadParametersFile(file.Name(), opts)
	assert.NoError(t, err)
	parameters := opts.Parameters
	if assert.Len(t, parameters, 1) {
		assert.Equal(t, "a=81861780812", parameters[0])
	}
}

func TestFormulateResubmitWorkflow(t *testing.T) {
	t.Run("Labels", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					common.LabelKeyControllerInstanceID:    "1",
					common.LabelKeyClusterWorkflowTemplate: "1",
					common.LabelKeyCronWorkflow:            "1",
					common.LabelKeyWorkflowTemplate:        "1",
					common.LabelKeyCreator:                 "1",
					common.LabelKeyPhase:                   "1",
					common.LabelKeyCompleted:               "1",
					common.LabelKeyWorkflowArchivingStatus: "1",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "test",
						Name:       "testObj",
					},
				},
			},
		}
		wf, err := FormulateResubmitWorkflow(wf, false)
		if assert.NoError(t, err) {
			assert.Contains(t, wf.GetLabels(), common.LabelKeyControllerInstanceID)
			assert.Contains(t, wf.GetLabels(), common.LabelKeyClusterWorkflowTemplate)
			assert.Contains(t, wf.GetLabels(), common.LabelKeyCronWorkflow)
			assert.Contains(t, wf.GetLabels(), common.LabelKeyWorkflowTemplate)
			assert.NotContains(t, wf.GetLabels(), common.LabelKeyCreator)
			assert.NotContains(t, wf.GetLabels(), common.LabelKeyPhase)
			assert.NotContains(t, wf.GetLabels(), common.LabelKeyCompleted)
			assert.NotContains(t, wf.GetLabels(), common.LabelKeyWorkflowArchivingStatus)
			assert.Contains(t, wf.GetLabels(), common.LabelKeyPreviousWorkflowName)
			assert.Equal(t, 1, len(wf.OwnerReferences))
			assert.Equal(t, "test", wf.OwnerReferences[0].APIVersion)
			assert.Equal(t, "testObj", wf.OwnerReferences[0].Name)
		}
	})
}

var deepDeleteOfNodes = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-09-16T16:07:54Z"
  generateName: steps-
  generation: 13
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Failed
  name: steps-9fkqc
  resourceVersion: "383660"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/workflows/steps-9fkqc
  uid: 241a39ef-4ff1-487f-8461-98df5d2b50fb
spec:
  
  entrypoint: foo
  templates:
  - 
    inputs: {}
    metadata: {}
    name: foo
    outputs: {}
    steps:
    - - 
        name: pass
        template: pass
    - - 
        name: fail
        template: fail
  - 
    container:
      args:
      - exit 0
      command:
      - sh
      - -c
      image: alpine
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: pass
    outputs: {}
  - 
    container:
      args:
      - exit 1
      command:
      - sh
      - -c
      image: alpine
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: fail
    outputs: {}
status:
  conditions:
  - status: "True"
    type: Completed
  finishedAt: "2020-09-16T16:09:32Z"
  message: child 'steps-9fkqc-3224593506' failed
  nodes:
    steps-9fkqc:
      children:
      - steps-9fkqc-2929074125
      displayName: steps-9fkqc
      finishedAt: "2020-09-16T16:09:32Z"
      id: steps-9fkqc
      message: child 'steps-9fkqc-3224593506' failed
      name: steps-9fkqc
      outboundNodes:
      - steps-9fkqc-3224593506
      phase: Failed
      startedAt: "2020-09-16T16:07:54Z"
      templateName: foo
      templateScope: local/steps-9fkqc
      type: Steps
    steps-9fkqc-1411266092:
      boundaryID: steps-9fkqc
      children:
      - steps-9fkqc-2862110744
      displayName: pass
      finishedAt: "2020-09-16T16:07:57Z"
      hostNodeName: minikube
      id: steps-9fkqc-1411266092
      name: steps-9fkqc[0].pass
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 1
      startedAt: "2020-09-16T16:07:54Z"
      templateName: pass
      templateScope: local/steps-9fkqc
      type: Pod
    steps-9fkqc-2862110744:
      boundaryID: steps-9fkqc
      children:
      - steps-9fkqc-3224593506
      displayName: '[1]'
      finishedAt: "2020-09-16T16:09:32Z"
      id: steps-9fkqc-2862110744
      message: child 'steps-9fkqc-3224593506' failed
      name: steps-9fkqc[1]
      phase: Failed
      startedAt: "2020-09-16T16:07:59Z"
      templateName: foo
      templateScope: local/steps-9fkqc
      type: StepGroup
    steps-9fkqc-2929074125:
      boundaryID: steps-9fkqc
      children:
      - steps-9fkqc-1411266092
      displayName: '[0]'
      finishedAt: "2020-09-16T16:07:59Z"
      id: steps-9fkqc-2929074125
      name: steps-9fkqc[0]
      phase: Succeeded
      startedAt: "2020-09-16T16:07:54Z"
      templateName: foo
      templateScope: local/steps-9fkqc
      type: StepGroup
    steps-9fkqc-3224593506:
      boundaryID: steps-9fkqc
      displayName: fail
      finishedAt: "2020-09-16T16:09:30Z"
      hostNodeName: minikube
      id: steps-9fkqc-3224593506
      message: failed with exit code 1
      name: steps-9fkqc[1].fail
      outputs:
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 2
        memory: 1
      startedAt: "2020-09-16T16:09:27Z"
      templateName: fail
      templateScope: local/steps-9fkqc
      type: Pod
  phase: Failed
  resourcesDuration:
    cpu: 4
    memory: 2
  startedAt: "2020-09-16T16:07:54Z"
`

func TestDeepDeleteNodes(t *testing.T) {
	wfIf := argofake.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
	origWf := wfv1.MustUnmarshalWorkflow(deepDeleteOfNodes)

	ctx := context.Background()
	wf, err := wfIf.Create(ctx, origWf, metav1.CreateOptions{})
	if assert.NoError(t, err) {
		newWf, _, err := FormulateRetryWorkflow(ctx, wf, false, "")
		assert.NoError(t, err)
		newWfBytes, err := yaml.Marshal(newWf)
		assert.NoError(t, err)
		assert.NotContains(t, string(newWfBytes), "steps-9fkqc-3224593506")
	}
}

func TestFormulateRetryWorkflow(t *testing.T) {
	ctx := context.Background()
	wfClient := argofake.NewSimpleClientset().ArgoprojV1alpha1().Workflows("my-ns")
	createdTime := metav1.Time{Time: time.Now().UTC()}
	finishedTime := metav1.Time{Time: createdTime.Add(time.Second * 2)}
	t.Run("Steps", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-steps",
				Labels: map[string]string{
					common.LabelKeyCompleted:               "true",
					common.LabelKeyWorkflowArchivingStatus: "Pending",
				},
			},
			Status: wfv1.WorkflowStatus{
				Phase:      wfv1.WorkflowFailed,
				StartedAt:  createdTime,
				FinishedAt: finishedTime,
				Nodes: map[string]wfv1.NodeStatus{
					"failed-node":    {Name: "failed-node", StartedAt: createdTime, FinishedAt: finishedTime, Phase: wfv1.NodeFailed, Message: "failed"},
					"succeeded-node": {Name: "succeeded-node", StartedAt: createdTime, FinishedAt: finishedTime, Phase: wfv1.NodeSucceeded, Message: "succeeded"}},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		wf, _, err = FormulateRetryWorkflow(ctx, wf, false, "")
		if assert.NoError(t, err) {
			assert.Equal(t, wfv1.WorkflowRunning, wf.Status.Phase)
			assert.Equal(t, metav1.Time{}, wf.Status.FinishedAt)
			assert.True(t, wf.Status.StartedAt.After(createdTime.Time))
			assert.NotContains(t, wf.Labels, common.LabelKeyCompleted)
			assert.NotContains(t, wf.Labels, common.LabelKeyWorkflowArchivingStatus)
			for _, node := range wf.Status.Nodes {
				switch node.Phase {
				case wfv1.NodeSucceeded:
					assert.Equal(t, "succeeded", node.Message)
					assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
					assert.Equal(t, createdTime, node.StartedAt)
					assert.Equal(t, finishedTime, node.FinishedAt)
				case wfv1.NodeFailed:
					assert.Equal(t, "", node.Message)
					assert.Equal(t, wfv1.NodeRunning, node.Phase)
					assert.Equal(t, metav1.Time{}, node.FinishedAt)
					assert.True(t, node.StartedAt.After(createdTime.Time))
				}
			}
		}
	})
	t.Run("DAG", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "my-dag",
				Labels: map[string]string{},
			},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowFailed,
				Nodes: map[string]wfv1.NodeStatus{
					"": {Phase: wfv1.NodeFailed, Type: wfv1.NodeTypeTaskGroup}},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		wf, _, err = FormulateRetryWorkflow(ctx, wf, false, "")
		if assert.NoError(t, err) {
			if assert.Len(t, wf.Status.Nodes, 1) {
				assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes[""].Phase)
			}

		}
	})
}

func TestFromUnstructuredObj(t *testing.T) {
	un := &unstructured.Unstructured{}
	wfv1.MustUnmarshal([]byte(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: example-integers
spec:
  schedule: "* * * * *"
  workflowSpec:
    entrypoint: whalesay
    templates:
      - name: whalesay
        inputs:
          parameters:
            - name: age
              value: 20
        container:
          image: my-image`), un)
	x := &wfv1.CronWorkflow{}
	err := FromUnstructuredObj(un, x)
	assert.NoError(t, err)
}

func TestToUnstructured(t *testing.T) {
	un, err := ToUnstructured(&wfv1.Workflow{})
	if assert.NoError(t, err) {
		gv := un.GetObjectKind().GroupVersionKind()
		assert.Equal(t, workflow.WorkflowKind, gv.Kind)
		assert.Equal(t, workflow.Group, gv.Group)
		assert.Equal(t, workflow.Version, gv.Version)
	}
}

func TestGetTemplateFromNode(t *testing.T) {
	cases := []struct {
		inputNode            wfv1.NodeStatus
		expectedTemplateName string
	}{
		{
			inputNode: wfv1.NodeStatus{
				TemplateRef: &wfv1.TemplateRef{
					Name:         "foo-workflowtemplate",
					Template:     "foo-template",
					ClusterScope: false,
				},
				TemplateName: "",
			},
			expectedTemplateName: "foo-template",
		},
		{
			inputNode: wfv1.NodeStatus{
				TemplateRef:  nil,
				TemplateName: "bar-template",
			},
			expectedTemplateName: "bar-template",
		},
	}

	for _, tc := range cases {
		actual := getTemplateFromNode(tc.inputNode)
		assert.Equal(t, tc.expectedTemplateName, actual)
	}
}
