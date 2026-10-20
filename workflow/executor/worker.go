package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argoerrors "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

// workerSupports reports whether a worker can run tmpl in-process: a
// container or script template whose command is spelled out, with no
// volumes, sidecars, env from the cluster or $(VAR) references (the
// kubelet expands those; the worker does not), and only absolute input and
// output paths (a relative one would resolve against the worker's own
// directory, not main's). The error names the first thing that is not
// supported.
func workerSupports(tmpl *wfv1.Template) error {
	switch tmpl.GetType() {
	case wfv1.TemplateTypeContainer, wfv1.TemplateTypeScript:
	case wfv1.TemplateTypeContainerSet:
		return errors.New("a worker cannot run a containerSet template")
	default:
		return fmt.Errorf("a worker cannot run a %s template", strings.ToLower(string(tmpl.GetType())))
	}
	c := mainContainer(tmpl)
	if len(c.Command) == 0 {
		return errors.New("a worker needs the template's command spelled out; it cannot look up an image's entrypoint")
	}
	if len(c.EnvFrom) > 0 {
		return errors.New("a worker cannot resolve envFrom")
	}
	for _, e := range c.Env {
		if e.ValueFrom != nil {
			return fmt.Errorf("a worker cannot resolve env var %s from valueFrom", e.Name)
		}
	}
	for _, e := range c.Env {
		ref := "$(" + e.Name + ")"
		for _, s := range append(append(append([]string{}, c.Command...), c.Args...), envValues(c.Env)...) {
			if strings.Contains(s, ref) {
				return fmt.Errorf("a worker does not expand %s", ref)
			}
		}
	}
	if len(c.VolumeMounts) > 0 {
		return errors.New("a worker cannot provide volumeMounts")
	}
	if len(tmpl.Volumes) > 0 {
		return errors.New("a worker cannot provide volumes")
	}
	if len(tmpl.Sidecars) > 0 {
		return errors.New("a worker cannot run sidecars")
	}
	for _, path := range templatePaths(tmpl) {
		if !filepath.IsAbs(path) {
			return fmt.Errorf("a worker needs absolute input and output paths, not %q", path)
		}
	}
	return nil
}

func envValues(env []apiv1.EnvVar) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		out = append(out, e.Value)
	}
	return out
}

// templatePaths lists the input artifact and output paths a template
// declares.
func templatePaths(tmpl *wfv1.Template) []string {
	var paths []string
	for _, a := range tmpl.Inputs.Artifacts {
		if a.Path != "" {
			paths = append(paths, a.Path)
		}
	}
	for _, p := range tmpl.Outputs.Parameters {
		if p.ValueFrom != nil && p.ValueFrom.Path != "" {
			paths = append(paths, p.ValueFrom.Path)
		}
	}
	for _, a := range tmpl.Outputs.Artifacts {
		if a.Path != "" {
			paths = append(paths, a.Path)
		}
	}
	return paths
}

// checkWorker fails the task before any work if the worker cannot run it:
// no driver, an unsupported template, or an input artifact path where
// linking would replace something. The worker's filesystem is the host's,
// not a throwaway container root, so nothing already there may be removed
// (a symlink of ours from an earlier attempt is the only exception), and
// no input may be linked over the task's own root.
func (we *WorkflowExecutor) checkWorker() error {
	if we.mainDriver == nil {
		return errors.New("the worker has no driver to run main")
	}
	if err := workerSupports(&we.Template); err != nil {
		return err
	}
	for _, art := range we.Template.Inputs.Artifacts {
		if art.Path == "" {
			continue
		}
		if rel, err := filepath.Rel(art.Path, we.paths.Root); err == nil && filepath.IsLocal(rel) || rel == "." {
			return fmt.Errorf("input artifact %s path %s contains the task's own files", art.Name, art.Path)
		}
		if fi, err := os.Lstat(art.Path); err == nil && fi.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("input artifact %s path %s already exists; a worker will not replace it", art.Name, art.Path)
		}
	}
	return nil
}

// createTaskDirs empties and recreates the task's own tree.
func (we *WorkflowExecutor) createTaskDirs() error {
	if we.paths.Root == "" {
		return errors.New("the task has no root of its own to create")
	}
	if err := os.RemoveAll(we.paths.Root); err != nil {
		return fmt.Errorf("failed to clear task root: %w", err)
	}
	for _, dir := range []string{we.paths.ContainerDir(common.MainContainerName), we.paths.InputArtifacts, we.paths.Staging, we.paths.Scratch} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}
	return nil
}

// mainContainer is the container a container or script template runs.
func mainContainer(tmpl *wfv1.Template) *apiv1.Container {
	if tmpl.Script != nil {
		return &tmpl.Script.Container
	}
	return tmpl.Container
}

// workerCommand is the command main runs, as the controller would give it
// to main's container: the container's command and args, plus the staged
// source for a script that has one.
func workerCommand(tmpl *wfv1.Template, paths TaskPaths) []string {
	c := mainContainer(tmpl)
	command := append(append([]string{}, c.Command...), c.Args...)
	if tmpl.Script != nil && tmpl.Script.Source != "" {
		command = append(command, paths.ScriptSource())
	}
	return command
}

