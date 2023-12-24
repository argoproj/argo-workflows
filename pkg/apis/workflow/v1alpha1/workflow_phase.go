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
	WorkflowCanceled  WorkflowPhase = "Canceled" // it is an intermediate state when enable failFast. Workflow phase will be changed from Canceled to Succeeded/Failed/Error
)

func (p WorkflowPhase) Completed() bool {
	switch p {
	case WorkflowSucceeded, WorkflowFailed, WorkflowError:
		return true
	default:
		return false
	}
}

func (p WorkflowPhase) Canceled() bool {
	return p == WorkflowCanceled
}
