package executor

import (
	"path"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
)

// TaskPaths is where one task's files live. A pod runs one task, so its
// paths are the fixed locations the controller mounts (PodPaths); a pool
// pod gives each task its own directory beneath each of the fixed
// locations (PoolTask). Paths outside TaskPaths are still
// shared between tasks: the template's own input and output paths, the
// resource manifest, and, outside a pool pod, the emissary's large-argument
// files.
type TaskPaths struct {
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

// PoolTask returns the paths of one task in a hot worker pool pod whose
// shared directories are p's (PodPaths in a pool pod), where the worker
// stages tasks for the looping main container one after another through
// the directories the controller mounts in both: each task's files are in
// a directory of its own, named task, beneath each of them (poolShared).
// Its VarRunArgo is also the task's directory in the pool pod's file
// protocol (poolpod), in the pool directory. The worker removes each
// directory once the task is done, so task must name a single
// directory (poolpod.CheckTaskName).
func (p TaskPaths) PoolTask(task string) (TaskPaths, error) {
	if err := poolpod.CheckTaskName(task); err != nil {
		return TaskPaths{}, err
	}
	shared := p.poolShared()
	return TaskPaths{
		VarRunArgo:     filepath.Join(shared.VarRunArgo, task),
		InputArtifacts: filepath.Join(shared.InputArtifacts, task),
		Staging:        filepath.Join(shared.Staging, task),
		MainFS:         p.MainFS,
		Scratch:        filepath.Join(shared.Scratch, task),
	}, nil
}

// poolShared returns the directories of a pool pod whose paths are p's
// that each of its tasks has a directory of its own beneath (PoolTask): the
// pool directory, the input artifacts and staging directories, and
// Scratch's outputs subdirectory, which a pool pod shares with its artifact
// plugin sidecars, as a pod's /tmp/argo/outputs, so the tarballs Collect
// writes to the task's Scratch/outputs are where the plugins can read them.
func (p TaskPaths) poolShared() TaskPaths {
	return TaskPaths{
		VarRunArgo:     filepath.Join(p.VarRunArgo, poolpod.DirName),
		InputArtifacts: filepath.Clean(p.InputArtifacts),
		Staging:        filepath.Clean(p.Staging),
		MainFS:         p.MainFS,
		Scratch:        filepath.Join(p.Scratch, "outputs"),
	}
}

// PoolDir is the pool directory of a pool pod whose paths are p: where
// the worker offers tasks and main takes them.
func (p TaskPaths) PoolDir() string {
	return p.poolShared().VarRunArgo
}

// taskPool returns the shared directories (poolShared) of the pool pod
// that p, a task's paths (PoolTask), are beneath.
func (p TaskPaths) taskPool() TaskPaths {
	return TaskPaths{
		VarRunArgo:     filepath.Dir(p.VarRunArgo),
		InputArtifacts: filepath.Dir(p.InputArtifacts),
		Staging:        filepath.Dir(p.Staging),
		MainFS:         p.MainFS,
		Scratch:        filepath.Dir(p.Scratch),
	}
}

// poolDirs lists p's VarRunArgo, InputArtifacts, Staging and Scratch. For
// a pool pod's shared directories (poolShared) they are the directories its
// tasks each have one of their own beneath, the pool directory first; for
// one of its tasks, the task's own.
func (p TaskPaths) poolDirs() []string {
	return []string{p.VarRunArgo, p.InputArtifacts, p.Staging, p.Scratch}
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
