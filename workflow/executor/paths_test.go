package executor

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
)

func TestPodPaths_MatchLegacyConstants(t *testing.T) {
	p := PodPaths()
	assert.Empty(t, p.Root)
	assert.Equal(t, common.VarRunArgoPath, p.VarRunArgo)
	assert.Equal(t, common.ExecutorArtifactBaseDir, p.InputArtifacts)
	assert.Equal(t, common.ExecutorStagingEmptyDir, p.Staging)
	assert.Equal(t, common.ExecutorScriptSourcePath, p.ScriptSource())
	assert.Equal(t, common.ExecutorMainFilesystemDir, p.MainFS)
	assert.Equal(t, "/tmp/argo", p.Scratch)
}

func TestWorkerPaths_AllUnderNodeRoot(t *testing.T) {
	p, err := WorkerPaths("w", "node-1")
	require.NoError(t, err)
	abs, err := filepath.Abs("w")
	require.NoError(t, err)
	root := filepath.Join(abs, "node-1")
	assert.Equal(t, TaskPaths{
		Root:           root,
		VarRunArgo:     filepath.Join(root, "run"),
		InputArtifacts: filepath.Join(root, "inputs"),
		Staging:        filepath.Join(root, "staging"),
		MainFS:         "",
		Scratch:        filepath.Join(root, "scratch"),
	}, p)
}

// The task root is deleted after each worker task, so a node ID that
// doesn't name a single child of the worker root must be refused.
func TestWorkerPaths_RejectsNodeIDOutsideItsOwnDir(t *testing.T) {
	for _, nodeID := range []string{"", ".", "..", "a/b", "../x", "/abs"} {
		_, err := WorkerPaths("/w", nodeID)
		assert.Error(t, err, "nodeID %q", nodeID)
	}
}

func TestPoolTask_ATaskDirectoryBeneathEachMount(t *testing.T) {
	p, err := PodPaths().PoolTask("wf-1")
	require.NoError(t, err)
	assert.Equal(t, TaskPaths{
		VarRunArgo:     poolpod.Dir + "/wf-1",
		InputArtifacts: common.ExecutorArtifactBaseDir + "/wf-1",
		Staging:        common.ExecutorStagingEmptyDir + "/wf-1",
		MainFS:         common.ExecutorMainFilesystemDir,
		Scratch:        "/tmp/argo/outputs/wf-1",
	}, p)
	assert.Equal(t, common.ExecutorStagingEmptyDir+"/wf-1/script", p.ScriptSource())
	assert.Equal(t, poolpod.Dir+"/wf-1/ctr/main", p.ContainerDir(common.MainContainerName))
	assert.Equal(t, poolpod.Dir, PodPaths().PoolDir(), "the pool directory main and the worker share")
}

// The worker removes each task directory when the task is done, and the
// pool directory holds the protocol's own files, so a task name must name
// a directory of its own.
func TestPoolTask_RejectsANameThatIsNotItsOwnDirectory(t *testing.T) {
	for _, task := range []string{"", ".", "..", "a/b", "../x", poolpod.CurrentFile, ".current-1"} {
		_, err := PodPaths().PoolTask(task)
		assert.Error(t, err, "task %q", task)
	}
}
