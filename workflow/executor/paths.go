package executor

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// TaskPaths is where one task's files live. A pod runs one task, so its
// paths are the fixed locations the controller mounts (PodPaths); a worker
// running several tasks in one process gives each its own tree
// (WorkerPaths). Paths outside TaskPaths are still shared between tasks:
// the template's own input and output paths, the resource manifest, and
// the emissary's large-argument files.
type TaskPaths struct {
	// Root is the directory holding all of the task's own files. A worker
	// empties it before the task and removes it after, so it must never
	// hold anything else. Empty for a pod, which has nothing to remove.
	Root string
	// VarRunArgo is the directory shared between main and its auxiliary
	// containers: template, staged outputs, per-container logs and exit
	// codes.
	VarRunArgo string
	// InputArtifacts is where Prepare loads input artifacts.
	InputArtifacts string
	// Staging holds files Prepare writes for main, such as script source.
	Staging string
	// MainFS is the prefix through which an auxiliary container reaches
	// main's volume mounts. Empty when main shares this filesystem.
	MainFS string
	// Scratch is Collect's temporary space for tarballs and logs. In a pod
	// its outputs/ subdirectory is also mounted into artifact plugin
	// sidecars, so the pod value must match the controller's mount.
	Scratch string
}

// PodPaths returns the paths of the classic one-task pod layout.
func PodPaths() TaskPaths {
	return TaskPaths{
		VarRunArgo:     common.VarRunArgoPath,
		InputArtifacts: common.ExecutorArtifactBaseDir,
		Staging:        common.ExecutorStagingEmptyDir,
		MainFS:         common.ExecutorMainFilesystemDir,
		Scratch:        "/tmp/argo",
	}
}

// WorkerPaths returns the paths of one task run in-process by a worker,
// all under root/nodeID, made absolute so symlinks into it resolve from
// anywhere. The task root is deleted once the task is done, so nodeID must
// name exactly one directory directly under root.
func WorkerPaths(root, nodeID string) (TaskPaths, error) {
	if nodeID == "" || nodeID == "." || nodeID == ".." || strings.ContainsAny(nodeID, `/\`) {
		return TaskPaths{}, fmt.Errorf("node ID %q cannot name a task directory", nodeID)
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return TaskPaths{}, fmt.Errorf("failed to resolve worker root: %w", err)
	}
	taskRoot := filepath.Join(root, nodeID)
	return TaskPaths{
		Root:           taskRoot,
		VarRunArgo:     filepath.Join(taskRoot, "run"),
		InputArtifacts: filepath.Join(taskRoot, "inputs"),
		Staging:        filepath.Join(taskRoot, "staging"),
		Scratch:        filepath.Join(taskRoot, "scratch"),
	}, nil
}

// ContainerDir is where a container's exit code, logs and signal file live.
func (p TaskPaths) ContainerDir(name string) string {
	return filepath.Join(p.VarRunArgo, "ctr", name)
}

// ScriptSource is where Prepare writes a script template's source. It is
// joined with forward slashes, like the paths the controller hands main.
func (p TaskPaths) ScriptSource() string {
	return path.Join(p.Staging, "script")
}
