package util

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	newWf, err := SubmitWorkflow(ctx, nil, wfClientSet, "test-namespace", newWf, nil, &wfv1.SubmitOpts{DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, wf.Spec, newWf.Spec)
	assert.Equal(t, wf.Status, newWf.Status)
}

// TestResubmitWorkflowWithOnExit ensures we do not carry over the onExit node even if successful
func TestResubmitWorkflowWithOnExit(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
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
	wf.Status.Nodes.Set(ctx, onExitID, onExitNode)
	newWF, err := FormulateResubmitWorkflow(ctx, &wf, true, nil)
	require.NoError(t, err)
	newWFOnExitName := newWF.Name + ".onExit"
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
			assert.Len(t, filePaths, len(body))
			require.NoError(t, err)
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
			require.Error(t, err)
			assert.Empty(t, body)
		})
	}
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

		ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		_, err := wfIf.Create(ctx, origWf, metav1.CreateOptions{})
		require.NoError(t, err)

		// will return error as displayName does not match any nodes
		err = ResumeWorkflow(ctx, wfIf, hydratorfake.Noop, "suspend", "displayName=nonexistant")
		require.Error(t, err)

		// displayName didn't match suspend node so should still be running
		wf, err := wfIf.Get(ctx, "suspend", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByDisplayName("approve").Phase)

		err = ResumeWorkflow(ctx, wfIf, hydratorfake.Noop, "suspend", "displayName=approve")
		require.NoError(t, err)

		// displayName matched node so has succeeded
		wf, err = wfIf.Get(ctx, "suspend", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByDisplayName("approve").Phase)
		assert.Empty(t, wf.Status.Nodes.FindByDisplayName("approve").Message)
	})

	t.Run("With user info", func(t *testing.T) {
		wfIf := argofake.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
		origWf := wfv1.MustUnmarshalWorkflow(suspendedWf)

		ctx := context.WithValue(logging.WithLogger(context.TODO(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())), auth.ClaimsKey,
			&types.Claims{Claims: jwt.Claims{Subject: strings.Repeat("x", 63) + "y"}, Email: "my@email", PreferredUsername: "username"})
		uim := creator.UserInfoMap(ctx)

		_, err := wfIf.Create(ctx, origWf, metav1.CreateOptions{})
		require.NoError(t, err)

		// will return error as displayName does not match any nodes
		err = ResumeWorkflow(ctx, wfIf, hydratorfake.Noop, "suspend", "displayName=nonexistant")
		require.Error(t, err)

		// displayName didn't match suspend node so should still be running
		wf, err := wfIf.Get(ctx, "suspend", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByDisplayName("approve").Phase)

		err = ResumeWorkflow(ctx, wfIf, hydratorfake.Noop, "suspend", "displayName=approve")
		require.NoError(t, err)

		// displayName matched node so has succeeded
		wf, err = wfIf.Get(ctx, "suspend", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByDisplayName("approve").Phase)
		assert.Equal(t, fmt.Sprintf("Resumed by: %v", uim), wf.Status.Nodes.FindByDisplayName("approve").Message)
	})
}

func TestStopWorkflowByNodeName(t *testing.T) {
	wfIf := argofake.NewSimpleClientset().ArgoprojV1alpha1().Workflows("")
	origWf := wfv1.MustUnmarshalWorkflow(suspendedWf)

	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	_, err := wfIf.Create(ctx, origWf, metav1.CreateOptions{})
	require.NoError(t, err)

	// will return error as displayName does not match any nodes
	err = StopWorkflow(ctx, wfIf, hydratorfake.Noop, "suspend", "displayName=nonexistant", "error occurred")
	require.Error(t, err)

	// displayName didn't match suspend node so should still be running
	wf, err := wfIf.Get(ctx, "suspend", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByDisplayName("approve").Phase)

	err = StopWorkflow(ctx, wfIf, hydratorfake.Noop, "suspend", "displayName=approve", "error occurred")
	require.NoError(t, err)

	// displayName matched node so has succeeded
	wf, err = wfIf.Get(ctx, "suspend", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, wf.Status.Nodes.FindByDisplayName("approve").Phase)

	origWf.Status = wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded}
	origWf.Name = "succeeded-wf"
	_, err = wfIf.Create(ctx, origWf, metav1.CreateOptions{})
	require.NoError(t, err)
	err = StopWorkflow(ctx, wfIf, hydratorfake.Noop, "succeeded-wf", "", "")
	require.EqualError(t, err, "cannot shutdown a completed workflow: workflow: \"succeeded-wf\", namespace: \"\"")
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
	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	p := AddParamToGlobalScope(ctx, &wf, nil, wfv1.Parameter{
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

	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	_, err := wfIf.Create(ctx, origWf, metav1.CreateOptions{})
	require.NoError(t, err)
	err = updateSuspendedNode(ctx, wfIf, hydratorfake.Noop, "does-not-exist", "displayName=approve", SetOperationValues{OutputParameters: map[string]string{"message": "Hello World"}}, creator.ActionNone)
	require.EqualError(t, err, "workflows.argoproj.io \"does-not-exist\" not found")
	err = updateSuspendedNode(ctx, wfIf, hydratorfake.Noop, "suspend-template", "displayName=does-not-exists", SetOperationValues{OutputParameters: map[string]string{"message": "Hello World"}}, creator.ActionNone)
	require.EqualError(t, err, "currently, set only targets suspend nodes: no suspend nodes matching nodeFieldSelector: displayName=does-not-exists")
	err = updateSuspendedNode(ctx, wfIf, hydratorfake.Noop, "suspend-template", "displayName=approve", SetOperationValues{OutputParameters: map[string]string{"does-not-exist": "Hello World"}}, creator.ActionNone)
	require.EqualError(t, err, "node is not expecting output parameter 'does-not-exist'")
	err = updateSuspendedNode(ctx, wfIf, hydratorfake.Noop, "suspend-template", "displayName=approve", SetOperationValues{OutputParameters: map[string]string{"message": "Hello World"}}, creator.ActionNone)
	require.NoError(t, err)
	err = updateSuspendedNode(ctx, wfIf, hydratorfake.Noop, "suspend-template", "name=suspend-template-kgfn7[0].approve", SetOperationValues{OutputParameters: map[string]string{"message2": "Hello World 2"}}, creator.ActionNone)
	require.NoError(t, err)

	// make sure global variable was updated
	wf, err := wfIf.Get(ctx, "suspend-template", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "Hello World 2", wf.Status.Outputs.Parameters[0].Value.String())

	noSpaceWf := wfv1.MustUnmarshalWorkflow(susWorkflow)
	noSpaceWf.Name = "suspend-template-no-outputs"
	node := noSpaceWf.Status.Nodes["suspend-template-kgfn7-2667278707"]
	node.Outputs = nil
	noSpaceWf.Status.Nodes["suspend-template-kgfn7-2667278707"] = node
	_, err = wfIf.Create(ctx, noSpaceWf, metav1.CreateOptions{})
	require.NoError(t, err)
	err = updateSuspendedNode(ctx, wfIf, hydratorfake.Noop, "suspend-template-no-outputs", "displayName=approve", SetOperationValues{OutputParameters: map[string]string{"message": "Hello World"}}, creator.ActionNone)
	require.EqualError(t, err, "cannot set output parameters because node is not expecting any raw parameters")
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
			require.NoError(t, err)
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
		require.NoError(t, ApplySubmitOpts(&wfv1.Workflow{}, nil))
	})
	t.Run("InvalidLabels", func(t *testing.T) {
		require.Error(t, ApplySubmitOpts(&wfv1.Workflow{}, &wfv1.SubmitOpts{Labels: "a"}))
	})
	t.Run("Labels", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		err := ApplySubmitOpts(wf, &wfv1.SubmitOpts{Labels: "a=1,b=1"})
		require.NoError(t, err)
		assert.Len(t, wf.GetLabels(), 2)
	})
	t.Run("MergeLabels", func(t *testing.T) {
		wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"a": "0", "b": "0"}}}
		err := ApplySubmitOpts(wf, &wfv1.SubmitOpts{Labels: "a=1"})
		require.NoError(t, err)
		require.Len(t, wf.GetLabels(), 2)
		assert.Equal(t, "1", wf.GetLabels()["a"])
		assert.Equal(t, "0", wf.GetLabels()["b"])
	})
	t.Run("InvalidParameters", func(t *testing.T) {
		require.Error(t, ApplySubmitOpts(&wfv1.Workflow{}, &wfv1.SubmitOpts{Parameters: []string{"a"}}))
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
		require.NoError(t, err)
		parameters := wf.Spec.Arguments.Parameters
		require.Len(t, parameters, 1)
		assert.Equal(t, "a", parameters[0].Name)
		assert.Equal(t, "81861780812", parameters[0].Value.String())
	})
	t.Run("PodPriorityClassName", func(t *testing.T) {
		wf := &wfv1.Workflow{}
		err := ApplySubmitOpts(wf, &wfv1.SubmitOpts{PodPriorityClassName: "abc"})
		require.NoError(t, err)
		assert.Equal(t, "abc", wf.Spec.PodPriorityClassName)
	})
}

