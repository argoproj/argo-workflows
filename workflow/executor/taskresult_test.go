package executor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/mocks"
)

// taskResultWrite is one create or patch of the task result, as the API
// server received it.
type taskResultWrite struct {
	verb string
	// completed is the LabelKeyReportOutputsCompleted value the write
	// carried, or "" if it left the label alone.
	completed string
	outputs   *wfv1.Outputs
	progress  wfv1.Progress
	phase     wfv1.NodePhase
}

// taskResultWrites returns the creates and patches of the task result, in
// the order they reached the clientset.
func taskResultWrites(t *testing.T, clientset *argofake.Clientset) []taskResultWrite {
	t.Helper()
	var writes []taskResultWrite
	for _, action := range clientset.Actions() {
		if action.GetResource().Resource != "workflowtaskresults" {
			continue
		}
		var result *wfv1.WorkflowTaskResult
		switch a := action.(type) {
		case k8stesting.CreateAction:
			result = a.GetObject().(*wfv1.WorkflowTaskResult)
		case k8stesting.PatchAction:
			result = &wfv1.WorkflowTaskResult{}
			require.NoError(t, json.Unmarshal(a.GetPatch(), result))
		default:
			continue
		}
		writes = append(writes, taskResultWrite{
			verb:      action.GetVerb(),
			completed: result.Labels[common.LabelKeyReportOutputsCompleted],
			outputs:   result.Outputs,
			progress:  result.Progress,
			phase:     result.Phase,
		})
	}
	return writes
}

func verbs(writes []taskResultWrite) []string {
	v := make([]string, len(writes))
	for i, w := range writes {
		v[i] = w.verb + ":" + w.completed
	}
	return v
}

// newTaskResultTestExecutor returns an executor whose task result writes
// can be inspected through the returned clientset.
func newTaskResultTestExecutor(t *testing.T, tmpl wfv1.Template, rt *mocks.ContainerRuntimeExecutor, clientset *argofake.Clientset) *WorkflowExecutor {
	t.Helper()
	we := newTestPostMainExecutor(t, tmpl, rt)
	we.taskResultClient = clientset.ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace)
	we.sink = we.NewTaskResultSink(TaskConfig{WorkflowName: we.workflow, WorkflowUID: we.workflowUID, NodeID: we.nodeID}, false)
	return we
}

func getStoredTaskResult(t *testing.T, clientset *argofake.Clientset) *wfv1.WorkflowTaskResult {
	t.Helper()
	result, err := clientset.ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace).Get(t.Context(), fakeNodeID, metav1.GetOptions{})
	require.NoError(t, err)
	return result
}

// TestTaskResultWrites_ContainerPlans checks the wait and supervisor flow:
// the placeholder create, nothing from Collect, then one patch from
// FinalizeOutput carrying both the outputs and the completed label.
func TestTaskResultWrites_ContainerPlans(t *testing.T) {
	script := wfv1.Template{Script: &wfv1.ScriptTemplate{}}
	tests := []struct {
		name string
		tmpl wfv1.Template
		// preMainFailed skips the stages that need main's outputs, but
		// report-outputs must still run and FinalizeOutput complete the
		// result.
		preMainFailed bool
		wantResult    *string
		// wantErr is set when a collect stage fails without stopping
		// report-outputs.
		wantErr bool
	}{
		{name: "Script", tmpl: script, wantResult: new("hello")},
		{name: "PreMainFailed", tmpl: script, preMainFailed: true},
		{
			// Only a resource template leaves completion to another
			// container, so archived logs here must not hold the label
			// back.
			name:    "ArchivedLogs",
			tmpl:    wfv1.Template{Container: &corev1.Container{}, ArchiveLocation: &wfv1.ArtifactLocation{ArchiveLogs: new(true)}},
			wantErr: true, // no artifact repository, so saving the logs records an error
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			rt := &mocks.ContainerRuntimeExecutor{}
			rt.On("Wait", mock.Anything, []string{"main"}).Return(nil)
			rt.On("GetOutputStream", mock.Anything, "main", false).Return(io.NopCloser(strings.NewReader("hello")), nil).Maybe()
			rt.On("GetOutputStream", mock.Anything, "main", true).Return(io.NopCloser(strings.NewReader("log")), nil).Maybe()
			clientset := argofake.NewClientset()
			we := newTaskResultTestExecutor(t, tt.tmpl, rt, clientset)
			we.IncludeScriptOutput = tt.tmpl.Script != nil

			err := we.PostMain(ctx, ctx, tt.preMainFailed)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, []string{"create:false"}, verbs(taskResultWrites(t, clientset)), "Collect leaves the outputs for FinalizeOutput")
			we.FinalizeOutput(ctx)

			writes := taskResultWrites(t, clientset)
			assert.Equal(t, []string{"create:false", "patch:true"}, verbs(writes))
			require.NotNil(t, writes[1].outputs, "the completing write carries the outputs")
			assert.Equal(t, tt.wantResult, writes[1].outputs.Result)
			assert.Equal(t, "true", getStoredTaskResult(t, clientset).Labels[common.LabelKeyReportOutputsCompleted])
		})
	}
}

