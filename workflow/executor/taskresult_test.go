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
	return we
}

func getTaskResult(t *testing.T, clientset *argofake.Clientset) *wfv1.WorkflowTaskResult {
	t.Helper()
	result, err := clientset.ArgoprojV1alpha1().WorkflowTaskResults(fakeNamespace).Get(t.Context(), fakeNodeID, metav1.GetOptions{})
	require.NoError(t, err)
	return result
}

// TestTaskResultWrites_ContainerPlans checks the wait and supervisor flow:
// the placeholder create, then one patch carrying both the outputs and the
// completed label, and nothing from the deferred FinalizeOutput.
func TestTaskResultWrites_ContainerPlans(t *testing.T) {
	for _, initless := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "initless"}[initless], func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			rt := &mocks.ContainerRuntimeExecutor{}
			rt.On("Wait", mock.Anything, []string{"main"}).Return(nil)
			rt.On("GetOutputStream", mock.Anything, "main", false).Return(io.NopCloser(strings.NewReader("hello")), nil)
			clientset := argofake.NewClientset()
			we := newTaskResultTestExecutor(t, wfv1.Template{Script: &wfv1.ScriptTemplate{}}, rt, clientset)
			we.IncludeScriptOutput = true
			we.initlessPod = initless

			require.NoError(t, we.PostMain(ctx, ctx, false))
			we.FinalizeOutput(ctx)

			writes := taskResultWrites(t, clientset)
			assert.Equal(t, []string{"create:false", "patch:true"}, verbs(writes))
			require.NotNil(t, writes[1].outputs, "the completing write carries the outputs")
			assert.Equal(t, "hello", *writes[1].outputs.Result)
			assert.Equal(t, "true", getTaskResult(t, clientset).Labels[common.LabelKeyReportOutputsCompleted])
			assert.NoError(t, we.HasError())
		})
	}
}

// TestTaskResultWrites_SaveFailureStillCompletes checks that a failing
// collect stage doesn't stop report-outputs from completing the result.
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

// TestTaskResultWrites_ReportFailureFinalizes checks that when the outputs
// report fails, the deferred FinalizeOutput still sets the label.
func TestTaskResultWrites_ReportFailureFinalizes(t *testing.T) {
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

	require.Error(t, we.PostMain(ctx, ctx, false))
	we.FinalizeOutput(ctx)

	writes := taskResultWrites(t, clientset)
	assert.Equal(t, []string{"create:false", "patch:true", "patch:true"}, verbs(writes), "rejected outputs patch, then FinalizeOutput's label patch")
	assert.Nil(t, writes[2].outputs, "FinalizeOutput writes only the label")
	assert.Equal(t, "true", getTaskResult(t, clientset).Labels[common.LabelKeyReportOutputsCompleted])
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
		// its report isn't final and it doesn't defer FinalizeOutput.
		main := newTaskResultTestExecutor(t, tmpl, &mocks.ContainerRuntimeExecutor{}, clientset)
		main.InitializeOutput(ctx)
		require.NoError(t, main.SaveResourceParameters(ctx, "", "", false))
		assert.Equal(t, []string{"create:false", "patch:"}, verbs(taskResultWrites(t, clientset)), "main must not complete the result")
		assert.Equal(t, "false", getTaskResult(t, clientset).Labels[common.LabelKeyReportOutputsCompleted])

		// wait: a separate process, so a fresh executor.
		rt := &mocks.ContainerRuntimeExecutor{}
		rt.On("Wait", mock.Anything, []string{"main"}).Return(nil)
		rt.On("GetOutputStream", mock.Anything, mock.Anything, true).Return(io.NopCloser(strings.NewReader("log")), nil).Maybe()
		wait := newTaskResultTestExecutor(t, tmpl, rt, clientset)
		_ = wait.PostMain(ctx, ctx, false) // no artifact repository, so saving the logs records an error
		wait.FinalizeOutput(ctx)

		writes := taskResultWrites(t, clientset)
		last := len(writes) - 1
		assert.Equal(t, "true", writes[last].completed, "the logs report completes the result")
		for _, w := range writes[:last] {
			assert.NotEqual(t, "true", w.completed, "only the last write may complete the result")
		}
		result := getTaskResult(t, clientset)
		assert.Equal(t, "true", result.Labels[common.LabelKeyReportOutputsCompleted])
		assert.Equal(t, "dflt", result.Outputs.Parameters[0].Value.String(), "main's parameters survive the logs report")
	})
}

// TestTaskResultWrites_Data checks the data template's flow: it has no
// placeholder, so its one report creates the result already complete.
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

		// As before, no result was ever created, so there is none for the
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
		return getTaskResult(t, clientset).Progress == "1/2"
	}, 10*time.Second, 10*time.Millisecond)
	cancel()
	<-done

	writes := taskResultWrites(t, clientset)
	require.Greater(t, len(writes), 1)
	for _, w := range writes {
		assert.NotEqual(t, "true", w.completed, "%s carrying progress %q completed the result", w.verb, w.progress)
	}
	assert.Equal(t, "false", getTaskResult(t, clientset).Labels[common.LabelKeyReportOutputsCompleted])
}