func TestReadParametersFile(t *testing.T) {
	file, err := os.CreateTemp("", "")
	require.NoError(t, err)
	defer func() { _ = os.Remove(file.Name()) }()
	err = os.WriteFile(file.Name(), []byte(`a: 81861780812`), 0o600)
	require.NoError(t, err)
	opts := &wfv1.SubmitOpts{}
	err = ReadParametersFile(file.Name(), opts)
	require.NoError(t, err)
	parameters := opts.Parameters
	require.Len(t, parameters, 1)
	assert.Equal(t, "a=81861780812", parameters[0])
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
		wf, err := FormulateResubmitWorkflow(func() context.Context {
			ctx := context.Background()
			return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		}(), wf, false, nil)
		require.NoError(t, err)
		assert.Contains(t, wf.GetLabels(), common.LabelKeyControllerInstanceID)
		assert.Contains(t, wf.GetLabels(), common.LabelKeyClusterWorkflowTemplate)
		assert.Contains(t, wf.GetLabels(), common.LabelKeyCronWorkflow)
		assert.Contains(t, wf.GetLabels(), common.LabelKeyWorkflowTemplate)
		assert.NotContains(t, wf.GetLabels(), common.LabelKeyCreator)
		assert.NotContains(t, wf.GetLabels(), common.LabelKeyPhase)
		assert.NotContains(t, wf.GetLabels(), common.LabelKeyCompleted)
		assert.NotContains(t, wf.GetLabels(), common.LabelKeyWorkflowArchivingStatus)
		assert.Contains(t, wf.GetLabels(), common.LabelKeyPreviousWorkflowName)
		assert.Len(t, wf.OwnerReferences, 1)
		assert.Equal(t, "test", wf.OwnerReferences[0].APIVersion)
		assert.Equal(t, "testObj", wf.OwnerReferences[0].Name)
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
		ctx := context.WithValue(func() context.Context {
			ctx := context.Background()
			return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		}(), auth.ClaimsKey, &types.Claims{
			Claims:            jwt.Claims{Subject: "yyyy-yyyy-yyyy-yyyy"},
			Email:             "bar.at.example.com",
			PreferredUsername: "bar",
		})
		wf, err := FormulateResubmitWorkflow(ctx, wf, false, nil)
		require.NoError(t, err)
		assert.Equal(t, "yyyy-yyyy-yyyy-yyyy", wf.Labels[common.LabelKeyCreator])
		assert.Equal(t, "bar.at.example.com", wf.Labels[common.LabelKeyCreatorEmail])
		assert.Equal(t, "bar", wf.Labels[common.LabelKeyCreatorPreferredUsername])
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
		wf, err := FormulateResubmitWorkflow(func() context.Context {
			ctx := context.Background()
			return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		}(), wf, false, nil)
		require.NoError(t, err)
		assert.Emptyf(t, wf.Labels[common.LabelKeyCreator], "should not %s label when a workflow is resubmitted by an unauthenticated request", common.LabelKeyCreator)
		assert.Emptyf(t, wf.Labels[common.LabelKeyCreatorEmail], "should not %s label when a workflow is resubmitted by an unauthenticated request", common.LabelKeyCreatorEmail)
		assert.Emptyf(t, wf.Labels[common.LabelKeyCreatorPreferredUsername], "should not %s label when a workflow is resubmitted by an unauthenticated request", common.LabelKeyCreatorPreferredUsername)

	})
	t.Run("OverrideParams", func(t *testing.T) {
		wf := &wfv1.Workflow{
			Spec: wfv1.WorkflowSpec{Arguments: wfv1.Arguments{
				Parameters: []wfv1.Parameter{
					{Name: "message", Value: wfv1.AnyStringPtr("default")},
				},
			}},
		}
		wf, err := FormulateResubmitWorkflow(func() context.Context {
			ctx := context.Background()
			return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		}(), wf, false, []string{"message=modified"})
		require.NoError(t, err)
		assert.Equal(t, "modified", wf.Spec.Arguments.Parameters[0].Value.String())
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

	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	wf, err := wfIf.Create(ctx, origWf, metav1.CreateOptions{})
	require.NoError(t, err)
	newWf, _, err := FormulateRetryWorkflow(ctx, wf, false, "", nil)
	require.NoError(t, err)
	newWfBytes, err := yaml.Marshal(newWf)
	require.NoError(t, err)
	assert.NotContains(t, string(newWfBytes), "steps-9fkqc-3224593506")
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

	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	wf, err := wfIf.Create(ctx, origWf, metav1.CreateOptions{})
	require.NoError(t, err)
	newWf, _, err := FormulateRetryWorkflow(ctx, wf, false, "", nil)
	require.NoError(t, err)
	newWfBytes, err := yaml.Marshal(newWf)
	require.NoError(t, err)
	t.Log(string(newWfBytes))
	assert.NotContains(t, string(newWfBytes), "retry-script-6xt68-3924170365")
}

func TestFormulateRetryWorkflow(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	wfClient := argofake.NewSimpleClientset().ArgoprojV1alpha1().Workflows("my-ns")
	t.Run("DAG", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "my-dag",
				Labels: map[string]string{},
			},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowFailed,
				Nodes: map[string]wfv1.NodeStatus{
					"my-dag": {Phase: wfv1.NodeFailed, Type: wfv1.NodeTypeDAG, Name: "my-dag", ID: "my-dag"}},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		wf, _, err = FormulateRetryWorkflow(ctx, wf, false, "", nil)
		require.NoError(t, err)
		assert.Len(t, wf.Status.Nodes, 1)
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
					"wf-with-skipped-and-suspended-nodes": {ID: "wf-with-skipped-and-suspended-nodes", Name: "wf-with-skipped-and-suspended-nodes", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeDAG, Children: []string{"suspended", "skipped"}},
					"suspended": {
						ID:         "suspended",
						Phase:      wfv1.NodeSucceeded,
						Type:       wfv1.NodeTypeSuspend,
						BoundaryID: "wf-with-skipped-and-suspended-nodes",
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
		require.NoError(t, err)
		wf, _, err = FormulateRetryWorkflow(ctx, wf, true, "id=suspended", nil)
		require.NoError(t, err)
		require.Len(t, wf.Status.Nodes, 2)
		assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["wf-with-skipped-and-suspended-nodes"].Phase)
		assert.Equal(t, wfv1.NodeSkipped, wf.Status.Nodes["skipped"].Phase)
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
					"my-nested-dag-1": {ID: "my-nested-dag-1", Name: "my-nested-dag-1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeDAG, Children: []string{"1"}},
					"1":               {ID: "1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, BoundaryID: "my-nested-dag-1", Children: []string{"2", "4"}},
					"2":               {ID: "2", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, BoundaryID: "1", Children: []string{"3"}},
					"3":               {ID: "3", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "2"},
					"4":               {ID: "4", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod, BoundaryID: "1"}},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		wf, _, err = FormulateRetryWorkflow(ctx, wf, true, "id=3", nil)
		require.NoError(t, err)
		// Node #3, #4 are deleted and will be recreated so only 3 nodes left in wf.Status.Nodes
		require.Len(t, wf.Status.Nodes, 3)
		assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["my-nested-dag-1"].Phase)
		// The parent group nodes should be running.
		assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["1"].Phase)
		assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["2"].Phase)
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
					"my-nested-dag-2": {ID: "my-nested-dag-2", Name: "my-nested-dag-2", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeDAG, Children: []string{"1"}},
					"1":               {ID: "1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, BoundaryID: "my-nested-dag-2", Children: []string{"2", "4"}},
					"2":               {ID: "2", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "1", Children: []string{"3"}},
					"3":               {ID: "3", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "2"},
					"4":               {ID: "4", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod, BoundaryID: "1"}},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		wf, _, err = FormulateRetryWorkflow(ctx, wf, true, "", nil)
		require.NoError(t, err)
		// Node #2, #3, and #4 are deleted and will be recreated so only 2 nodes left in wf.Status.Nodes
		require.Len(t, wf.Status.Nodes, 4)
		assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["my-nested-dag-2"].Phase)
		assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["1"].Phase)
		assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["2"].Phase)
		assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["3"].Phase)
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
					"override-param-wf": {ID: "override-param-wf", Name: "override-param-wf", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeDAG},
				}},
		}
		wf, _, err := FormulateRetryWorkflow(func() context.Context {
			ctx := context.Background()
			return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		}(), wf, false, "", []string{"message=modified"})
		require.NoError(t, err)
		assert.Equal(t, "modified", wf.Spec.Arguments.Parameters[0].Value.String())

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
					"override-param-wf": {ID: "override-param-wf", Name: "override-param-wf", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup},
				},
				StoredWorkflowSpec: &wfv1.WorkflowSpec{Arguments: wfv1.Arguments{
					Parameters: []wfv1.Parameter{
						{Name: "message", Value: wfv1.AnyStringPtr("default")},
					}},
				}},
		}
		wf, _, err := FormulateRetryWorkflow(func() context.Context {
			ctx := context.Background()
			return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		}(), wf, false, "", []string{"message=modified"})
		require.NoError(t, err)
		assert.Equal(t, "modified", wf.Spec.Arguments.Parameters[0].Value.String())
		assert.Equal(t, "modified", wf.Status.StoredWorkflowSpec.Arguments.Parameters[0].Value.String())

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
		require.NoError(t, err)
		_, _, err = FormulateRetryWorkflow(ctx, wf, false, "", nil)
		require.Error(t, err)
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
		require.NoError(t, err)
		_, _, err = FormulateRetryWorkflow(ctx, wf, false, "", nil)
		require.Error(t, err)
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
		require.NoError(t, err)
		_, _, err = FormulateRetryWorkflow(ctx, wf, false, "", nil)
		require.Error(t, err)
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
					"successful-workflow-2": {ID: "successful-workflow-2", Name: "successful-workflow-2", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeDAG, Children: []string{"1"}},
					"1":                     {ID: "1", Name: "1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, BoundaryID: "successful-workflow-2", Children: []string{"2", "4"}},
					"2":                     {ID: "2", Name: "2", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeTaskGroup, BoundaryID: "1", Children: []string{"3"}},
					"3":                     {ID: "3", Name: "3", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "2"},
					"4":                     {ID: "4", Name: "4", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "1"}},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		wf, _, err = FormulateRetryWorkflow(ctx, wf, true, "id=4", nil)
		require.NoError(t, err)
		// Node #4 is deleted and will be recreated so only 4 nodes left in wf.Status.Nodes
		require.Len(t, wf.Status.Nodes, 4)
		assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["successful-workflow-2"].Phase)
		// The parent group nodes should be running.
		assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["1"].Phase)
		assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["2"].Phase)
		assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["3"].Phase)
	})

	t.Run("Retry continue on failed workflow", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "continue-on-failed-workflow",
				Labels: map[string]string{},
			},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowFailed,
				Nodes: map[string]wfv1.NodeStatus{
					"continue-on-failed-workflow": {ID: "continue-on-failed-workflow", Name: "continue-on-failed-workflow", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypeDAG, Children: []string{"1"}, OutboundNodes: []string{"3", "5"}},
					"1":                           {ID: "1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "continue-on-failed-workflow", Children: []string{"2", "4"}, Name: "node1"},
					"2":                           {ID: "2", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod, BoundaryID: "continue-on-failed-workflow", Children: []string{"3"}, Name: "node2"},
					"3":                           {ID: "3", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "continue-on-failed-workflow", Name: "node3"},
					"4":                           {ID: "4", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod, BoundaryID: "continue-on-failed-workflow", Children: []string{"5"}, Name: "node4"},
					"5":                           {ID: "5", Phase: wfv1.NodeOmitted, Type: wfv1.NodeTypeSkipped, BoundaryID: "continue-on-failed-workflow", Name: "node5"}},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		wf, podsToDelete, err := FormulateRetryWorkflow(ctx, wf, false, "", nil)
		require.NoError(t, err)
		require.Len(t, wf.Status.Nodes, 4)
		assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["1"].Phase)
		assert.Len(t, podsToDelete, 1)
	})

	t.Run("Retry continue on failed workflow with restartSuccessful and nodeFieldSelector", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "continue-on-failed-workflow-2",
				Labels: map[string]string{},
			},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowFailed,
				Nodes: map[string]wfv1.NodeStatus{
					"continue-on-failed-workflow-2": {ID: "continue-on-failed-workflow-2", Name: "continue-on-failed-workflow-2", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypeDAG, Children: []string{"1"}, OutboundNodes: []string{"3", "5"}},
					"1":                             {ID: "1", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "continue-on-failed-workflow-2", Children: []string{"2", "4"}, Name: "node1"},
					"2":                             {ID: "2", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod, BoundaryID: "continue-on-failed-workflow-2", Children: []string{"3"}, Name: "node2"},
					"3":                             {ID: "3", Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypePod, BoundaryID: "continue-on-failed-workflow-2", Name: "node3"},
					"4":                             {ID: "4", Phase: wfv1.NodeFailed, Type: wfv1.NodeTypePod, BoundaryID: "continue-on-failed-workflow-2", Children: []string{"5"}, Name: "node4"},
					"5":                             {ID: "5", Phase: wfv1.NodeOmitted, Type: wfv1.NodeTypeSkipped, BoundaryID: "continue-on-failed-workflow-2", Name: "node5"}},
			},
		}
		_, err := wfClient.Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		wf, podsToDelete, err := FormulateRetryWorkflow(ctx, wf, true, "id=3", nil)
		require.NoError(t, err)
		require.Len(t, wf.Status.Nodes, 2)
		assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes["1"].Phase)
		assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["continue-on-failed-workflow-2"].Phase)
		assert.Len(t, podsToDelete, 3)
	})
}

