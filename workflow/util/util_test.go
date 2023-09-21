package util

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/creator"
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
	onExitNode := wfv1.NodeStatus{
		Name:  onExitName,
		Phase: wfv1.NodeSucceeded,
	}
	wf.Status.Nodes.Set(onExitID, onExitNode)
	newWF, err := FormulateResubmitWorkflow(context.Background(), &wf, true, nil)
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
				err := os.WriteFile(tmpfn, content, 0o600)
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
					err := os.WriteFile(tmpfn, content, 0o600)
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
	t.Run("Withought user info", func(t *testing.T) {
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
			assert.Equal(t, "", wf.Status.Nodes.FindByDisplayName("approve").Message)
		}
	})

	t.Run("With user info", func(t *testing.T) {
		wfIf := argofake.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
		origWf := wfv1.MustUnmarshalWorkflow(suspendedWf)

		ctx := context.WithValue(context.TODO(), auth.ClaimsKey,
			&types.Claims{Claims: jwt.Claims{Subject: strings.Repeat("x", 63) + "y"}, Email: "my@email", PreferredUsername: "username"})
		uim := creator.UserInfoMap(ctx)

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
			assert.Equal(t, fmt.Sprintf("Resumed by: %v", uim), wf.Status.Nodes.FindByDisplayName("approve").Message)
		}
	})
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

	origWf.Status = wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded}
	origWf.Name = "succeeded-wf"
	_, err = wfIf.Create(ctx, origWf, metav1.CreateOptions{})
	assert.NoError(t, err)
	err = StopWorkflow(ctx, wfIf, hydratorfake.Noop, "succeeded-wf", "", "")
	assert.EqualError(t, err, "cannot shutdown a completed workflow: workflow: \"succeeded-wf\", namespace: \"\"")
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

		// make sure global variable was updated
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
	file, err := os.CreateTemp("", "")
	assert.NoError(t, err)
	defer func() { _ = os.Remove(file.Name()) }()
	err = os.WriteFile(file.Name(), []byte(`a: 81861780812`), 0o600)
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
		wf, err := FormulateResubmitWorkflow(context.Background(), wf, false, nil)
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
	t.Run("OverrideCreatorLabels", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					common.LabelKeyCreator:                  "xxxx-xxxx-xxxx",
					common.LabelKeyCreatorEmail:             "foo.at.example.com",
					common.LabelKeyCreatorPreferredUsername: "foo",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "test",
						Name:       "testObj",
					},
				},
			},
		}
		ctx := context.WithValue(context.Background(), auth.ClaimsKey, &types.Claims{
			Claims:            jwt.Claims{Subject: "yyyy-yyyy-yyyy-yyyy"},
			Email:             "bar.at.example.com",
			PreferredUsername: "bar",
		})
		wf, err := FormulateResubmitWorkflow(ctx, wf, false, nil)
		if assert.NoError(t, err) {
			assert.Equal(t, "yyyy-yyyy-yyyy-yyyy", wf.Labels[common.LabelKeyCreator])
			assert.Equal(t, "bar.at.example.com", wf.Labels[common.LabelKeyCreatorEmail])
			assert.Equal(t, "bar", wf.Labels[common.LabelKeyCreatorPreferredUsername])
		}
	})
	t.Run("UnlabelCreatorLabels", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					common.LabelKeyCreator:                  "xxxx-xxxx-xxxx",
					common.LabelKeyCreatorEmail:             "foo.at.example.com",
					common.LabelKeyCreatorPreferredUsername: "foo",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "test",
						Name:       "testObj",
					},
				},
			},
		}
		wf, err := FormulateResubmitWorkflow(context.Background(), wf, false, nil)
		if assert.NoError(t, err) {
			assert.Emptyf(t, wf.Labels[common.LabelKeyCreator], "should not %s label when a workflow is resubmitted by an unauthenticated request", common.LabelKeyCreator)
			assert.Emptyf(t, wf.Labels[common.LabelKeyCreatorEmail], "should not %s label when a workflow is resubmitted by an unauthenticated request", common.LabelKeyCreatorEmail)
			assert.Emptyf(t, wf.Labels[common.LabelKeyCreatorPreferredUsername], "should not %s label when a workflow is resubmitted by an unauthenticated request", common.LabelKeyCreatorPreferredUsername)
		}
	})
	t.Run("OverrideParams", func(t *testing.T) {
		wf := &wfv1.Workflow{
			Spec: wfv1.WorkflowSpec{Arguments: wfv1.Arguments{
				Parameters: []wfv1.Parameter{
					{Name: "message", Value: wfv1.AnyStringPtr("default")},
				},
			}},
		}
		wf, err := FormulateResubmitWorkflow(context.Background(), wf, false, []string{"message=modified"})
		if assert.NoError(t, err) {
			assert.Equal(t, "modified", wf.Spec.Arguments.Parameters[0].Value.String())
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
		newWf, _, err := FormulateRetryWorkflow(ctx, wf, false, "", nil)
		assert.NoError(t, err)
		newWfBytes, err := yaml.Marshal(newWf)
		assert.NoError(t, err)
		assert.NotContains(t, string(newWfBytes), "steps-9fkqc-3224593506")
	}
}

var exitHandler = `
metadata:
  name: retry-script-6xt68
  generateName: retry-script-
  uid: d0cb3766-6bbc-48f0-84c3-820ababfc8ff
  resourceVersion: '9897789'
  generation: 4
  creationTimestamp: '2022-11-09T09:08:23Z'
  labels:
    workflows.argoproj.io/completed: 'true'
    workflows.argoproj.io/phase: Failed
spec:
  templates:
    - name: retry-script
      inputs: {}
      outputs: {}
      metadata: {}
      script:
        name: ''
        image: python:alpine3.6
        command:
          - python
        resources: {}
        source: |
          import sys; 
          exit_code = 1; 
          sys.exit(exit_code)
    - name: handler
      inputs:
        parameters:
          - name: message
      outputs: {}
      metadata: {}
      container:
        name: ''
        image: alpine:latest
        command:
          - sh
          - '-c'
        args:
          - echo {{inputs.parameters.message}}
        resources: {}
  entrypoint: retry-script
  arguments: {}
  hooks:
    exit:
      template: handler
      arguments:
        parameters:
          - name: message
            value: '{{ workflow.status }}'
status:
  phase: Failed
  startedAt: '2022-11-09T09:08:23Z'
  finishedAt: '2022-11-09T09:08:43Z'
  progress: 1/2
  message: Error (exit code 1)
  nodes:
    retry-script-6xt68:
      id: retry-script-6xt68
      name: retry-script-6xt68
      displayName: retry-script-6xt68
      type: Pod
      templateName: retry-script
      templateScope: local/retry-script-6xt68
      phase: Failed
      message: Error (exit code 1)
      startedAt: '2022-11-09T09:08:23Z'
      finishedAt: '2022-11-09T09:08:28Z'
      progress: 0/1
      resourcesDuration:
        cpu: 4
        memory: 4
      outputs:
        artifacts:
          - name: main-logs
            s3:
              key: retry-script-6xt68/retry-script-6xt68/main.log
        exitCode: '1'
      hostNodeName: minikube
    retry-script-6xt68-3924170365:
      id: retry-script-6xt68-3924170365
      name: retry-script-6xt68.onExit
      displayName: retry-script-6xt68.onExit
      type: Pod
      templateName: handler
      templateScope: local/retry-script-6xt68
      phase: Succeeded
      startedAt: '2022-11-09T09:08:33Z'
      finishedAt: '2022-11-09T09:08:38Z'
      progress: 1/1
      resourcesDuration:
        cpu: 3
        memory: 3
      inputs:
        parameters:
          - name: message
            value: Failed
      outputs:
        artifacts:
          - name: main-logs
            s3:
              key: >-
                retry-script-6xt68/retry-script-6xt68-handler-3924170365/main.log
        exitCode: '0'
      hostNodeName: minikube
  conditions:
    - type: PodRunning
      status: 'False'
    - type: Completed
      status: 'True'
  resourcesDuration:
    cpu: 7
    memory: 7
`

func TestRetryExitHandler(t *testing.T) {
	wfIf := argofake.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
	origWf := wfv1.MustUnmarshalWorkflow(exitHandler)

	ctx := context.Background()
	wf, err := wfIf.Create(ctx, origWf, metav1.CreateOptions{})
	if assert.NoError(t, err) {
		newWf, _, err := FormulateRetryWorkflow(ctx, wf, false, "", nil)
		assert.NoError(t, err)
		newWfBytes, err := yaml.Marshal(newWf)
		assert.NoError(t, err)
		t.Log(string(newWfBytes))
		assert.NotContains(t, string(newWfBytes), "retry-script-6xt68-3924170365")
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
		wf, _, err = FormulateRetryWorkflow(ctx, wf, false, "", nil)
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
		wf, _, err = FormulateRetryWorkflow(ctx, wf, false, "", nil)
		if assert.NoError(t, err) {
			if assert.Len(t, wf.Status.Nodes, 1) {
				assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes[""].Phase)
			}

		}
	})
	t.Run("Skipped and Suspended Nodes", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "wf-with-skipped-and-suspended-nodes",
				Labels: map[string]string{},
			},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowFailed,
				Nodes: map[string]wfv1.NodeStatus{
					"entrypoint": {ID: "entrypoint", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, Children: []string{"suspended", "skipped"}},
					"suspended": {
						ID:         "suspended",
						Phase:      wfv1.NodeSucceeded,
						Type:       wfv1.NodeTypeSuspend,
						BoundaryID: "entrypoint",
						Children:   []string{"child"},
						Outputs: &wfv1.Outputs{Parameters: []wfv1.Parameter{{
							Name:      "param-1",
							Value:     wfv1.AnyStringPtr("3"),
							ValueFrom: &wfv1.ValueFrom{Supplied: &wfv1.SuppliedValueFrom{}},
						}}}},
					"child":   {ID: "child", Phase: wfv1.NodeSkipped, Type: wfv1.NodeTypeSkipped, BoundaryID: "suspended"},
					"skipped": {ID: "skipped", Phase: wfv1.NodeSkipped, Type: wfv1.NodeTypeSkipped, BoundaryID: "entrypoint"},
				}},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		wf, _, err = FormulateRetryWorkflow(ctx, wf, true, "id=suspended", nil)
		if assert.NoError(t, err) {
			if assert.Len(t, wf.Status.Nodes, 3) {
				assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["entrypoint"].Phase)
				assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["suspended"].Phase)
				assert.Equal(t, wfv1.Parameter{
					Name:      "param-1",
					Value:     nil,
					ValueFrom: &wfv1.ValueFrom{Supplied: &wfv1.SuppliedValueFrom{}},
				}, wf.Status.Nodes["suspended"].Outputs.Parameters[0])
				assert.Equal(t, wfv1.NodeSkipped, wf.Status.Nodes["skipped"].Phase)
			}
		}
	})
	t.Run("Nested DAG with Non-group Node Selected", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "my-nested-dag-1",
				Labels: map[string]string{},
			},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowFailed,
				Nodes: map[string]wfv1.NodeStatus{
					"my-nested-dag-1": {ID: "my-nested-dag-1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, Children: []string{"1"}},
					"1":               {ID: "1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, BoundaryID: "my-nested-dag-1", Children: []string{"2", "4"}},
					"2":               {ID: "2", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, BoundaryID: "1", Children: []string{"3"}},
					"3":               {ID: "3", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "2"},
					"4":               {ID: "4", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod, BoundaryID: "1"}},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		wf, _, err = FormulateRetryWorkflow(ctx, wf, true, "id=3", nil)
		if assert.NoError(t, err) {
			// Node #3, #4 are deleted and will be recreated so only 3 nodes left in wf.Status.Nodes
			if assert.Len(t, wf.Status.Nodes, 3) {
				assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["my-nested-dag-1"].Phase)
				// The parent group nodes should be running.
				assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["1"].Phase)
				assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["2"].Phase)
			}
		}
	})
	t.Run("Nested DAG without Node Selected", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "my-nested-dag-2",
				Labels: map[string]string{},
			},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowFailed,
				Nodes: map[string]wfv1.NodeStatus{
					"my-nested-dag-2": {ID: "my-nested-dag-2", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, Children: []string{"1"}},
					"1":               {ID: "1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, BoundaryID: "my-nested-dag-2", Children: []string{"2", "4"}},
					"2":               {ID: "2", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, BoundaryID: "1", Children: []string{"3"}},
					"3":               {ID: "3", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "2"},
					"4":               {ID: "4", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod, BoundaryID: "1"}},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		wf, _, err = FormulateRetryWorkflow(ctx, wf, true, "", nil)
		if assert.NoError(t, err) {
			// Node #2, #3, and #4 are deleted and will be recreated so only 2 nodes left in wf.Status.Nodes
			if assert.Len(t, wf.Status.Nodes, 4) {
				assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["my-nested-dag-2"].Phase)
				assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["1"].Phase)
				assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["2"].Phase)
				assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["3"].Phase)
				assert.Equal(t, "", string(wf.Status.Nodes["4"].Phase))
			}
		}
	})
	t.Run("OverrideParams", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "override-param-wf",
				Labels: map[string]string{},
			},
			Spec: wfv1.WorkflowSpec{Arguments: wfv1.Arguments{
				Parameters: []wfv1.Parameter{
					{Name: "message", Value: wfv1.AnyStringPtr("default")},
				},
			}},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowFailed,
				Nodes: map[string]wfv1.NodeStatus{
					"1": {ID: "1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup},
				}},
		}
		wf, _, err := FormulateRetryWorkflow(context.Background(), wf, false, "", []string{"message=modified"})
		if assert.NoError(t, err) {
			assert.Equal(t, "modified", wf.Spec.Arguments.Parameters[0].Value.String())
		}
	})

	t.Run("OverrideParamsSubmitFromWfTmpl", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "override-param-wf",
				Labels: map[string]string{},
			},
			Spec: wfv1.WorkflowSpec{Arguments: wfv1.Arguments{
				Parameters: []wfv1.Parameter{
					{Name: "message", Value: wfv1.AnyStringPtr("default")},
				},
			}},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowFailed,
				Nodes: map[string]wfv1.NodeStatus{
					"1": {ID: "1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup},
				},
				StoredWorkflowSpec: &wfv1.WorkflowSpec{Arguments: wfv1.Arguments{
					Parameters: []wfv1.Parameter{
						{Name: "message", Value: wfv1.AnyStringPtr("default")},
					}},
				}},
		}
		wf, _, err := FormulateRetryWorkflow(context.Background(), wf, false, "", []string{"message=modified"})
		if assert.NoError(t, err) {
			assert.Equal(t, "modified", wf.Spec.Arguments.Parameters[0].Value.String())
			assert.Equal(t, "modified", wf.Status.StoredWorkflowSpec.Arguments.Parameters[0].Value.String())
		}
	})

	t.Run("Fail on running workflow", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "running-workflow-1",
				Labels: map[string]string{},
			},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowRunning,
				Nodes: map[string]wfv1.NodeStatus{},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		_, _, err = FormulateRetryWorkflow(ctx, wf, false, "", nil)
		assert.Error(t, err)
	})

	t.Run("Fail on pending workflow", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "pending-workflow-1",
				Labels: map[string]string{},
			},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowPending,
				Nodes: map[string]wfv1.NodeStatus{},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		_, _, err = FormulateRetryWorkflow(ctx, wf, false, "", nil)
		assert.Error(t, err)
	})

	t.Run("Fail on successful workflow without restartSuccessful and nodeFieldSelector", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "successful-workflow-1",
				Labels: map[string]string{},
			},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowSucceeded,
				Nodes: map[string]wfv1.NodeStatus{
					"successful-workflow-1": {ID: "successful-workflow-1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, Children: []string{"1"}},
					"1":                     {ID: "1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, BoundaryID: "successful-workflow-1", Children: []string{"2"}},
					"2":                     {ID: "2", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "1"}},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		_, _, err = FormulateRetryWorkflow(ctx, wf, false, "", nil)
		assert.Error(t, err)
	})

	t.Run("Retry successful workflow with restartSuccessful and nodeFieldSelector", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "successful-workflow-2",
				Labels: map[string]string{},
			},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowSucceeded,
				Nodes: map[string]wfv1.NodeStatus{
					"successful-workflow-2": {ID: "successful-workflow-2", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, Children: []string{"1"}},
					"1":                     {ID: "1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, BoundaryID: "successful-workflow-2", Children: []string{"2", "4"}},
					"2":                     {ID: "2", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, BoundaryID: "1", Children: []string{"3"}},
					"3":                     {ID: "3", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "2"},
					"4":                     {ID: "4", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "1"}},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		assert.NoError(t, err)
		wf, _, err = FormulateRetryWorkflow(ctx, wf, true, "id=4", nil)
		if assert.NoError(t, err) {
			// Node #4 is deleted and will be recreated so only 4 nodes left in wf.Status.Nodes
			if assert.Len(t, wf.Status.Nodes, 4) {
				assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["successful-workflow-2"].Phase)
				// The parent group nodes should be running.
				assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["1"].Phase)
				assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["2"].Phase)
				assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["3"].Phase)
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
		actual := GetTemplateFromNode(tc.inputNode)
		assert.Equal(t, tc.expectedTemplateName, actual)
	}
}

