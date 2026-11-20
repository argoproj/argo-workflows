package executor

import (
	"context"
	"time"

	"github.com/argoproj/pkg/stats"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
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
// the init, wait and supervisor commands.
func RunHarness(ctx context.Context, clientConfig clientcmd.ClientConfig, varRunArgo string, role executor.Role) error {
	ctx = tracing.InjectTraceContext(ctx)
	process := InitProcess(ctx, clientConfig)
	defer func() {
		if err := process.Tracing.Shutdown(context.WithoutCancel(ctx)); err != nil {
			logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "Failed to shutdown tracing")
		}
	}()
	stats.StartStatsTicker(5 * time.Minute)
	defer stats.LogStats()
	h := &executor.Harness{
		Process: process,
		Source:  &PodSource{VarRunArgo: varRunArgo},
		Sink:    podSink(process),
		Role:    role,
	}
	return h.Run(ctx)
}