func TestFromUnstructuredObj(t *testing.T) {
	un := &unstructured.Unstructured{}
	wfv1.MustUnmarshal([]byte(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: example-integers
spec:
  schedules:
    - "* * * * *"
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
	require.NoError(t, err)
}

func TestToUnstructured(t *testing.T) {
	un, err := ToUnstructured(&wfv1.Workflow{})
	require.NoError(t, err)
	gv := un.GetObjectKind().GroupVersionKind()
	assert.Equal(t, workflow.WorkflowKind, gv.Kind)
	assert.Equal(t, workflow.Group, gv.Group)
	assert.Equal(t, workflow.Version, gv.Version)
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
	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	wf := wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)

	// Retry top individual pod node
	wf, podsToDelete, err := FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step1", nil)
	require.NoError(t, err)
	assert.Len(t, wf.Status.Nodes, 1)
	assert.Len(t, podsToDelete, 6)

	// Retry top individual suspend node
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step2", nil)
	require.NoError(t, err)
	require.Len(t, wf.Status.Nodes, 2)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step1").Phase)
	assert.Len(t, podsToDelete, 5)

	// Retry the starting on first DAG in one of the branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step3-middle2", nil)
	require.NoError(t, err)

	assert.Len(t, wf.Status.Nodes, 9)
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
	assert.Len(t, podsToDelete, 3)

	// Retry the starting on second DAG in one of the branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1", nil)
	require.NoError(t, err)
	assert.Len(t, wf.Status.Nodes, 10)
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
	assert.Len(t, podsToDelete, 3)

	// Retry the first individual node (suspended node) connecting to the second DAG in one of the branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step1", nil)
	require.NoError(t, err)
	assert.Len(t, wf.Status.Nodes, 11)
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
	assert.Len(t, podsToDelete, 3)

	// Retry the second individual node (pod node) connecting to the second DAG in one of the branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step2", nil)
	require.NoError(t, err)
	assert.Len(t, wf.Status.Nodes, 12)
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
	assert.Len(t, podsToDelete, 3)

	// Retry the third individual node (pod node) connecting to the second DAG in one of the branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step1.dag3-step3", nil)
	require.NoError(t, err)
	assert.Len(t, wf.Status.Nodes, 13)
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
	assert.Len(t, podsToDelete, 2)

	// Retry the last individual node (suspend node) connecting to the second DAG in one of the branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step3-middle2.dag2-branch2-step2", nil)
	require.NoError(t, err)
	assert.Len(t, wf.Status.Nodes, 14)
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
	assert.Len(t, podsToDelete, 1)

	// Retry the node that connects the two branches
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step4", nil)
	require.NoError(t, err)
	assert.Len(t, wf.Status.Nodes, 15)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step2").Phase)
	assert.Len(t, podsToDelete, 1)

	// Retry the last node (failing node)
	wf = wfv1.MustUnmarshalWorkflow(retryWorkflowWithNestedDAGsWithSuspendNodes)
	wf, podsToDelete, err = FormulateRetryWorkflow(ctx, wf, true, "name=fail-two-nested-dag-suspend.dag1-step5-tofail", nil)
	require.NoError(t, err)
	assert.Len(t, wf.Status.Nodes, 16)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Nodes["fail-two-nested-dag-suspend"].Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step1").Phase)
	assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("fail-two-nested-dag-suspend.dag1-step2").Phase)
	assert.Len(t, podsToDelete, 1)
}

const stepsRetryFormulate = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  annotations:
    workflows.argoproj.io/pod-name-format: v2
  creationTimestamp: "2024-09-19T02:41:51Z"
  generateName: steps-
  generation: 29
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
  name: steps-4k5vn
  namespace: argo
  resourceVersion: "50080"
  uid: 0e7608c7-4555-46a4-8697-3be04eec428b