// TestTaskResultWrites_SaveFailureStillCompletes checks that a failing
// collect stage doesn't stop report-outputs from reporting, or
// FinalizeOutput from completing the result in one write.
func TestTaskResultWrites_SaveFailureStillCompletes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	rt := &mocks.ContainerRuntimeExecutor{}
	rt.On("Wait", mock.Anything, []string{"main"}).Return(nil)
	rt.On("GetFileContents", mock.Anything, mock.Anything).Return("", errors.New("no such file")).Maybe()
	clientset := argofake.NewClientset()
	tmpl := wfv1.Template{
		Container: &corev1.Container{},
		Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{
			{Name: "p", ValueFrom: &wfv1.ValueFrom{Path: "/tmp/argo-test-missing-param"}},
		}},
	}
	we := newTaskResultTestExecutor(t, tmpl, rt, clientset)

	require.Error(t, we.PostMain(ctx, ctx, false), "save-parameters fails on the missing file")
	we.FinalizeOutput(ctx)

	assert.Equal(t, []string{"create:false", "patch:true"}, verbs(taskResultWrites(t, clientset)))
}

// TestTaskResultWrites_RejectedResultStillCompletes checks that when the
// API server rejects the final write's outputs, FinalizeOutput still sets
// the label, without them, and records the rejection.
func TestTaskResultWrites_RejectedResultStillCompletes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	rt := &mocks.ContainerRuntimeExecutor{}
	rt.On("Wait", mock.Anything, []string{"main"}).Return(nil)
	clientset := argofake.NewClientset()
	clientset.PrependReactor("patch", "workflowtaskresults", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		if strings.Contains(string(action.(k8stesting.PatchAction).GetPatch()), `"outputs"`) {
			return true, nil, apierr.NewBadRequest("outputs rejected")
		}
		return false, nil, nil
	})
	we := newTaskResultTestExecutor(t, wfv1.Template{Container: &corev1.Container{}}, rt, clientset)

	require.NoError(t, we.PostMain(ctx, ctx, false))
	we.FinalizeOutput(ctx)

	writes := taskResultWrites(t, clientset)
	assert.Equal(t, []string{"create:false", "patch:true", "patch:true"}, verbs(writes), "rejected final write, then the label alone")
	assert.Nil(t, writes[2].outputs, "the fallback writes only the label")
	assert.Equal(t, "true", getStoredTaskResult(t, clientset).Labels[common.LabelKeyReportOutputsCompleted])
	require.Error(t, we.HasError())
	assert.True(t, apierr.IsBadRequest(we.HasError()), "the rejection is the error, got %v", we.HasError())
}

// TestTaskResultWrites_FinalizeOutputFor checks that a command outside the
// harness (`argoexec resource`, `argoexec data`) fails when its final write
// does, keeping its own earlier error if it had one.
func TestTaskResultWrites_FinalizeOutputFor(t *testing.T) {
	rejecting := func() *argofake.Clientset {
		clientset := argofake.NewClientset()
		clientset.PrependReactor("patch", "workflowtaskresults", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			if strings.Contains(string(action.(k8stesting.PatchAction).GetPatch()), `"outputs"`) {
				return true, nil, apierr.NewBadRequest("outputs rejected")
			}
			return false, nil, nil
		})
		return clientset
	}
	tests := []struct {
		name      string
		clientset *argofake.Clientset
		err       error
		check     func(t *testing.T, err error)
	}{
		{"write succeeds", argofake.NewClientset(), nil, func(t *testing.T, err error) {
			require.NoError(t, err)
		}},
		{"write rejected", rejecting(), nil, func(t *testing.T, err error) {
			require.Error(t, err)
			assert.True(t, apierr.IsBadRequest(err), "the rejection is the command's error, got %v", err)
		}},
		{"earlier error kept", rejecting(), errors.New("earlier"), func(t *testing.T, err error) {
			require.EqualError(t, err, "earlier")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			we := newTaskResultTestExecutor(t, wfv1.Template{Container: &corev1.Container{}}, &mocks.ContainerRuntimeExecutor{}, tt.clientset)
			we.InitializeOutput(ctx)
			require.NoError(t, we.ReportOutputs(ctx, nil, true))

			tt.check(t, we.FinalizeOutputFor(ctx, tt.err))
			assert.Equal(t, "true", getStoredTaskResult(t, tt.clientset).Labels[common.LabelKeyReportOutputsCompleted], "the result is complete either way")
		})
	}
}

