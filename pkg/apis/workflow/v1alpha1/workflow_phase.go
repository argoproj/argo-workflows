package v1alpha1

// the workflow's phase
type WorkflowPhase string

const (
	WorkflowUnknown    WorkflowPhase = ""
	WorkflowPending    WorkflowPhase = "Pending" // pending some set-up - rarely used
	WorkflowRunning    WorkflowPhase = "Running" // any node has started; pods might not be running yet, the workflow maybe suspended too
	WorkflowSucceeded  WorkflowPhase = "Succeeded"
	WorkflowFailed     WorkflowPhase = "Failed"     // it maybe that the the workflow was terminated
	WorkflowTerminated WorkflowPhase = "Terminated" // it maybe that the the workflow was terminated by human
	WorkflowError      WorkflowPhase = "Error"
)

func (p WorkflowPhase) Completed() bool {
	switch p {
	case WorkflowSucceeded, WorkflowFailed, WorkflowTerminated, WorkflowError:
		return true
	default:
		return false
	}
}