spec:
  activeDeadlineSeconds: 300
  arguments: {}
  entrypoint: hello-hello-hello
  podSpecPatch: |
    terminationGracePeriodSeconds: 3
  templates:
  - inputs: {}
    metadata: {}
    name: hello-hello-hello
    outputs: {}
    steps:
    - - arguments:
          parameters:
          - name: message
            value: hello1
        name: hello1
        template: whalesay
    - - arguments:
          parameters:
          - name: message
            value: hello2a
        name: hello2a
        template: whalesay
      - arguments:
          parameters:
          - name: message
            value: hello2b
        name: hello2b
        template: whalesay
  - container:
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
  artifactGCStatus:
    notSpecified: true
  artifactRepositoryRef:
    artifactRepository:
      archiveLogs: true
      s3:
        accessKeySecret:
          key: accesskey
          name: my-minio-cred
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        secretKeySecret:
          key: secretkey
          name: my-minio-cred
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2024-09-19T02:43:44Z"
  nodes:
    steps-4k5vn:
      children:
      - steps-4k5vn-899690889
      displayName: steps-4k5vn
      finishedAt: "2024-09-19T02:43:44Z"
      id: steps-4k5vn
      name: steps-4k5vn
      outboundNodes:
      - steps-4k5vn-2627784879
      - steps-4k5vn-2644562498
      phase: Succeeded
      progress: 3/3
      resourcesDuration:
        cpu: 1
        memory: 22
      startedAt: "2024-09-19T02:43:30Z"
      templateName: hello-hello-hello
      templateScope: local/steps-4k5vn
      type: Steps
    steps-4k5vn-899690889:
      boundaryID: steps-4k5vn
      children:
      - steps-4k5vn-1044844302
      displayName: '[0]'
      finishedAt: "2024-09-19T02:42:12Z"
      id: steps-4k5vn-899690889
      name: steps-4k5vn[0]
      nodeFlag: {}
      phase: Succeeded
      progress: 3/3
      resourcesDuration:
        cpu: 1
        memory: 22
      startedAt: "2024-09-19T02:41:51Z"
      templateScope: local/steps-4k5vn
      type: StepGroup
    steps-4k5vn-1044844302:
      boundaryID: steps-4k5vn
      children:
      - steps-4k5vn-4053927188
      displayName: hello1
      finishedAt: "2024-09-19T02:42:09Z"
      hostNodeName: k3d-k3s-default-server-0
      id: steps-4k5vn-1044844302
      inputs:
        parameters:
        - name: message
          value: hello1
      name: steps-4k5vn[0].hello1
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: steps-4k5vn/steps-4k5vn-whalesay-1044844302/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 1
        memory: 11
      startedAt: "2024-09-19T02:41:51Z"
      templateName: whalesay
      templateScope: local/steps-4k5vn
      type: Pod
    steps-4k5vn-2627784879:
      boundaryID: steps-4k5vn
      displayName: hello2a
      finishedAt: "2024-09-19T02:43:39Z"
      hostNodeName: k3d-k3s-default-server-0
      id: steps-4k5vn-2627784879
      inputs:
        parameters:
        - name: message
          value: hello2a
      name: steps-4k5vn[1].hello2a
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: steps-4k5vn/steps-4k5vn-whalesay-2627784879/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 5
      startedAt: "2024-09-19T02:43:30Z"
      templateName: whalesay
      templateScope: local/steps-4k5vn
      type: Pod
    steps-4k5vn-2644562498:
      boundaryID: steps-4k5vn
      displayName: hello2b
      finishedAt: "2024-09-19T02:43:41Z"
      hostNodeName: k3d-k3s-default-server-0
      id: steps-4k5vn-2644562498
      inputs:
        parameters:
        - name: message
          value: hello2b
      name: steps-4k5vn[1].hello2b
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: steps-4k5vn/steps-4k5vn-whalesay-2644562498/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 6
      startedAt: "2024-09-19T02:43:30Z"
      templateName: whalesay
      templateScope: local/steps-4k5vn
      type: Pod
    steps-4k5vn-4053927188:
      boundaryID: steps-4k5vn
      children:
      - steps-4k5vn-2627784879
      - steps-4k5vn-2644562498
      displayName: '[1]'
      finishedAt: "2024-09-19T02:43:44Z"
      id: steps-4k5vn-4053927188
      name: steps-4k5vn[1]
      nodeFlag: {}
      phase: Succeeded
      progress: 2/2
      resourcesDuration:
        cpu: 0
        memory: 11
      startedAt: "2024-09-19T02:43:30Z"
      templateScope: local/steps-4k5vn
      type: StepGroup
  phase: Succeeded
  progress: 3/3
  resourcesDuration:
    cpu: 1
    memory: 22
  startedAt: "2024-09-19T02:43:30Z"
  taskResultsCompletionStatus:
    steps-4k5vn-1044844302: true
    steps-4k5vn-2627784879: true
    steps-4k5vn-2644562498: true

`

func TestStepsRetryWorkflow(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	wf := wfv1.MustUnmarshalWorkflow(stepsRetryFormulate)
	selectorStr := "id=steps-4k5vn-2627784879"

	running := map[string]bool{
		"steps-4k5vn-4053927188": true,
		"steps-4k5vn":            true,
	}

	deleted := map[string]bool{
		"steps-4k5vn-2627784879": true,
	}

	succeeded := make(map[string]bool)

	for _, node := range wf.Status.Nodes {
		_, inRunning := running[node.ID]
		_, inDeleted := deleted[node.ID]
		if !inRunning && !inDeleted {
			succeeded[node.ID] = true
		}
	}
	newWf, podsToDelete, err := FormulateRetryWorkflow(func() context.Context {
		ctx := context.Background()
		return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	}(), wf, true, selectorStr, []string{})
	require.NoError(err)
	assert.Len(podsToDelete, 1)
	assert.Len(newWf.Status.Nodes, 5)

	for _, node := range newWf.Status.Nodes {
		if _, ok := running[node.ID]; ok {
			assert.Equal(wfv1.NodeRunning, node.Phase)
		}
		if _, ok := succeeded[node.ID]; ok {
			assert.Equal(wfv1.NodeSucceeded, node.Phase)
		}
	}

}

func TestDagConversion(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	wf := wfv1.MustUnmarshalWorkflow(stepsRetryFormulate)

	nodes, err := newWorkflowsDag(wf)
	require.NoError(err)
	assert.Len(nodes, len(wf.Status.Nodes))

	numNilParent := 0
	for _, n := range nodes {
		if n.parent == nil {
			numNilParent++
		}
	}
	assert.Equal(1, numNilParent)
}

const dagDiamondRetry = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  annotations:
    workflows.argoproj.io/pod-name-format: v2
  creationTimestamp: "2024-10-01T04:27:23Z"
  generateName: dag-diamond-
  generation: 16
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
  name: dag-diamond-82q7s
  namespace: argo
  resourceVersion: "4633"
  uid: dd3d2674-43d8-446a-afdf-17ec95afade2
spec:
  activeDeadlineSeconds: 300
  arguments: {}
  entrypoint: diamond
  podSpecPatch: |
    terminationGracePeriodSeconds: 3
  templates:
  - dag:
      tasks:
      - arguments:
          parameters:
          - name: message
            value: A
        name: A
        template: echo
      - arguments:
          parameters:
          - name: message
            value: B
        depends: A
        name: B
        template: echo
      - arguments:
          parameters:
          - name: message
            value: C
        depends: A
        name: C
        template: echo
      - arguments:
          parameters:
          - name: message
            value: D
        depends: B && C
        name: D
        template: echo
    inputs: {}
    metadata: {}
    name: diamond
    outputs: {}
  - container:
      command:
      - echo
      - '{{inputs.parameters.message}}'
      image: alpine:3.7
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: message
    metadata: {}
    name: echo
    outputs: {}
status:
  artifactGCStatus:
    notSpecified: true
  artifactRepositoryRef:
    artifactRepository:
      archiveLogs: true
      s3:
        accessKeySecret:
          key: accesskey
          name: my-minio-cred
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        secretKeySecret:
          key: secretkey
          name: my-minio-cred
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2024-10-01T04:27:42Z"
  nodes:
    dag-diamond-82q7s:
      children:
      - dag-diamond-82q7s-1310542453
      displayName: dag-diamond-82q7s
      finishedAt: "2024-10-01T04:27:42Z"
      id: dag-diamond-82q7s
      name: dag-diamond-82q7s
      outboundNodes:
      - dag-diamond-82q7s-1226654358
      phase: Succeeded
      progress: 4/4
      resourcesDuration:
        cpu: 0
        memory: 8
      startedAt: "2024-10-01T04:27:23Z"
      templateName: diamond
      templateScope: local/dag-diamond-82q7s
      type: DAG
    dag-diamond-82q7s-1226654358:
      boundaryID: dag-diamond-82q7s
      displayName: D
      finishedAt: "2024-10-01T04:27:39Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-diamond-82q7s-1226654358
      inputs:
        parameters:
        - name: message
          value: D
      name: dag-diamond-82q7s.D
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-diamond-82q7s/dag-diamond-82q7s-echo-1226654358/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-01T04:27:36Z"
      templateName: echo
      templateScope: local/dag-diamond-82q7s
      type: Pod
    dag-diamond-82q7s-1260209596:
      boundaryID: dag-diamond-82q7s
      children:
      - dag-diamond-82q7s-1226654358
      displayName: B
      finishedAt: "2024-10-01T04:27:33Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-diamond-82q7s-1260209596
      inputs:
        parameters:
        - name: message
          value: B
      name: dag-diamond-82q7s.B
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-diamond-82q7s/dag-diamond-82q7s-echo-1260209596/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-01T04:27:30Z"
      templateName: echo
      templateScope: local/dag-diamond-82q7s
      type: Pod
    dag-diamond-82q7s-1276987215:
      boundaryID: dag-diamond-82q7s
      children:
      - dag-diamond-82q7s-1226654358
      displayName: C
      finishedAt: "2024-10-01T04:27:33Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-diamond-82q7s-1276987215
      inputs:
        parameters:
        - name: message
          value: C
      name: dag-diamond-82q7s.C
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-diamond-82q7s/dag-diamond-82q7s-echo-1276987215/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-01T04:27:30Z"
      templateName: echo
      templateScope: local/dag-diamond-82q7s
      type: Pod
    dag-diamond-82q7s-1310542453:
      boundaryID: dag-diamond-82q7s
      children:
      - dag-diamond-82q7s-1260209596
      - dag-diamond-82q7s-1276987215
      displayName: A
      finishedAt: "2024-10-01T04:27:27Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-diamond-82q7s-1310542453
      inputs:
        parameters:
        - name: message
          value: A
      name: dag-diamond-82q7s.A
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-diamond-82q7s/dag-diamond-82q7s-echo-1310542453/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-01T04:27:23Z"
      templateName: echo
      templateScope: local/dag-diamond-82q7s
      type: Pod
  phase: Succeeded
  progress: 4/4
  resourcesDuration:
    cpu: 0
    memory: 8
  startedAt: "2024-10-01T04:27:23Z"
  taskResultsCompletionStatus:
    dag-diamond-82q7s-1226654358: true
    dag-diamond-82q7s-1260209596: true
    dag-diamond-82q7s-1276987215: true
    dag-diamond-82q7s-1310542453: true

`

