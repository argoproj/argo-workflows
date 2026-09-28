package k8s

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
)

type recordingSink []maindriver.Output

func (s *recordingSink) Put(_ context.Context, _ string, out maindriver.Output) error {
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

	code, err := Container{}.Run(ctx, maindriver.Task{
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
		{Kind: maindriver.OutputStdout, Path: filepath.Join(dir, "stdout")},
		{Kind: maindriver.OutputCombined, Path: filepath.Join(dir, "combined")},
		{Kind: maindriver.OutputParameter, Path: param},
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