// TestTaskResultWrites_OutageFailsOnce checks that when the API server is
// unavailable for the whole retry timeout, FinalizeOutput gives up without
// a second round of retries for the label alone.
func TestTaskResultWrites_OutageFailsOnce(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	we, _ := newFlakyTaskResultExecutor(t, -1, 50*time.Millisecond)
	require.NoError(t, we.ReportOutputs(ctx, nil, true))

	start := time.Now()
	we.FinalizeOutput(ctx)
	assert.Less(t, time.Since(start), 10*time.Second)
	require.Len(t, we.errors, 1)
	assert.True(t, apierr.IsServiceUnavailable(we.errors[0]), "got %v", we.errors[0])
}

// TestTaskResultWrites_PanicFinalizes checks that a panic before the
// outputs are reported still leaves the result complete, through the
// deferred FinalizeOutput the commands install.
func TestTaskResultWrites_PanicFinalizes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	clientset := argofake.NewClientset()
	we := newTaskResultTestExecutor(t, wfv1.Template{Container: &corev1.Container{}}, &mocks.ContainerRuntimeExecutor{}, clientset)

	func() {
		defer func() { assert.NotNil(t, recover()) }()
		defer we.FinalizeOutput(ctx)
		we.InitializeOutput(ctx)
		panic("boom")
	}()

	assert.Equal(t, []string{"create:false", "patch:true"}, verbs(taskResultWrites(t, clientset)))
}

// TestTaskResultWrites_Resource covers the resource template, whose main
// container (`argoexec resource`) reports the output parameters. Without
// archived logs it is the only reporter and completes the result itself;
// with archived logs a wait container reports the logs afterwards, so main
// must leave the label for that later write.
func TestTaskResultWrites_Resource(t *testing.T) {
	param := wfv1.Parameter{Name: "p", ValueFrom: &wfv1.ValueFrom{JSONPath: "{.metadata.name}", Default: wfv1.AnyStringPtr("dflt")}}

	t.Run("NoArchivedLogs", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		clientset := argofake.NewClientset()
		tmpl := wfv1.Template{Resource: &wfv1.ResourceTemplate{Action: "create"}, Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{param}}}
		we := newTaskResultTestExecutor(t, tmpl, &mocks.ContainerRuntimeExecutor{}, clientset)

		we.InitializeOutput(ctx)
		require.NoError(t, we.SaveResourceParameters(ctx, "", "", true))
		assert.Equal(t, []string{"create:false"}, verbs(taskResultWrites(t, clientset)), "the parameters wait for FinalizeOutput")
		we.FinalizeOutput(ctx)

		writes := taskResultWrites(t, clientset)
		assert.Equal(t, []string{"create:false", "patch:true"}, verbs(writes))
		assert.Equal(t, "dflt", writes[1].outputs.Parameters[0].Value.String())
	})

	t.Run("NoOutputParameters", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		clientset := argofake.NewClientset()
		we := newTaskResultTestExecutor(t, wfv1.Template{Resource: &wfv1.ResourceTemplate{Action: "create"}}, &mocks.ContainerRuntimeExecutor{}, clientset)

		we.InitializeOutput(ctx)
		require.NoError(t, we.SaveResourceParameters(ctx, "", "", true))
		we.FinalizeOutput(ctx)

		assert.Equal(t, []string{"create:false", "patch:true"}, verbs(taskResultWrites(t, clientset)), "nothing to report, so FinalizeOutput sets the label")
	})

	t.Run("ArchivedLogs", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		clientset := argofake.NewClientset()
		tmpl := wfv1.Template{
			Resource:        &wfv1.ResourceTemplate{Action: "create"},
			Outputs:         wfv1.Outputs{Parameters: []wfv1.Parameter{param}},
			ArchiveLocation: &wfv1.ArtifactLocation{ArchiveLogs: new(true)},
		}

		// main: `argoexec resource`, which leaves completion to wait here, so
		// its report isn't final: it is written straight away, and main
		// doesn't defer FinalizeOutput.
		main := newTaskResultTestExecutor(t, tmpl, &mocks.ContainerRuntimeExecutor{}, clientset)
		main.InitializeOutput(ctx)
		require.NoError(t, main.SaveResourceParameters(ctx, "", "", false))
		assert.Equal(t, []string{"create:false", "patch:"}, verbs(taskResultWrites(t, clientset)), "main must not complete the result")
		assert.Equal(t, "false", getStoredTaskResult(t, clientset).Labels[common.LabelKeyReportOutputsCompleted])

		// wait: a separate process, so a fresh executor.
		rt := &mocks.ContainerRuntimeExecutor{}
		rt.On("Wait", mock.Anything, []string{"main"}).Return(nil)
		rt.On("GetOutputStream", mock.Anything, mock.Anything, true).Return(io.NopCloser(strings.NewReader("log")), nil).Maybe()
		wait := newTaskResultTestExecutor(t, tmpl, rt, clientset)
		_ = wait.PostMain(ctx, ctx, false) // no artifact repository, so saving the logs records an error
		writes := taskResultWrites(t, clientset)
		wait.FinalizeOutput(ctx)

		// wait's placeholder create finds main's result and patches it;
		// FinalizeOutput then patches in the logs and the label together.
		assert.Equal(t, []string{"create:false", "patch:", "create:false", "patch:"}, verbs(writes), "the logs wait for FinalizeOutput")
		writes = taskResultWrites(t, clientset)
		assert.Equal(t, []string{"create:false", "patch:", "create:false", "patch:", "patch:true"}, verbs(writes))
		result := getStoredTaskResult(t, clientset)
		assert.Equal(t, "true", result.Labels[common.LabelKeyReportOutputsCompleted])
		assert.Equal(t, "dflt", result.Outputs.Parameters[0].Value.String(), "main's parameters survive the logs report")
	})
}