func TestDAGDiamondRetryWorkflow(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	wf := wfv1.MustUnmarshalWorkflow(dagDiamondRetry)
	selectorStr := "id=dag-diamond-82q7s-1260209596"

	running := map[string]bool{
		"dag-diamond-82q7s": true,
	}

	deleted := map[string]bool{
		"dag-diamond-82q7s-1226654358": true,
	}

	succeeded := make(map[string]bool)

	for _, node := range wf.Status.Nodes {
		_, inRunning := running[node.ID]
		_, inDeleted := deleted[node.ID]
		if !inRunning && !inDeleted {
			succeeded[node.ID] = true
		}
	}
	newWf, podsToDelete, err := FormulateRetryWorkflow(func() context.Context {
		ctx := context.Background()
		return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	}(), wf, true, selectorStr, []string{})

	require.NoError(err)
	assert.Len(podsToDelete, 2)
	assert.Len(newWf.Status.Nodes, 3)

	for _, node := range newWf.Status.Nodes {
		if _, ok := running[node.ID]; ok {
			assert.Equal(wfv1.NodeRunning, node.Phase)
		}
		if _, ok := succeeded[node.ID]; ok {
			assert.Equal(wfv1.NodeSucceeded, node.Phase)
		}
	}
}

const onExitWorkflowRetry = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  annotations:
    workflows.argoproj.io/pod-name-format: v2
  creationTimestamp: "2024-10-02T05:54:00Z"
  generateName: work-avoidance-
  generation: 25
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
    workflows.argoproj.io/resubmitted-from-workflow: work-avoidance-xghlj
  name: work-avoidance-trkkq
  namespace: argo
  resourceVersion: "2661"
  uid: 0271624e-0096-428a-81da-643dbbd69440
spec:
  activeDeadlineSeconds: 300
  arguments: {}
  entrypoint: main
  onExit: save-markers
  podSpecPatch: |
    terminationGracePeriodSeconds: 3
  templates:
  - inputs: {}
    metadata: {}
    name: main
    outputs: {}
    steps:
    - - arguments: {}
        name: load-markers
        template: load-markers
    - - arguments:
          parameters:
          - name: num
            value: '{{item}}'
        name: echo
        template: echo
        withSequence:
          count: "3"
  - container:
      command:
      - mkdir
      - -p
      - /work/markers
      image: docker/whalesay:latest
      name: ""
      resources: {}
      volumeMounts:
      - mountPath: /work
        name: work
    inputs:
      artifacts:
      - name: markers
        optional: true
        path: /work/markers
        s3:
          accessKeySecret:
            key: accesskey
            name: my-minio-cred
          bucket: my-bucket
          endpoint: minio:9000
          insecure: true
          key: work-avoidance-markers
          secretKeySecret:
            key: secretkey
            name: my-minio-cred
    metadata: {}
    name: load-markers
    outputs: {}
  - inputs:
      parameters:
      - name: num
    metadata: {}
    name: echo
    outputs: {}
    script:
      command:
      - bash
      - -eux
      image: docker/whalesay:latest
      name: ""
      resources: {}
      source: |
        marker=/work/markers/$(date +%Y-%m-%d)-echo-{{inputs.parameters.num}}
        if [ -e  ${marker} ]; then
          echo "work already done"
          exit 0
        fi
        echo "working very hard"
        # toss a virtual coin and exit 1 if 1
        if [ $(($(($RANDOM%10))%2)) -eq 1 ]; then
          echo "oh no!"
          exit 1
        fi
        touch ${marker}
      volumeMounts:
      - mountPath: /work
        name: work
  - container:
      command:
      - "true"
      image: docker/whalesay:latest
      name: ""
      resources: {}
      volumeMounts:
      - mountPath: /work
        name: work
    inputs: {}
    metadata: {}
    name: save-markers
    outputs:
      artifacts:
      - name: markers
        path: /work/markers
        s3:
          accessKeySecret:
            key: accesskey
            name: my-minio-cred
          bucket: my-bucket
          endpoint: minio:9000
          insecure: true
          key: work-avoidance-markers
          secretKeySecret:
            key: secretkey
            name: my-minio-cred
  volumeClaimTemplates:
  - metadata:
      creationTimestamp: null
      name: work
    spec:
      accessModes:
      - ReadWriteOnce
      resources:
        requests:
          storage: 10Mi
    status: {}
status:
  artifactGCStatus:
    notSpecified: true
  artifactRepositoryRef:
    artifactRepository:
      archiveLogs: true
      s3:
        accessKeySecret:
          key: accesskey
          name: my-minio-cred
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        secretKeySecret:
          key: secretkey
          name: my-minio-cred
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2024-10-02T05:54:41Z"
  nodes:
    work-avoidance-trkkq:
      children:
      - work-avoidance-trkkq-88427725
      displayName: work-avoidance-trkkq
      finishedAt: "2024-10-02T05:54:30Z"
      id: work-avoidance-trkkq
      name: work-avoidance-trkkq
      outboundNodes:
      - work-avoidance-trkkq-4180283560
      - work-avoidance-trkkq-605537244
      - work-avoidance-trkkq-4183398008
      phase: Succeeded
      progress: 4/4
      resourcesDuration:
        cpu: 1
        memory: 22
      startedAt: "2024-10-02T05:54:00Z"
      templateName: main
      templateScope: local/work-avoidance-trkkq
      type: Steps
    work-avoidance-trkkq-21464344:
      boundaryID: work-avoidance-trkkq
      children:
      - work-avoidance-trkkq-4180283560
      - work-avoidance-trkkq-605537244
      - work-avoidance-trkkq-4183398008
      displayName: '[1]'
      finishedAt: "2024-10-02T05:54:30Z"
      id: work-avoidance-trkkq-21464344
      name: work-avoidance-trkkq[1]
      nodeFlag: {}
      phase: Succeeded
      progress: 3/3
      resourcesDuration:
        cpu: 1
        memory: 18
      startedAt: "2024-10-02T05:54:14Z"
      templateScope: local/work-avoidance-trkkq
      type: StepGroup
    work-avoidance-trkkq-88427725:
      boundaryID: work-avoidance-trkkq
      children:
      - work-avoidance-trkkq-3329426915
      displayName: '[0]'
      finishedAt: "2024-10-02T05:54:14Z"
      id: work-avoidance-trkkq-88427725
      name: work-avoidance-trkkq[0]
      nodeFlag: {}
      phase: Succeeded
      progress: 4/4
      resourcesDuration:
        cpu: 1
        memory: 22
      startedAt: "2024-10-02T05:54:00Z"
      templateScope: local/work-avoidance-trkkq
      type: StepGroup
    work-avoidance-trkkq-605537244:
      boundaryID: work-avoidance-trkkq
      displayName: echo(1:1)
      finishedAt: "2024-10-02T05:54:24Z"
      hostNodeName: k3d-k3s-default-server-0
      id: work-avoidance-trkkq-605537244
      inputs:
        parameters:
        - name: num
          value: "1"
      name: work-avoidance-trkkq[1].echo(1:1)
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: work-avoidance-trkkq/work-avoidance-trkkq-echo-605537244/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 6
      startedAt: "2024-10-02T05:54:14Z"
      templateName: echo
      templateScope: local/work-avoidance-trkkq
      type: Pod
    work-avoidance-trkkq-1461956272:
      displayName: work-avoidance-trkkq.onExit
      finishedAt: "2024-10-02T05:54:38Z"
      hostNodeName: k3d-k3s-default-server-0
      id: work-avoidance-trkkq-1461956272
      name: work-avoidance-trkkq.onExit
      nodeFlag:
        hooked: true
      outputs:
        artifacts:
        - name: markers
          path: /work/markers
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: work-avoidance-markers
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        - name: main-logs
          s3:
            key: work-avoidance-trkkq/work-avoidance-trkkq-save-markers-1461956272/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 5
      startedAt: "2024-10-02T05:54:30Z"
      templateName: save-markers
      templateScope: local/work-avoidance-trkkq
      type: Pod
    work-avoidance-trkkq-3329426915:
      boundaryID: work-avoidance-trkkq
      children:
      - work-avoidance-trkkq-21464344
      displayName: load-markers
      finishedAt: "2024-10-02T05:54:12Z"
      hostNodeName: k3d-k3s-default-server-0
      id: work-avoidance-trkkq-3329426915
      inputs:
        artifacts:
        - name: markers
          optional: true
          path: /work/markers
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: work-avoidance-markers
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
      name: work-avoidance-trkkq[0].load-markers
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: work-avoidance-trkkq/work-avoidance-trkkq-load-markers-3329426915/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 4
      startedAt: "2024-10-02T05:54:00Z"
      templateName: load-markers
      templateScope: local/work-avoidance-trkkq
      type: Pod
    work-avoidance-trkkq-4180283560:
      boundaryID: work-avoidance-trkkq
      displayName: echo(0:0)
      finishedAt: "2024-10-02T05:54:27Z"
      hostNodeName: k3d-k3s-default-server-0
      id: work-avoidance-trkkq-4180283560
      inputs:
        parameters:
        - name: num
          value: "0"
      name: work-avoidance-trkkq[1].echo(0:0)
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: work-avoidance-trkkq/work-avoidance-trkkq-echo-4180283560/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 1
        memory: 8
      startedAt: "2024-10-02T05:54:14Z"
      templateName: echo
      templateScope: local/work-avoidance-trkkq
      type: Pod
    work-avoidance-trkkq-4183398008:
      boundaryID: work-avoidance-trkkq
      displayName: echo(2:2)
      finishedAt: "2024-10-02T05:54:21Z"
      hostNodeName: k3d-k3s-default-server-0
      id: work-avoidance-trkkq-4183398008
      inputs:
        parameters:
        - name: num
          value: "2"
      name: work-avoidance-trkkq[1].echo(2:2)
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: work-avoidance-trkkq/work-avoidance-trkkq-echo-4183398008/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 4
      startedAt: "2024-10-02T05:54:14Z"
      templateName: echo
      templateScope: local/work-avoidance-trkkq
      type: Pod
  phase: Succeeded
  progress: 5/5
  resourcesDuration:
    cpu: 1
    memory: 27
  startedAt: "2024-10-02T05:54:00Z"
  taskResultsCompletionStatus:
    work-avoidance-trkkq-605537244: true
    work-avoidance-trkkq-1461956272: true
    work-avoidance-trkkq-3329426915: true
    work-avoidance-trkkq-4180283560: true
    work-avoidance-trkkq-4183398008: true
