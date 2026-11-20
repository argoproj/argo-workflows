package commands

import (
	"time"

	"github.com/spf13/cobra"

	argoexecexecutor "github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
)

// How main's emissary watches the supervisor's status marker (the writer
// and the protocol live in workflow/executor).
const (
	// supervisorHeartbeatTimeout is how long main tolerates a status marker that
	// has neither appeared nor advanced before presuming the supervisor dead.
	// Generously larger than the heartbeat interval so a GC pause or CPU
	// starvation in the supervisor doesn't trigger a false-positive death.
	supervisorHeartbeatTimeout = 30 * time.Second
	// supervisorStatusPollInterval is how often main re-checks the marker's
	// freshness. inotify delivers the terminal READY/FAILED write promptly; this
	// poll exists only to notice the *absence* of heartbeats, which inotify can't.
	supervisorStatusPollInterval = 2 * time.Second
)

// NewSupervisorCommand is the init-less auxiliary container: it prepares
// main, signals readiness, then collects outputs.
func NewSupervisorCommand() *cobra.Command {
	command := cobra.Command{
		Use:   "supervisor",
		Short: "init-less auxiliary: prepare main, signal readiness, then collect outputs",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Returned unwrapped: main() reads the exit code with a type assertion.
			return argoexecexecutor.RunHarness(cmd.Context(), clientConfig, varRunArgo, wfexecutor.SupervisorRole)
		},
	}
	return &command
}