// TestTaskResultWrites_Data checks the data template's flow: it has no
// placeholder, so FinalizeOutput creates the result already complete.
func TestTaskResultWrites_Data(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		clientset := argofake.NewClientset()
		we := newTaskResultTestExecutor(t, wfv1.Template{Data: &wfv1.Data{}}, &mocks.ContainerRuntimeExecutor{}, clientset)
		we.capturedOutputs().Result = new("[]")

		// the tail of Data(), after the data is processed
		require.NoError(t, we.ReportOutputs(ctx, nil, true))
		we.FinalizeOutput(ctx)

		writes := taskResultWrites(t, clientset)
		assert.Equal(t, []string{"create:true"}, verbs(writes))
		assert.Equal(t, "[]", *writes[0].outputs.Result)
	})

	t.Run("ProcessingFails", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		clientset := argofake.NewClientset()
		we := newTaskResultTestExecutor(t, wfv1.Template{}, &mocks.ContainerRuntimeExecutor{}, clientset)

		require.Error(t, we.Data(ctx))
		we.FinalizeOutput(ctx)

		// Nothing was reported, so FinalizeOutput writes only the label. As
		// before, no result was ever created, so there is none for the
		// controller to wait on.
		assert.Equal(t, []string{"patch:true"}, verbs(taskResultWrites(t, clientset)))
		_, err := clientset.ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace).Get(ctx, fakeNodeID, metav1.GetOptions{})
		assert.True(t, apierr.IsNotFound(err))
	})
}

// TestTaskResultWrites_ProgressNeverCompletes checks that progress reports,
// which run while main is still going, never set the completed label.
func TestTaskResultWrites_ProgressNeverCompletes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	clientset := argofake.NewClientset()
	we := newTaskResultTestExecutor(t, wfv1.Template{Container: &corev1.Container{}}, &mocks.ContainerRuntimeExecutor{}, clientset)
	we.annotationPatchTickDuration = 5 * time.Millisecond
	we.readProgressFileTickDuration = time.Millisecond
	progressFile := filepath.Join(t.TempDir(), "progress")

	we.InitializeOutput(ctx)
	monitorCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		we.monitorProgress(monitorCtx, progressFile)
	}()
	require.NoError(t, os.WriteFile(progressFile, []byte("1/2\n"), 0o600))
	require.Eventually(t, func() bool {
		return getStoredTaskResult(t, clientset).Progress == "1/2"
	}, 10*time.Second, 10*time.Millisecond)
	cancel()
	<-done

	writes := taskResultWrites(t, clientset)
	require.Greater(t, len(writes), 1)
	for _, w := range writes {
		assert.NotEqual(t, "true", w.completed, "%s carrying progress %q completed the result", w.verb, w.progress)
	}
	assert.Equal(t, "false", getStoredTaskResult(t, clientset).Labels[common.LabelKeyReportOutputsCompleted])
}