var retryWorkflowWithNestedDAGsWithSuspendNodes = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  annotations:
    workflows.argoproj.io/pod-name-format: v2
  creationTimestamp: "2022-09-02T14:52:10Z"
  generation: 37
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Failed
  name: fail-two-nested-dag-suspend
  namespace: argo
  resourceVersion: "218451"
  uid: ba03145e-b4db-46c9-96e6-8cc15bf81a79
spec:
  arguments: {}
  entrypoint: outer-dag
  templates:
  - dag:
      tasks:
      - arguments: {}
        name: dag1-step1
        template: gen-random-int-javascript
      - arguments: {}
        dependencies:
        - dag1-step1
        name: dag1-step2
        template: approve
      - arguments: {}
        dependencies:
        - dag1-step2
        name: dag1-step3-middle1
        template: middle-dag1
      - arguments: {}
        dependencies:
        - dag1-step2
        name: dag1-step3-middle2
        template: middle-dag2
      - arguments: {}
        dependencies:
        - dag1-step3-middle1
        - dag1-step3-middle2
        name: dag1-step4
        template: approve
      - arguments: {}
        dependencies:
        - dag1-step4
        name: dag1-step5-tofail
        template: node-to-fail
    inputs: {}
    metadata: {}
    name: outer-dag
    outputs: {}
  - dag:
      tasks:
      - arguments: {}
        name: dag2-branch1-step1
        template: approve
      - arguments: {}
        dependencies:
        - dag2-branch1-step1
        name: dag2-branch1-step2
        template: inner-dag-1
    inputs: {}
    metadata: {}
    name: middle-dag1
    outputs: {}
  - dag:
      tasks:
      - arguments: {}
        name: dag2-branch2-step1
        template: inner-dag-1
      - arguments: {}
        dependencies:
        - dag2-branch2-step1
        name: dag2-branch2-step2
        template: approve
    inputs: {}
    metadata: {}
    name: middle-dag2
    outputs: {}
  - dag:
      tasks:
      - arguments: {}
        name: dag3-step1
        template: approve
      - arguments: {}
        dependencies:
        - dag3-step1
        name: dag3-step2
        template: gen-random-int-javascript
      - arguments: {}
        dependencies:
        - dag3-step2
        name: dag3-step3
        template: gen-random-int-javascript
    inputs: {}
    metadata: {}
    name: inner-dag-1
    outputs: {}
  - inputs: {}
    metadata: {}
    name: gen-random-int-javascript
    outputs: {}
    script:
      command:
      - node
      image: node:9.1-alpine
      name: ""
      resources: {}
      source: |
        var rand = Math.floor(Math.random() * 100);
        console.log(rand);
  - container:
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
    name: node-to-fail
    outputs: {}
  - inputs: {}
    metadata: {}
    name: approve
    outputs: {}
    suspend:
      duration: "1"
