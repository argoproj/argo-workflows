package executor

import (
	"path"
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
	pool := filepath.Join(common.VarRunArgoPath, poolpod.DirName)
	assert.Equal(t, TaskPaths{
		VarRunArgo:     filepath.Join(pool, "wf-1"),
		InputArtifacts: filepath.Join(common.ExecutorArtifactBaseDir, "wf-1"),
		Staging:        filepath.Join(common.ExecutorStagingEmptyDir, "wf-1"),
		MainFS:         common.ExecutorMainFilesystemDir,
		Scratch:        filepath.Join("/tmp/argo/outputs", "wf-1"),
	}, p)
	// Joined with a forward slash, as ScriptSource always is.
	assert.Equal(t, path.Join(filepath.Join(common.ExecutorStagingEmptyDir, "wf-1"), "script"), p.ScriptSource())
	assert.Equal(t, filepath.Join(pool, "wf-1", "ctr", common.MainContainerName), p.ContainerDir(common.MainContainerName))
	assert.Equal(t, PodPaths().poolShared(), p.taskPool(), "a task's pool is the one it is beneath")
	assert.Equal(t, pool, PodPaths().PoolDir(), "the pool directory main and the worker share")
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