`

func TestOnExitWorkflowRetry(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	wf := wfv1.MustUnmarshalWorkflow(onExitWorkflowRetry)
	running := map[string]bool{
		"work-avoidance-trkkq-21464344": true,
		"work-avoidance-trkkq":          true,
	}
	deleted := map[string]bool{
		"work-avoidance-trkkq-1461956272": true,
		"work-avoidance-trkkq-4183398008": true,
	}

	succeeded := make(map[string]bool)

	for _, node := range wf.Status.Nodes {
		_, inRunning := running[node.ID]
		_, inDeleted := deleted[node.ID]
		if !inRunning && !inDeleted {
			succeeded[node.ID] = true
		}
	}

	selectorStr := "id=work-avoidance-trkkq-4183398008"
	newWf, podsToDelete, err := FormulateRetryWorkflow(func() context.Context {
		ctx := context.Background()
		return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	}(), wf, true, selectorStr, []string{})
	require.NoError(err)
	assert.Len(newWf.Status.Nodes, 6)
	assert.Len(podsToDelete, 2)

	for _, node := range newWf.Status.Nodes {
		if _, ok := running[node.ID]; ok {
			assert.Equal(wfv1.NodeRunning, node.Phase)
		}
		if _, ok := succeeded[node.ID]; ok {
			assert.Equal(wfv1.NodeSucceeded, node.Phase)
		}
	}

}

const onExitWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  annotations:
    workflows.argoproj.io/pod-name-format: v2
  creationTimestamp: "2024-10-14T09:21:14Z"
  generation: 10
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Failed
  name: retry-workflow-with-failed-exit-handler
  namespace: argo
  resourceVersion: "13510"
  uid: f72bf6f7-3d8c-4b31-893b-ef03d4718959
spec:
  activeDeadlineSeconds: 300
  arguments: {}
  entrypoint: hello
  onExit: exit-handler
  podSpecPatch: |
    terminationGracePeriodSeconds: 3
  templates:
  - container:
      args:
      - echo hello
      command:
      - sh
      - -c
      image: alpine:3.18
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: hello
    outputs: {}
  - container:
      args:
      - exit 1
      command:
      - sh
      - -c
      image: alpine:3.18
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: exit-handler
    outputs: {}
status:
  artifactGCStatus:
    notSpecified: true
  artifactRepositoryRef:
    artifactRepository:
      archiveLogs: true
      s3:
        accessKeySecret:
          key: accesskey
          name: my-minio-cred
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        secretKeySecret:
          key: secretkey
          name: my-minio-cred
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2024-10-14T09:21:27Z"
  message: Error (exit code 1)
  nodes:
    retry-workflow-with-failed-exit-handler:
      displayName: retry-workflow-with-failed-exit-handler
      finishedAt: "2024-10-14T09:21:18Z"
      hostNodeName: k3d-k3s-default-server-0
      id: retry-workflow-with-failed-exit-handler
      name: retry-workflow-with-failed-exit-handler
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: retry-workflow-with-failed-exit-handler/retry-workflow-with-failed-exit-handler/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-14T09:21:14Z"
      templateName: hello
      templateScope: local/retry-workflow-with-failed-exit-handler
      type: Pod
    retry-workflow-with-failed-exit-handler-512308683:
      displayName: retry-workflow-with-failed-exit-handler.onExit
      finishedAt: "2024-10-14T09:21:24Z"
      hostNodeName: k3d-k3s-default-server-0
      id: retry-workflow-with-failed-exit-handler-512308683
      message: Error (exit code 1)
      name: retry-workflow-with-failed-exit-handler.onExit
      nodeFlag:
        hooked: true
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: retry-workflow-with-failed-exit-handler/retry-workflow-with-failed-exit-handler-exit-handler-512308683/main.log
        exitCode: "1"
      phase: Failed
      progress: 0/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-14T09:21:21Z"
      templateName: exit-handler
      templateScope: local/retry-workflow-with-failed-exit-handler
      type: Pod
  phase: Failed
  progress: 1/2
  resourcesDuration:
    cpu: 0
    memory: 4
  startedAt: "2024-10-14T09:21:14Z"
  taskResultsCompletionStatus:
    retry-workflow-with-failed-exit-handler: true
    retry-workflow-with-failed-exit-handler-512308683: true
`

func TestOnExitWorkflow(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	wf := wfv1.MustUnmarshalWorkflow(onExitWorkflow)

	newWf, podsToDelete, err := FormulateRetryWorkflow(func() context.Context {
		ctx := context.Background()
		return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	}(), wf, false, "", []string{})
	require.NoError(err)
	assert.Len(podsToDelete, 1)
	assert.Len(newWf.Status.Nodes, 1)
	assert.Equal(wfv1.NodeSucceeded, newWf.Status.Nodes["retry-workflow-with-failed-exit-handler"].Phase)

}

const nestedDAG = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  annotations:
    workflows.argoproj.io/pod-name-format: v2
  creationTimestamp: "2024-10-16T04:12:51Z"
  generateName: dag-nested-
  generation: 39
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
    workflows.argoproj.io/resubmitted-from-workflow: dag-nested-52l5t
  name: dag-nested-zxlc2
  namespace: argo
  resourceVersion: "11348"
  uid: 402ed1f0-0dbf-42fd-92b8-b7858ba2979c
spec:
  activeDeadlineSeconds: 300
  arguments: {}
  entrypoint: diamond
  podSpecPatch: |
    terminationGracePeriodSeconds: 3
  templates:
  - container:
      command:
      - echo
      - '{{inputs.parameters.message}}'
      image: alpine:3.7
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: message
    metadata: {}
    name: echo
    outputs: {}
  - dag:
      tasks:
      - arguments:
          parameters:
          - name: message
            value: A
        name: A
        template: nested-diamond
      - arguments:
          parameters:
          - name: message
            value: B
        depends: A
        name: B
        template: nested-diamond
      - arguments:
          parameters:
          - name: message
            value: C
        depends: A
        name: C
        template: nested-diamond
      - arguments:
          parameters:
          - name: message
            value: D
        depends: B && C
        name: D
        template: nested-diamond
    inputs: {}
    metadata: {}
    name: diamond
    outputs: {}
  - dag:
      tasks:
      - arguments:
          parameters:
          - name: message
            value: '{{inputs.parameters.message}}A'
        name: A
        template: echo
      - arguments:
          parameters:
          - name: message
            value: '{{inputs.parameters.message}}B'
        depends: A
        name: B
        template: echo
      - arguments:
          parameters:
          - name: message
            value: '{{inputs.parameters.message}}C'
        depends: A
        name: C
        template: echo
      - arguments:
          parameters:
          - name: message
            value: '{{inputs.parameters.message}}D'
        depends: B && C
        name: D
        template: echo
    inputs:
      parameters:
      - name: message
    metadata: {}
    name: nested-diamond
    outputs: {}
