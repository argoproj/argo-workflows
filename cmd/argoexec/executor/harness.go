package executor

import (
	"context"
	"time"

	"github.com/argoproj/pkg/stats"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
)

// PodSource yields the one task a classic pod runs, described by the pod's
// environment, then is exhausted.
type PodSource struct {
	// VarRunArgo is the pod's shared directory.
	VarRunArgo string
	done       bool
}

var _ executor.TaskSource = &PodSource{}

func (s *PodSource) Next(ctx context.Context) (executor.Task, bool, error) {
	if s.done {
		return executor.Task{}, false, nil
	}
	s.done = true
	cfg := TaskConfigFromEnv(s.VarRunArgo)
	logging.RequireLoggerFromContext(ctx).
		WithField("templateName", cfg.Template.Name).
		WithField("includeScriptOutput", cfg.IncludeScriptOutput).
		WithField("deadline", cfg.Deadline).
		Info(ctx, "Executor initialized")
	return executor.Task{Config: cfg, Paths: PodPaths(s.VarRunArgo)}, true, nil
}

// podSink returns the result sink for a pod's task: its WorkflowTaskResult,
// with failures reported through the termination log.
func podSink(process *executor.Process) func(executor.Task) executor.ResultSink {
	return func(task executor.Task) executor.ResultSink {
		return process.NewTaskResultSink(task.Config, false)
	}
}

// RunHarness runs the pod's task through role: the composition root for
// the init, wait and supervisor commands. The task's spans continue the
// trace context the controller gave the pod, which the task carries
// (executor.TaskConfig.TraceContext).
func RunHarness(ctx context.Context, clientConfig clientcmd.ClientConfig, varRunArgo string, role executor.Role) error {
	process := InitProcess(ctx, clientConfig)
	stopProcess := startProcess(ctx, process)
	defer stopProcess()
	h := &executor.Harness{
		Process: process,
		Source:  &PodSource{VarRunArgo: varRunArgo},
		Sink:    podSink(process),
		Role:    role,
	}
	return h.Run(ctx)
}

// startProcess starts logging process's stats every five minutes, and
// returns what stops it: the stats are logged once more and tracing shut
// down.
func startProcess(ctx context.Context, process *executor.Process) (stop func()) {
	stats.StartStatsTicker(5 * time.Minute)
	return func() {
		stats.LogStats()
		if err := process.Tracing.Shutdown(context.WithoutCancel(ctx)); err != nil {
			logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "Failed to shutdown tracing")
		}
	}
}
