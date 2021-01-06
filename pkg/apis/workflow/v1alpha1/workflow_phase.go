package v1alpha1

type WorkflowPhase string

const (
	WorkflowUnknown   WorkflowPhase = ""        // the workflow has yet to be reconciled
	WorkflowPending   WorkflowPhase = "Pending" // pending some set-up - rarely used
	WorkflowRunning   WorkflowPhase = "Running" // any node has started; pods might not be running yet, the workflow maybe suspended too
	WorkflowSucceeded WorkflowPhase = "Succeeded"
	WorkflowFailed    WorkflowPhase = "Failed" // it maybe that the the workflow was terminated
	WorkflowError     WorkflowPhase = "Error"
)