// runMain runs the task's main command in this process through the
// executor's driver, staging its outputs where Collect reads them. A
// non-zero exit is the task's failure, not an executor error: it is
// reported to the sink and Collect still runs, as in a pod.
func (we *WorkflowExecutor) runMain(ctx context.Context) error {
	if we.mainDriver == nil {
		return errors.New("no driver to run main in-process")
	}
	if err := linkInputArtifacts(ctx, we.paths.InputArtifacts, &we.Template); err != nil {
		return err
	}
	c := mainContainer(&we.Template)
	// Not slices.Clone: a nil env would make the child inherit the
	// worker's whole environment.
	env := append([]string{}, we.environ...)
	for _, e := range c.Env {
		env = append(env, e.Name+"="+e.Value)
	}
	var watch *deadlineWatch
	stopWatch := func() {
		if watch != nil {
			watch.stop()
		}
	}
	// Deferred too, so a panic after main starts still ends the watch.
	defer stopWatch()
	task := maindriver.Task{
		NodeID:              we.nodeID,
		Template:            &we.Template,
		Command:             workerCommand(&we.Template, we.paths),
		Env:                 env,
		IncludeScriptOutput: we.IncludeScriptOutput,
		Dir:                 c.WorkingDir,
		WorkDir:             we.paths.ContainerDir(common.MainContainerName),
		OnStart: func(pid int) {
			watch = we.watchDeadline(ctx, pid)
		},
		// Main has been reaped: its pid must not be signalled again, and a
		// deadline passing while outputs are staged is not main's fault.
		OnExit: stopWatch,
	}
	stager := maindriver.PodStager{VarRunArgo: we.paths.VarRunArgo, ContainerName: common.MainContainerName, Template: &we.Template}
	_, err := we.mainDriver.Run(ctx, task, stager)
	stopWatch()
	// Reported on a context the task's end can't cancel, so a "worker
	// stopped" failure is still recorded.
	reportCtx := context.WithoutCancel(ctx)
	if watch != nil && watch.reason != "" {
		we.sink.Fail(reportCtx, errors.New(watch.reason))
		return nil
	}
	var exited argoerrors.Exited
	if errors.As(err, &exited) {
		we.sink.Fail(reportCtx, err)
		return nil
	}
	return err
}

// deadlineWatch stops main if the task's deadline passes or the worker
// stops. reason says why, once stop has returned; it is empty if main
// exited first.
type deadlineWatch struct {
	reason string
	done   chan struct{}
	exited chan struct{}
	once   sync.Once
}

// stop ends the watch and waits for it, so main is never signalled after
// it returns.
func (w *deadlineWatch) stop() {
	w.once.Do(func() { close(w.done) })
	<-w.exited
}

// watchDeadline watches main (pid) until stopped. When the deadline passes
// or ctx ends, it records why, then sends main SIGTERM and, after the
// grace period, SIGKILL. It never reports to the sink itself: reporting can
// retry for a long time, and must not delay stopping main.
func (we *WorkflowExecutor) watchDeadline(ctx context.Context, pid int) *deadlineWatch {
	w := &deadlineWatch{done: make(chan struct{}), exited: make(chan struct{})}
	go func() {
		defer close(w.exited)
		var deadline <-chan time.Time
		if !we.Deadline.IsZero() {
			t := time.NewTimer(time.Until(we.Deadline))
			defer t.Stop()
			deadline = t.C
		}
		select {
		case <-w.done:
			return
		case <-deadline:
			w.reason = deadlineExceededMessage
		case <-ctx.Done():
			w.reason = "worker stopped"
		}
		_ = osspecific.Kill(pid, syscall.SIGTERM)
		select {
		case <-w.done:
		case <-time.After(we.terminationGracePeriod):
			_ = osspecific.Kill(pid, syscall.SIGKILL)
		}
	}()
	return w
}

// removeTaskRoot removes the task's root and the input-artifact symlinks
// runMain made (a link at the artifact's path pointing at exactly the
// file Prepare loaded for it). checkWorker refused to link over anything
// else, and declared output paths are the user's, so those stay.
func (we *WorkflowExecutor) removeTaskRoot() error {
	if we.paths.Root == "" {
		return nil
	}
	var errs []error
	for _, art := range we.Template.Inputs.Artifacts {
		if art.Path == "" {
			continue
		}
		if target, err := os.Readlink(art.Path); err == nil && target == filepath.Join(we.paths.InputArtifacts, art.Name) {
			if err := os.Remove(art.Path); err != nil {
				errs = append(errs, fmt.Errorf("failed to remove input artifact link %s: %w", art.Path, err))
			}
		}
	}
	if err := os.RemoveAll(we.paths.Root); err != nil {
		errs = append(errs, fmt.Errorf("failed to remove task root: %w", err))
	}
	return errors.Join(errs...)
}
