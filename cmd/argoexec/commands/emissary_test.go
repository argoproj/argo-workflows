//go:build !windows

package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo-workflows/v4/util/cmd"
	"github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
)

func TestEmissary(t *testing.T) {
	tmp := t.TempDir()

	varRunArgo = tmp

	err := os.WriteFile(varRunArgo+"/template", []byte(`{}`), 0o600)
	require.NoError(t, err)

	t.Run("Exit0", func(t *testing.T) {
		err = run("exit")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/ctr/main/exitcode")
		require.NoError(t, err)
		assert.Equal(t, "0", string(data))
	})

	t.Run("Exit1", func(t *testing.T) {
		err = run("exit 1")
		assert.Equal(t, 1, err.(errors.Exited).ExitCode())
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/ctr/main/exitcode")
		require.NoError(t, err)
		assert.Equal(t, "1", string(data))
	})
	t.Run("Stdout", func(t *testing.T) {
		_ = os.Remove(varRunArgo + "/ctr/main/stdout")
		err = run("echo hello")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/ctr/main/stdout")
		require.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("Sub-process", func(t *testing.T) {
		_ = os.Remove(varRunArgo + "/ctr/main/stdout")
		err = run(`(sleep 60; echo 'should not wait for sub-process')& echo "hello\c"`)
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/ctr/main/stdout")
		require.NoError(t, err)
		assert.Equal(t, "hello", string(data))
	})
	t.Run("Combined", func(t *testing.T) {
		err = run("echo hello > /dev/stderr")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/ctr/main/combined")
		require.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("Signal", func(t *testing.T) {
		for signal := range map[syscall.Signal]string{
			syscall.SIGTERM: "terminated",
			syscall.SIGKILL: "killed",
		} {
			err = os.WriteFile(varRunArgo+"/ctr/main/signal", []byte(strconv.Itoa(int(signal))), 0o600)
			require.NoError(t, err)
			var wg sync.WaitGroup
			wg.Go(func() {
				runErr := run("sleep 3")
				assert.EqualError(t, runErr, fmt.Sprintf("exit status %d", 128+signal))
			})
			wg.Wait()
		}
	})
	t.Run("Artifact", func(t *testing.T) {
		err = os.WriteFile(varRunArgo+"/template", []byte(`
{
	"outputs": {
		"artifacts": [
			{"path": "/tmp/artifact"}
		]
	}
}
`), 0o600)
		require.NoError(t, err)
		err = run("echo hello > /tmp/artifact")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		require.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
	t.Run("ArtifactWithTrailingAndLeadingSlash", func(t *testing.T) {
		err = os.WriteFile(varRunArgo+"/template", []byte(`
{
	"outputs": {
		"artifacts": [
			{"path": "/tmp/artifact/"}
		]
	}
}
`), 0o600)
		require.NoError(t, err)
		err = run("echo hello > /tmp/artifact")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		require.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
	t.Run("Parameter", func(t *testing.T) {
		err = os.WriteFile(varRunArgo+"/template", []byte(`
{
	"outputs": {
		"parameters": [
			{
				"valueFrom": {"path": "/tmp/parameter"}
			}
		]
	}
}
`), 0o600)
		require.NoError(t, err)
		err = run("echo hello > /tmp/parameter")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/outputs/parameters/tmp/parameter")
		require.NoError(t, err)
		assert.Contains(t, string(data), "hello")
	})
	t.Run("RetryContainerSetFail", func(t *testing.T) {
		err = os.WriteFile(varRunArgo+"/template", []byte(`
{
	"outputs": {
		"artifacts": [
			{
				"path": "/tmp/artifact/"
			}
		]
	},
	"containerSet": {
		"containers": [
			{	"name": "main"
			}
		],
		"retryStrategy":
		{
			"retries": 1
		}
	}
}
`), 0o600)
		require.NoError(t, err)
		_ = os.Remove("test.txt")
		err = run("sh ./test/containerSetRetryTest.sh /tmp/artifact")
		require.Error(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		require.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
	t.Run("RetryContainerSetSuccess", func(t *testing.T) {
		err = os.WriteFile(varRunArgo+"/template", []byte(`
{
	"outputs": {
		"artifacts": [
			{
				"path": "/tmp/artifact/"
			}
		]
	},
	"containerSet": {
		"containers": [
			{	"name": "main"
			}
		],
		"retryStrategy":
		{
			"retries": 2
		}
	}
}
`), 0o600)
		require.NoError(t, err)
		_ = os.Remove("test.txt")
		err = run("sh ./test/containerSetRetryTest.sh /tmp/artifact")
		require.NoError(t, err)
		var data []byte
		data, err = os.ReadFile(varRunArgo + "/outputs/artifacts/tmp/artifact.tgz")
		require.NoError(t, err)
		assert.NotEmpty(t, string(data)) // data is tgz format
	})
}

func run(script string) error {
	cmd := NewEmissaryCommand()
	ctx, _, err := cmdutil.ContextWithLogger(cmd, string(logging.Info), string(logging.Text))
	if err != nil {
		return err
	}
	return runEmissary(ctx, "main", newPodSource("main", true, append([]string{"sh", "-c"}, script)), maindriver.Container{})
}

const (
	podTraceParent = "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"
	podTraceID     = "0af7651916cd43dd8448eb211c80319c"
	podSpanID      = "b7ad6b7169203331"
)

func parseTraceParent(t *testing.T, traceParent string) trace.SpanContext {
	t.Helper()
	ctx := propagation.TraceContext{}.Extract(t.Context(), propagation.MapCarrier{"traceparent": traceParent})
	spanContext := trace.SpanContextFromContext(ctx)
	require.True(t, spanContext.IsValid(), "not a W3C traceparent: %q", traceParent)
	return spanContext
}

// The user's process must see the emissary's runMainContainer span, a child
// of the span the pod spec carries: same trace id, different span id.
func TestEmissary_ChildGetsRunMainContainerTraceParent(t *testing.T) {
	useTempVarRunArgo(t)
	require.NoError(t, os.WriteFile(varRunArgo+"/template", []byte(`{}`), 0o600))
	t.Setenv("TRACEPARENT", podTraceParent)

	require.NoError(t, run("echo -n \"$TRACEPARENT\""))

	data, err := os.ReadFile(varRunArgo + "/ctr/main/stdout")
	require.NoError(t, err)
	got := parseTraceParent(t, string(data))
	assert.Equal(t, podTraceID, got.TraceID().String(), "trace id must be the pod's")
	assert.NotEqual(t, podSpanID, got.SpanID().String(), "span id must be runMainContainer's, not the pod's")
}

// neverRunDriver fails the test if the harness reaches the driver.
type neverRunDriver struct{ t *testing.T }

func (d neverRunDriver) Run(context.Context, maindriver.Task, maindriver.ResultSink) (int, error) {
	d.t.Fatal("driver must not run")
	return 0, nil
}

type taskList []maindriver.Task

func (l *taskList) Next(context.Context) (maindriver.Task, bool, error) {
	if len(*l) == 0 {
		return maindriver.Task{}, false, nil
	}
	task := (*l)[0]
	*l = (*l)[1:]
	return task, true, nil
}

func TestEmissary_TasksShareRootSpan(t *testing.T) {
	useTempVarRunArgo(t)
	ctx := logging.TestContext(t.Context())
	t.Setenv("TRACEPARENT", podTraceParent)
	task := maindriver.Task{
		Template:            &wfv1.Template{},
		Command:             []string{"sh", "-c", `echo "$TRACEPARENT"`},
		Env:                 os.Environ(),
		IncludeScriptOutput: true,
		WorkDir:             varRunArgo + "/ctr/main",
	}
	source := &taskList{task, task}

	require.NoError(t, runEmissary(ctx, "main", source, maindriver.Container{}))

	data, err := os.ReadFile(varRunArgo + "/ctr/main/stdout")
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 2)
	var spanIDs []trace.SpanID
	for _, line := range lines {
		got := parseTraceParent(t, line)
		assert.Equal(t, podTraceID, got.TraceID().String(), "trace id must be the pod's")
		assert.NotEqual(t, podSpanID, got.SpanID().String(), "span id must not be the pod's")
		spanIDs = append(spanIDs, got.SpanID())
	}
	assert.Equal(t, spanIDs[0], spanIDs[1], "tasks run in the root span")
}

// A source with no task must fail the emissary rather than run an empty
// command.
func TestEmissary_ExhaustedSource(t *testing.T) {
	useTempVarRunArgo(t)
	require.NoError(t, os.WriteFile(varRunArgo+"/template", []byte(`{}`), 0o600))
	ctx := logging.TestContext(t.Context())
	source := newPodSource("main", false, []string{"true"})
	_, ok, err := source.Next(ctx)
	require.NoError(t, err)
	require.True(t, ok)

	err = runEmissary(ctx, "main", source, neverRunDriver{t})

	require.EqualError(t, err, "no task to run")
	data, err := os.ReadFile(varRunArgo + "/ctr/main/exitcode")
	require.NoError(t, err)
	assert.Equal(t, "64", string(data))
}