status:
  artifactGCStatus:
    notSpecified: true
  artifactRepositoryRef:
    artifactRepository:
      archiveLogs: true
      s3:
        accessKeySecret:
          key: accesskey
          name: my-minio-cred
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        secretKeySecret:
          key: secretkey
          name: my-minio-cred
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2024-10-16T04:13:49Z"
  nodes:
    dag-nested-zxlc2:
      children:
      - dag-nested-zxlc2-1970677234
      displayName: dag-nested-zxlc2
      finishedAt: "2024-10-16T04:13:49Z"
      id: dag-nested-zxlc2
      name: dag-nested-zxlc2
      outboundNodes:
      - dag-nested-zxlc2-644277987
      phase: Succeeded
      progress: 16/16
      resourcesDuration:
        cpu: 0
        memory: 30
      startedAt: "2024-10-16T04:12:51Z"
      templateName: diamond
      templateScope: local/dag-nested-zxlc2
      type: DAG
    dag-nested-zxlc2-644277987:
      boundaryID: dag-nested-zxlc2-1920344377
      displayName: D
      finishedAt: "2024-10-16T04:13:46Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-644277987
      inputs:
        parameters:
        - name: message
          value: DD
      name: dag-nested-zxlc2.D.D
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-644277987/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:13:43Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-694610844:
      boundaryID: dag-nested-zxlc2-1920344377
      children:
      - dag-nested-zxlc2-744943701
      - dag-nested-zxlc2-728166082
      displayName: A
      finishedAt: "2024-10-16T04:13:33Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-694610844
      inputs:
        parameters:
        - name: message
          value: DA
      name: dag-nested-zxlc2.D.A
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-694610844/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 1
      startedAt: "2024-10-16T04:13:30Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-725087280:
      boundaryID: dag-nested-zxlc2-1970677234
      children:
      - dag-nested-zxlc2-1953899615
      - dag-nested-zxlc2-1937121996
      displayName: D
      finishedAt: "2024-10-16T04:13:07Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-725087280
      inputs:
        parameters:
        - name: message
          value: AD
      name: dag-nested-zxlc2.A.D
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-725087280/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:13:03Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-728166082:
      boundaryID: dag-nested-zxlc2-1920344377
      children:
      - dag-nested-zxlc2-644277987
      displayName: C
      finishedAt: "2024-10-16T04:13:40Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-728166082
      inputs:
        parameters:
        - name: message
          value: DC
      name: dag-nested-zxlc2.D.C
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-728166082/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:13:36Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-744943701:
      boundaryID: dag-nested-zxlc2-1920344377
      children:
      - dag-nested-zxlc2-644277987
      displayName: B
      finishedAt: "2024-10-16T04:13:40Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-744943701
      inputs:
        parameters:
        - name: message
          value: DB
      name: dag-nested-zxlc2.D.B
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-744943701/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:13:36Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-808975375:
      boundaryID: dag-nested-zxlc2-1970677234
      children:
      - dag-nested-zxlc2-825752994
      - dag-nested-zxlc2-842530613
      displayName: A
      finishedAt: "2024-10-16T04:12:54Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-808975375
      inputs:
        parameters:
        - name: message
          value: AA
      name: dag-nested-zxlc2.A.A
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-808975375/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 1
      startedAt: "2024-10-16T04:12:51Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-825752994:
      boundaryID: dag-nested-zxlc2-1970677234
      children:
      - dag-nested-zxlc2-725087280
      displayName: B
      finishedAt: "2024-10-16T04:13:00Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-825752994
      inputs:
        parameters:
        - name: message
          value: AB
      name: dag-nested-zxlc2.A.B
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-825752994/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:12:57Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-842530613:
      boundaryID: dag-nested-zxlc2-1970677234
      children:
      - dag-nested-zxlc2-725087280
      displayName: C
      finishedAt: "2024-10-16T04:13:00Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-842530613
      inputs:
        parameters:
        - name: message
          value: AC
      name: dag-nested-zxlc2.A.C
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-842530613/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:12:57Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-903321510:
      boundaryID: dag-nested-zxlc2-1937121996
      children:
      - dag-nested-zxlc2-1920344377
      displayName: D
      finishedAt: "2024-10-16T04:13:27Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-903321510
      inputs:
        parameters:
        - name: message
          value: CD
      name: dag-nested-zxlc2.C.D
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-903321510/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:13:23Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-936876748:
      boundaryID: dag-nested-zxlc2-1937121996
      children:
      - dag-nested-zxlc2-903321510
      displayName: B
      finishedAt: "2024-10-16T04:13:20Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-936876748
      inputs:
        parameters:
        - name: message
          value: CB
      name: dag-nested-zxlc2.C.B
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-936876748/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:13:16Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-953654367:
      boundaryID: dag-nested-zxlc2-1937121996
      children:
      - dag-nested-zxlc2-903321510
      displayName: C
      finishedAt: "2024-10-16T04:13:20Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-953654367
      inputs:
        parameters:
        - name: message
          value: CC
      name: dag-nested-zxlc2.C.C
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-953654367/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:13:16Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-987209605:
      boundaryID: dag-nested-zxlc2-1937121996
      children:
      - dag-nested-zxlc2-936876748
      - dag-nested-zxlc2-953654367
      displayName: A
      finishedAt: "2024-10-16T04:13:13Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-987209605
      inputs:
        parameters:
        - name: message
          value: CA
      name: dag-nested-zxlc2.C.A
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-987209605/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:13:10Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-1920344377:
      boundaryID: dag-nested-zxlc2
      children:
      - dag-nested-zxlc2-694610844
      displayName: D
      finishedAt: "2024-10-16T04:13:49Z"
      id: dag-nested-zxlc2-1920344377
      inputs:
        parameters:
        - name: message
          value: D
      name: dag-nested-zxlc2.D
      outboundNodes:
      - dag-nested-zxlc2-644277987
      phase: Succeeded
      progress: 4/4
      resourcesDuration:
        cpu: 0
        memory: 7
      startedAt: "2024-10-16T04:13:30Z"
      templateName: nested-diamond
      templateScope: local/dag-nested-zxlc2
      type: DAG
    dag-nested-zxlc2-1937121996:
      boundaryID: dag-nested-zxlc2
      children:
      - dag-nested-zxlc2-987209605
      displayName: C
      finishedAt: "2024-10-16T04:13:30Z"
      id: dag-nested-zxlc2-1937121996
      inputs:
        parameters:
        - name: message
          value: C
      name: dag-nested-zxlc2.C
      outboundNodes:
      - dag-nested-zxlc2-903321510
      phase: Succeeded
      progress: 8/8
      resourcesDuration:
        cpu: 0
        memory: 15
      startedAt: "2024-10-16T04:13:10Z"
      templateName: nested-diamond
      templateScope: local/dag-nested-zxlc2
      type: DAG
    dag-nested-zxlc2-1953899615:
      boundaryID: dag-nested-zxlc2
      children:
      - dag-nested-zxlc2-3753141766
      displayName: B
      finishedAt: "2024-10-16T04:13:30Z"
      id: dag-nested-zxlc2-1953899615
      inputs:
        parameters:
        - name: message
          value: B
      name: dag-nested-zxlc2.B
      outboundNodes:
      - dag-nested-zxlc2-3837029861
      phase: Succeeded
      progress: 8/8
      resourcesDuration:
        cpu: 0
        memory: 15
      startedAt: "2024-10-16T04:13:10Z"
      templateName: nested-diamond
      templateScope: local/dag-nested-zxlc2
      type: DAG
    dag-nested-zxlc2-1970677234:
      boundaryID: dag-nested-zxlc2
      children:
      - dag-nested-zxlc2-808975375
      displayName: A
      finishedAt: "2024-10-16T04:13:10Z"
      id: dag-nested-zxlc2-1970677234
      inputs:
        parameters:
        - name: message
          value: A
      name: dag-nested-zxlc2.A
      outboundNodes:
      - dag-nested-zxlc2-725087280
      phase: Succeeded
      progress: 16/16
      resourcesDuration:
        cpu: 0
        memory: 30
      startedAt: "2024-10-16T04:12:51Z"
      templateName: nested-diamond
      templateScope: local/dag-nested-zxlc2
      type: DAG
    dag-nested-zxlc2-3719586528:
      boundaryID: dag-nested-zxlc2-1953899615
      children:
      - dag-nested-zxlc2-3837029861
      displayName: C
      finishedAt: "2024-10-16T04:13:20Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-3719586528
      inputs:
        parameters:
        - name: message
          value: BC
      name: dag-nested-zxlc2.B.C
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-3719586528/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:13:16Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-3736364147:
      boundaryID: dag-nested-zxlc2-1953899615
      children:
      - dag-nested-zxlc2-3837029861
      displayName: B
      finishedAt: "2024-10-16T04:13:20Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-3736364147
      inputs:
        parameters:
        - name: message
          value: BB
      name: dag-nested-zxlc2.B.B
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-3736364147/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:13:16Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-3753141766:
      boundaryID: dag-nested-zxlc2-1953899615
      children:
      - dag-nested-zxlc2-3736364147
      - dag-nested-zxlc2-3719586528
      displayName: A
      finishedAt: "2024-10-16T04:13:13Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-3753141766
      inputs:
        parameters:
        - name: message
          value: BA
      name: dag-nested-zxlc2.B.A
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-3753141766/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:13:10Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
    dag-nested-zxlc2-3837029861:
      boundaryID: dag-nested-zxlc2-1953899615
      children:
      - dag-nested-zxlc2-1920344377
      displayName: D
      finishedAt: "2024-10-16T04:13:27Z"
      hostNodeName: k3d-k3s-default-server-0
      id: dag-nested-zxlc2-3837029861
      inputs:
        parameters:
        - name: message
          value: BD
      name: dag-nested-zxlc2.B.D
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: dag-nested-zxlc2/dag-nested-zxlc2-echo-3837029861/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 2
      startedAt: "2024-10-16T04:13:23Z"
      templateName: echo
      templateScope: local/dag-nested-zxlc2
      type: Pod
  phase: Succeeded
  progress: 16/16
  resourcesDuration:
    cpu: 0
    memory: 30
  startedAt: "2024-10-16T04:12:51Z"
  taskResultsCompletionStatus:
    dag-nested-zxlc2-644277987: true
    dag-nested-zxlc2-694610844: true
    dag-nested-zxlc2-725087280: true
    dag-nested-zxlc2-728166082: true
    dag-nested-zxlc2-744943701: true
    dag-nested-zxlc2-808975375: true
    dag-nested-zxlc2-825752994: true
    dag-nested-zxlc2-842530613: true
    dag-nested-zxlc2-903321510: true
    dag-nested-zxlc2-936876748: true
    dag-nested-zxlc2-953654367: true
    dag-nested-zxlc2-987209605: true
    dag-nested-zxlc2-3719586528: true
    dag-nested-zxlc2-3736364147: true
    dag-nested-zxlc2-3753141766: true
    dag-nested-zxlc2-3837029861: true

