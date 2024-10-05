package v1alpha1

// the workflow's phase
type WorkflowPhase string

const (
	WorkflowUnknown   WorkflowPhase = ""
	WorkflowPending   WorkflowPhase = "Pending" // pending some set-up - rarely used
	WorkflowRunning   WorkflowPhase = "Running" // any node has started; pods might not be running yet, the workflow maybe suspended too
	WorkflowSucceeded WorkflowPhase = "Succeeded"
	WorkflowFailed    WorkflowPhase = "Failed" // it maybe that the workflow was terminated
	WorkflowError     WorkflowPhase = "Error"
	WorkflowCancelled WorkflowPhase = "Cancelled"
)

func (p WorkflowPhase) Completed() bool {
	switch p {
	case WorkflowSucceeded, WorkflowFailed, WorkflowError, WorkflowCancelled:
		return true
	default:
		return false
	}
}