status:
  artifactGCStatus:
    notSpecified: true
  artifactRepositoryRef:
    artifactRepository: {}
    default: true
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2022-09-02T14:56:56Z"
  nodes:
    fail-two-nested-dag-suspend:
      children:
      - fail-two-nested-dag-suspend-1199792179
      displayName: fail-two-nested-dag-suspend
      finishedAt: "2022-09-02T14:56:56Z"
      id: fail-two-nested-dag-suspend
      name: fail-two-nested-dag-suspend
      outboundNodes:
      - fail-two-nested-dag-suspend-2528852583
      phase: Failed
      progress: 11/12
      resourcesDuration:
        cpu: 18
        memory: 18
      startedAt: "2022-09-02T14:56:23Z"
      templateName: outer-dag
      templateScope: local/fail-two-nested-dag-suspend
      type: DAG
    fail-two-nested-dag-suspend-437639056:
      boundaryID: fail-two-nested-dag-suspend-1841799687
      children:
      - fail-two-nested-dag-suspend-1250125036
      displayName: dag3-step3
      finishedAt: "2022-09-02T14:53:32Z"
      hostNodeName: k3d-k3s-default-server-0
      id: fail-two-nested-dag-suspend-437639056
      name: fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step3
      outputs:
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 3
        memory: 3
      startedAt: "2022-09-02T14:53:29Z"
      templateName: gen-random-int-javascript
      templateScope: local/fail-two-nested-dag-suspend
      type: Pod
    fail-two-nested-dag-suspend-454416675:
      boundaryID: fail-two-nested-dag-suspend-1841799687
      children:
      - fail-two-nested-dag-suspend-437639056
      displayName: dag3-step2
      finishedAt: "2022-09-02T14:53:22Z"
      hostNodeName: k3d-k3s-default-server-0
      id: fail-two-nested-dag-suspend-454416675
      name: fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step2
      outputs:
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 3
        memory: 3
      startedAt: "2022-09-02T14:53:19Z"
      templateName: gen-random-int-javascript
      templateScope: local/fail-two-nested-dag-suspend
      type: Pod
    fail-two-nested-dag-suspend-471194294:
      boundaryID: fail-two-nested-dag-suspend-1841799687
      children:
      - fail-two-nested-dag-suspend-454416675
      displayName: dag3-step1
      finishedAt: "2022-09-02T14:53:19Z"
      id: fail-two-nested-dag-suspend-471194294
      name: fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step1
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 9
        memory: 9
      startedAt: "2022-09-02T14:53:18Z"
      templateName: approve
      templateScope: local/fail-two-nested-dag-suspend
      type: Suspend
    fail-two-nested-dag-suspend-476458868:
      boundaryID: fail-two-nested-dag-suspend-2864264609
      children:
      - fail-two-nested-dag-suspend-2781431063
      displayName: dag2-branch2-step1
      finishedAt: "2022-09-02T14:56:44Z"
      id: fail-two-nested-dag-suspend-476458868
      name: fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1
      outboundNodes:
      - fail-two-nested-dag-suspend-2814986301
      phase: Succeeded
      progress: 5/6
      resourcesDuration:
        cpu: 9
        memory: 9
      startedAt: "2022-09-02T14:56:23Z"
      templateName: inner-dag-1
      templateScope: local/fail-two-nested-dag-suspend
      type: DAG
    fail-two-nested-dag-suspend-526791725:
      boundaryID: fail-two-nested-dag-suspend-2864264609
      children:
      - fail-two-nested-dag-suspend-1250125036
      displayName: dag2-branch2-step2
      finishedAt: "2022-09-02T14:56:45Z"
      id: fail-two-nested-dag-suspend-526791725
      name: fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step2
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 3
        memory: 3
      startedAt: "2022-09-02T14:56:44Z"
      templateName: approve
      templateScope: local/fail-two-nested-dag-suspend
      type: Suspend
    fail-two-nested-dag-suspend-1199792179:
      boundaryID: fail-two-nested-dag-suspend
      children:
      - fail-two-nested-dag-suspend-1216569798
      displayName: dag1-step1
      finishedAt: "2022-09-02T14:52:14Z"
      hostNodeName: k3d-k3s-default-server-0
      id: fail-two-nested-dag-suspend-1199792179
      name: fail-two-nested-dag-suspend.dag1-step1
      outputs:
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 3
        memory: 3
      startedAt: "2022-09-02T14:52:10Z"
      templateName: gen-random-int-javascript
      templateScope: local/fail-two-nested-dag-suspend
      type: Pod
    fail-two-nested-dag-suspend-1216569798:
      boundaryID: fail-two-nested-dag-suspend
      children:
      - fail-two-nested-dag-suspend-2813931752
      - fail-two-nested-dag-suspend-2864264609
      displayName: dag1-step2
      finishedAt: "2022-09-02T14:52:21Z"
      id: fail-two-nested-dag-suspend-1216569798
      name: fail-two-nested-dag-suspend.dag1-step2
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 15
        memory: 15
      startedAt: "2022-09-02T14:52:20Z"
      templateName: approve
      templateScope: local/fail-two-nested-dag-suspend
      type: Suspend
    fail-two-nested-dag-suspend-1250125036:
      boundaryID: fail-two-nested-dag-suspend
      children:
      - fail-two-nested-dag-suspend-2528852583
      displayName: dag1-step4
      finishedAt: "2022-09-02T14:56:46Z"
      id: fail-two-nested-dag-suspend-1250125036
      name: fail-two-nested-dag-suspend.dag1-step4
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 3
        memory: 3
      startedAt: "2022-09-02T14:56:45Z"
      templateName: approve
      templateScope: local/fail-two-nested-dag-suspend
      type: Suspend
    fail-two-nested-dag-suspend-1841799687:
      boundaryID: fail-two-nested-dag-suspend-2813931752
      children:
      - fail-two-nested-dag-suspend-471194294
      displayName: dag2-branch1-step2
      finishedAt: "2022-09-02T14:53:39Z"
      id: fail-two-nested-dag-suspend-1841799687
      name: fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2
      outboundNodes:
      - fail-two-nested-dag-suspend-437639056
      phase: Succeeded
      progress: 4/5
      resourcesDuration:
        cpu: 9
        memory: 9
      startedAt: "2022-09-02T14:53:18Z"
      templateName: inner-dag-1
      templateScope: local/fail-two-nested-dag-suspend
      type: DAG
    fail-two-nested-dag-suspend-1858577306:
      boundaryID: fail-two-nested-dag-suspend-2813931752
      children:
      - fail-two-nested-dag-suspend-1841799687
      displayName: dag2-branch1-step1
      finishedAt: "2022-09-02T14:52:22Z"
      id: fail-two-nested-dag-suspend-1858577306
      name: fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step1
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 9
        memory: 9
      startedAt: "2022-09-02T14:52:21Z"
      templateName: approve
      templateScope: local/fail-two-nested-dag-suspend
      type: Suspend
    fail-two-nested-dag-suspend-2528852583:
      boundaryID: fail-two-nested-dag-suspend
      displayName: dag1-step5-tofail
      finishedAt: "2022-09-02T14:56:48Z"
      hostNodeName: k3d-k3s-default-server-0
      id: fail-two-nested-dag-suspend-2528852583
      message: Error (exit code 1)
      name: fail-two-nested-dag-suspend.dag1-step5-tofail
      outputs:
        exitCode: "1"
      phase: Failed
      progress: 0/1
      resourcesDuration:
        cpu: 3
        memory: 3
      startedAt: "2022-09-02T14:56:46Z"
      templateName: node-to-fail
      templateScope: local/fail-two-nested-dag-suspend
      type: Pod
    fail-two-nested-dag-suspend-2781431063:
      boundaryID: fail-two-nested-dag-suspend-476458868
      children:
      - fail-two-nested-dag-suspend-2798208682
      displayName: dag3-step1
      finishedAt: "2022-09-02T14:56:24Z"
      id: fail-two-nested-dag-suspend-2781431063
      name: fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step1
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 9
        memory: 9
      startedAt: "2022-09-02T14:56:23Z"
      templateName: approve
      templateScope: local/fail-two-nested-dag-suspend
      type: Suspend
    fail-two-nested-dag-suspend-2798208682:
      boundaryID: fail-two-nested-dag-suspend-476458868
      children:
      - fail-two-nested-dag-suspend-2814986301
      displayName: dag3-step2
      finishedAt: "2022-09-02T14:56:27Z"
      hostNodeName: k3d-k3s-default-server-0
      id: fail-two-nested-dag-suspend-2798208682
      name: fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step2
      outputs:
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 3
        memory: 3
      startedAt: "2022-09-02T14:56:24Z"
      templateName: gen-random-int-javascript
      templateScope: local/fail-two-nested-dag-suspend
      type: Pod
    fail-two-nested-dag-suspend-2813931752:
      boundaryID: fail-two-nested-dag-suspend
      children:
      - fail-two-nested-dag-suspend-1858577306
      displayName: dag1-step3-middle1
      finishedAt: "2022-09-02T14:53:39Z"
      id: fail-two-nested-dag-suspend-2813931752
      name: fail-two-nested-dag-suspend.dag1-step3-middle1
      outboundNodes:
      - fail-two-nested-dag-suspend-437639056
      phase: Succeeded
      progress: 5/6
      resourcesDuration:
        cpu: 9
        memory: 9
      startedAt: "2022-09-02T14:53:18Z"
      templateName: middle-dag1
      templateScope: local/fail-two-nested-dag-suspend
      type: DAG
    fail-two-nested-dag-suspend-2814986301:
      boundaryID: fail-two-nested-dag-suspend-476458868
      children:
      - fail-two-nested-dag-suspend-526791725
      displayName: dag3-step3
      finishedAt: "2022-09-02T14:56:36Z"
      hostNodeName: k3d-k3s-default-server-0
      id: fail-two-nested-dag-suspend-2814986301
      name: fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step3
      outputs:
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 3
        memory: 3
      startedAt: "2022-09-02T14:56:34Z"
      templateName: gen-random-int-javascript
      templateScope: local/fail-two-nested-dag-suspend
      type: Pod
    fail-two-nested-dag-suspend-2864264609:
      boundaryID: fail-two-nested-dag-suspend
      children:
      - fail-two-nested-dag-suspend-476458868
      displayName: dag1-step3-middle2
      finishedAt: "2022-09-02T14:56:45Z"
      id: fail-two-nested-dag-suspend-2864264609
      name: fail-two-nested-dag-suspend.dag1-step3-middle2
      outboundNodes:
      - fail-two-nested-dag-suspend-526791725
      phase: Succeeded
      progress: 5/6
      resourcesDuration:
        cpu: 9
        memory: 9
      startedAt: "2022-09-02T14:56:23Z"
      templateName: middle-dag2
      templateScope: local/fail-two-nested-dag-suspend
      type: DAG
  phase: Failed
  progress: 11/12
  resourcesDuration:
    cpu: 18
    memory: 18
  startedAt: "2022-09-02T14:56:23Z"