`

func TestNestedDAG(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	wf := wfv1.MustUnmarshalWorkflow(nestedDAG)

	running := map[string]bool{
		"dag-nested-zxlc2-1920344377":  true,
		"dag-nested-zxlc2-1970677234 ": true,
		"dag-nested-zxlc2":             true,
	}
	deleted := map[string]bool{
		"dag-nested-zxlc2-744943701": true,
		"dag-nested-zxlc2-644277987": true,
	}

	succeeded := map[string]bool{}

	for _, node := range wf.Status.Nodes {
		_, inRunning := running[node.ID]
		_, inDeleted := deleted[node.ID]
		if !inRunning && !inDeleted {
			succeeded[node.ID] = true
		}
	}

	newWf, podsToDelete, err := FormulateRetryWorkflow(func() context.Context {
		ctx := context.Background()
		return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	}(), wf, true, "id=dag-nested-zxlc2-744943701", []string{})
	require.NoError(err)
	assert.Len(podsToDelete, 2)

	for _, node := range newWf.Status.Nodes {
		if _, ok := running[node.ID]; ok {
			assert.Equal(wfv1.NodeRunning, node.Phase)
		}
		if _, ok := succeeded[node.ID]; ok {
			assert.Equal(wfv1.NodeSucceeded, node.Phase)
		}
	}

}

const onExitPanic = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  annotations:
    workflows.argoproj.io/pod-name-format: v2
  creationTimestamp: "2025-02-11T05:25:47Z"
  generateName: exit-handlers-
  generation: 21
  labels:
    default-label: thisLabelIsFromWorkflowDefaults
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Failed
  name: exit-handlers-n7s4n
  namespace: argo
  resourceVersion: "2255"
  uid: 7b2f1451-9a9a-4f66-a0d9-0364f814d948
spec:
  activeDeadlineSeconds: 300
  arguments: {}
  entrypoint: intentional-fail
  onExit: exit-handler
  podSpecPatch: |
    terminationGracePeriodSeconds: 3
  templates:
  - container:
      args:
      - echo intentional failure; exit 1
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: intentional-fail
    outputs: {}
  - inputs: {}
    metadata: {}
    name: exit-handler
    outputs: {}
    steps:
    - - arguments: {}
        name: notify
        template: send-email
      - arguments: {}
        name: celebrate
        template: celebrate
        when: '{{workflow.status}} == Succeeded'
      - arguments: {}
        name: cry
        template: cry
        when: '{{workflow.status}} != Succeeded'
  - container:
      args:
      - 'echo send e-mail: {{workflow.name}} {{workflow.status}} {{workflow.duration}}.
        Failed steps {{workflow.failures}}'
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: send-email
    outputs: {}
  - container:
      args:
      - echo hooray!
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: celebrate
    outputs: {}
  - container:
      args:
      - echo boohoo!
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: cry
    outputs: {}
  workflowMetadata:
    labels:
      default-label: thisLabelIsFromWorkflowDefaults
status:
  artifactGCStatus:
    notSpecified: true
  artifactRepositoryRef:
    artifactRepository:
      archiveLogs: true
      s3:
        accessKeySecret:
          key: accesskey
          name: my-minio-cred
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        secretKeySecret:
          key: secretkey
          name: my-minio-cred
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2025-02-11T05:31:30Z"
  message: 'main: Error (exit code 1)'
  nodes:
    exit-handlers-n7s4n:
      displayName: exit-handlers-n7s4n
      finishedAt: "2025-02-11T05:31:18Z"
      hostNodeName: k3d-k3s-default-server-0
      id: exit-handlers-n7s4n
      message: 'main: Error (exit code 1)'
      name: exit-handlers-n7s4n
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: exit-handlers-n7s4n/exit-handlers-n7s4n/main.log
        exitCode: "1"
      phase: Failed
      progress: 0/1
      resourcesDuration:
        cpu: 0
        memory: 4
      startedAt: "2025-02-11T05:31:12Z"
      templateName: intentional-fail
      templateScope: local/exit-handlers-n7s4n
      type: Pod
    exit-handlers-n7s4n-134905866:
      boundaryID: exit-handlers-n7s4n-1410405845
      displayName: celebrate
      finishedAt: "2025-02-11T05:31:21Z"
      id: exit-handlers-n7s4n-134905866
      message: when 'Failed == Succeeded' evaluated false
      name: exit-handlers-n7s4n.onExit[0].celebrate
      nodeFlag: {}
      phase: Skipped
      startedAt: "2025-02-11T05:31:21Z"
      templateName: celebrate
      templateScope: local/exit-handlers-n7s4n
      type: Skipped
    exit-handlers-n7s4n-975057257:
      boundaryID: exit-handlers-n7s4n-1410405845
      children:
      - exit-handlers-n7s4n-3201878844
      - exit-handlers-n7s4n-134905866
      - exit-handlers-n7s4n-2699669595
      displayName: '[0]'
      finishedAt: "2025-02-11T05:31:30Z"
      id: exit-handlers-n7s4n-975057257
      name: exit-handlers-n7s4n.onExit[0]
      nodeFlag: {}
      phase: Succeeded
      progress: 2/2
      resourcesDuration:
        cpu: 0
        memory: 6
      startedAt: "2025-02-11T05:31:21Z"
      templateScope: local/exit-handlers-n7s4n
      type: StepGroup
    exit-handlers-n7s4n-1410405845:
      children:
      - exit-handlers-n7s4n-975057257
      displayName: exit-handlers-n7s4n.onExit
      finishedAt: "2025-02-11T05:31:30Z"
      id: exit-handlers-n7s4n-1410405845
      name: exit-handlers-n7s4n.onExit
      nodeFlag:
        hooked: true
      outboundNodes:
      - exit-handlers-n7s4n-3201878844
      - exit-handlers-n7s4n-134905866
      - exit-handlers-n7s4n-2699669595
      phase: Succeeded
      progress: 2/2
      resourcesDuration:
        cpu: 0
        memory: 6
      startedAt: "2025-02-11T05:31:21Z"
      templateName: exit-handler
      templateScope: local/exit-handlers-n7s4n
      type: Steps
    exit-handlers-n7s4n-2699669595:
      boundaryID: exit-handlers-n7s4n-1410405845
      displayName: cry
      finishedAt: "2025-02-11T05:31:27Z"
      hostNodeName: k3d-k3s-default-server-0
      id: exit-handlers-n7s4n-2699669595
      name: exit-handlers-n7s4n.onExit[0].cry
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: exit-handlers-n7s4n/exit-handlers-n7s4n-cry-2699669595/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 3
      startedAt: "2025-02-11T05:31:21Z"
      templateName: cry
      templateScope: local/exit-handlers-n7s4n
      type: Pod
    exit-handlers-n7s4n-3201878844:
      boundaryID: exit-handlers-n7s4n-1410405845
      displayName: notify
      finishedAt: "2025-02-11T05:31:27Z"
      hostNodeName: k3d-k3s-default-server-0
      id: exit-handlers-n7s4n-3201878844
      name: exit-handlers-n7s4n.onExit[0].notify
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: exit-handlers-n7s4n/exit-handlers-n7s4n-send-email-3201878844/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 3
      startedAt: "2025-02-11T05:31:21Z"
      templateName: send-email
      templateScope: local/exit-handlers-n7s4n
      type: Pod
  phase: Failed
  progress: 2/3
  resourcesDuration:
    cpu: 0
    memory: 10
  startedAt: "2025-02-11T05:31:12Z"
  taskResultsCompletionStatus:
    exit-handlers-n7s4n: true
    exit-handlers-n7s4n-2699669595: true
    exit-handlers-n7s4n-3201878844: true
`

func TestRegressions(t *testing.T) {
	t.Run("exit handler", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(onExitPanic)
		newWf, _, err := FormulateRetryWorkflow(func() context.Context {
			ctx := context.Background()
			return logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
		}(), wf, true, "id=exit-handlers-n7s4n-975057257", []string{})
		require.NoError(t, err)
		// we can't really handle exit handlers granually yet
		assert.Empty(t, newWf.Status.Nodes)
	})
}
