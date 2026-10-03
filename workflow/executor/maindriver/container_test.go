//go:build !windows

package maindriver

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type recordingSink []Output

func (s *recordingSink) Put(_ context.Context, _ string, out Output) error {
	*s = append(*s, out)
	return nil
}

func TestContainerRun(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	dir := t.TempDir()
	param := filepath.Join(dir, "param")
	tmpl := &wfv1.Template{Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "p", ValueFrom: &wfv1.ValueFrom{Path: param}}}}}
	var sink recordingSink
	var pid int

	code, err := Container{}.Run(ctx, Task{
		NodeID:              "n1",
		Template:            tmpl,
		Command:             []string{"sh", "-c", "echo hi; echo v > " + param + "; exit 3"},
		IncludeScriptOutput: true,
		WorkDir:             dir,
		OnStart:             func(p int) { pid = p },
	}, &sink)

	require.Error(t, err)
	assert.Equal(t, 3, code)
	assert.NotZero(t, pid)
	stdout, err := os.ReadFile(filepath.Join(dir, "stdout"))
	require.NoError(t, err)
	assert.Equal(t, "hi\n", string(stdout))
	assert.Equal(t, recordingSink{
		{Kind: OutputStdout, Path: filepath.Join(dir, "stdout")},
		{Kind: OutputCombined, Path: filepath.Join(dir, "combined")},
		{Kind: OutputParameter, Path: param},
	}, sink)
}

func TestPodSourceDriverSink(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	varRunArgo := t.TempDir()
	param := filepath.Join(t.TempDir(), "param")
	require.NoError(t, os.MkdirAll(filepath.Join(varRunArgo, "ctr", "main"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(varRunArgo, "template"),
		[]byte(`{"outputs":{"parameters":[{"name":"p","valueFrom":{"path":"`+param+`"}}]}}`), 0o644))
	argsFile := filepath.Join(t.TempDir(), "args.json")
	require.NoError(t, os.WriteFile(argsFile, []byte(`["echo v > `+param+`"]`), 0o644))

	src := &PodSource{VarRunArgo: varRunArgo, ArgsFile: argsFile, NodeID: "n1", ContainerName: "main", Command: []string{"sh", "-c"}}
	task, ok, err := src.Next(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, []string{"sh", "-c", "echo v > " + param}, task.Command)
	assert.Equal(t, filepath.Join(varRunArgo, "ctr", "main"), task.WorkDir)

	code, err := Container{}.Run(ctx, task, PodSink{VarRunArgo: varRunArgo, ContainerName: "main", Template: task.Template})
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	staged, err := os.ReadFile(varRunArgo + "/outputs/parameters/" + param)
	require.NoError(t, err)
	assert.Equal(t, "v\n", string(staged))

	_, ok, err = src.Next(ctx)
	require.NoError(t, err)
	assert.False(t, ok)
}

type failingSink struct{ err error }

func (s failingSink) Put(context.Context, string, Output) error { return s.err }

// A handover failure must not hide the command's exit code.
func TestContainerRun_SinkErrorKeepsExitCode(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := &wfv1.Template{Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "p", ValueFrom: &wfv1.ValueFrom{Path: "/nonexistent"}}}}}
	sinkErr := errors.New("sink down")

	code, err := Container{}.Run(ctx, Task{Template: tmpl, Command: []string{"sh", "-c", "exit 7"}}, failingSink{sinkErr})

	assert.Equal(t, 7, code)
	require.ErrorIs(t, err, sinkErr)
}

func TestContainerRun_RejectsBadTask(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	var sink recordingSink

	_, err := Container{}.Run(ctx, Task{Command: []string{"true"}}, &sink)
	require.EqualError(t, err, "task has no template")

	_, err = Container{}.Run(ctx, Task{Template: &wfv1.Template{}, Command: []string{"true"}, IncludeScriptOutput: true}, &sink)
	require.EqualError(t, err, "task has no workdir to capture logs in")

	assert.Empty(t, sink)
}

func TestContainerRun_Env(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	dir := t.TempDir()
	t.Setenv("MAINDRIVER_TEST_INHERITED", "yes")

	run := func(env []string) string {
		t.Helper()
		var sink recordingSink
		code, err := Container{}.Run(ctx, Task{
			Template:            &wfv1.Template{},
			Command:             []string{"sh", "-c", "echo \"$MAINDRIVER_TEST_INHERITED/$MAINDRIVER_TEST_SET\""},
			Env:                 env,
			IncludeScriptOutput: true,
			WorkDir:             dir,
		}, &sink)
		require.NoError(t, err)
		require.Equal(t, 0, code)
		out, err := os.ReadFile(filepath.Join(dir, "stdout"))
		require.NoError(t, err)
		require.NoError(t, os.Remove(filepath.Join(dir, "stdout")))
		return string(out)
	}

	assert.Equal(t, "yes/\n", run(nil), "nil Env inherits")
	assert.Equal(t, "/set\n", run([]string{"MAINDRIVER_TEST_SET=set"}), "explicit Env replaces")
}

func TestContainerRun_ArchivedLogsCapture(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	dir := t.TempDir()
	tmpl := &wfv1.Template{ArchiveLocation: &wfv1.ArtifactLocation{ArchiveLogs: new(true)}}
	var sink recordingSink

	code, err := Container{}.Run(ctx, Task{Template: tmpl, Command: []string{"sh", "-c", "echo out; echo err >&2"}, WorkDir: dir}, &sink)

	require.NoError(t, err)
	assert.Equal(t, 0, code)
	combined, err := os.ReadFile(filepath.Join(dir, "combined"))
	require.NoError(t, err)
	assert.Contains(t, string(combined), "out\n")
	assert.Contains(t, string(combined), "err\n")
	assert.Equal(t, recordingSink{
		{Kind: OutputStdout, Path: filepath.Join(dir, "stdout")},
		{Kind: OutputCombined, Path: filepath.Join(dir, "combined")},
	}, sink)
}