`

func TestRetryWorkflowWithNestedDAGsWithSuspendNodes(t *testing.T) {
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)

	// Retry top individual pod node
	wf, podsToDelete, err := FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step1", nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(wf.Status.Nodes))
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, 6, len(podsToDelete))

	// Retry top individual suspend node
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step2", nil)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(wf.Status.Nodes))
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step1").Phase)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step2").Phase)
	assert.Equal(t, 5, len(podsToDelete))

	// Retry the starting on first DAG in one of the branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step3-middle2", nil)
	assert.NoError(t, err)
	assert.Equal(t, 12, len(wf.Status.Nodes))
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step2").Phase)
	// All nodes in the other branch remains succeeded
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step3").Phase)
	// The nodes in the retrying branch are reset
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2").Phase)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1").Phase)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step1").Phase)
	assert.Equal(t, 3, len(podsToDelete))

	// Retry the starting on second DAG in one of the branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1", nil)
	assert.NoError(t, err)
	assert.Equal(t, 12, len(wf.Status.Nodes))
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step2").Phase)
	// All nodes in the other branch remains succeeded
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step3").Phase)
	// The nodes in the retrying branch are reset
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2").Phase)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1").Phase)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step1").Phase)
	assert.Equal(t, 3, len(podsToDelete))

	// Retry the first individual node (suspended node) connecting to the second DAG in one of the branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step1", nil)
	assert.NoError(t, err)
	assert.Equal(t, 12, len(wf.Status.Nodes))
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step2").Phase)
	// All nodes in the other branch remains succeeded
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step3").Phase)
	// The nodes in the retrying branch are reset (parent DAGs are marked as running)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2").Phase)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1").Phase)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step1").Phase)
	assert.Equal(t, 3, len(podsToDelete))

	// Retry the second individual node (pod node) connecting to the second DAG in one of the branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step2", nil)
	assert.NoError(t, err)
	assert.Equal(t, 12, len(wf.Status.Nodes))
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step2").Phase)
	// All nodes in the other branch remains succeeded
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step3").Phase)
	// The nodes in the retrying branch are reset (parent DAGs are marked as running)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2").Phase)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1").Phase)
	// The suspended node remains succeeded
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step1").Phase)
	assert.Equal(t, 3, len(podsToDelete))

	// Retry the third individual node (pod node) connecting to the second DAG in one of the branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step3", nil)
	assert.NoError(t, err)
	assert.Equal(t, 13, len(wf.Status.Nodes))
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step2").Phase)
	// All nodes in the other branch remains succeeded
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step3").Phase)
	// The nodes in the retrying branch are reset (parent DAGs are marked as running)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2").Phase)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1").Phase)
	// The suspended node remains succeeded
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step2").Phase)
	assert.Equal(t, 2, len(podsToDelete))

	// Retry the last individual node (suspend node) connecting to the second DAG in one of the branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step2", nil)
	assert.NoError(t, err)
	assert.Equal(t, 15, len(wf.Status.Nodes))
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step2").Phase)
	// All nodes in the other branch remains succeeded
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step3").Phase)
	// The nodes in the retrying branch are reset (parent DAGs are marked as running)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1").Phase)
	// The suspended node remains succeeded
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step2").Phase)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step2").Phase)
	assert.Equal(t, 1, len(podsToDelete))

	// Retry the node that connects the two branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step4", nil)
	assert.NoError(t, err)
	assert.Equal(t, 16, len(wf.Status.Nodes))
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step2").Phase)
	// All nodes in two branches remains succeeded
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step3").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step2").Phase)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step4").Phase)
	assert.Equal(t, 1, len(podsToDelete))

	// Retry the last node (failing node)
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step5-tofail", nil)
	assert.NoError(t, err)
	assert.Equal(t, 16, len(wf.Status.Nodes))
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step2").Phase)
	// All nodes in two branches remains succeeded
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle1.dag2-branch1-step2.dag3-step3").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step2").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step4").Phase)
	assert.Equal(t, 1, len(podsToDelete))
}
