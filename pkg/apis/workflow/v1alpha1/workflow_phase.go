package v1alpha1

import (
	"encoding/json"
	"fmt"
)

// the workflow's phase
type WorkflowPhase int

const (
	WorkflowUnknown WorkflowPhase = iota
	WorkflowPending WorkflowPhase = iota // pending some set-up - rarely used
	WorkflowRunning                      // any node has started; pods might not be running yet, the workflow maybe suspended too
	WorkflowSucceeded
	WorkflowFailed // it maybe that the the workflow was terminated
	WorkflowError
)

func (p *WorkflowPhase) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	m, ok := map[string]WorkflowPhase{
		"Pending":   WorkflowPending,
		"Running":   WorkflowRunning,
		"Succeeded": WorkflowSucceeded,
		"Failed":    WorkflowFailed,
		"Error":     WorkflowError,
	}[j]
	if ok {
		*p = m
	} else {
		*p = WorkflowUnknown
	}
	return nil
}

func (p WorkflowPhase) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%v"`, p)), nil
}

func (i WorkflowPhase) OpenAPISchemaType() []string {
	return []string{"string"}
}

func (p WorkflowPhase) String() string {
	return map[WorkflowPhase]string{
		WorkflowUnknown:   "",
		WorkflowPending:   "Pending",
		WorkflowRunning:   "Running",
		WorkflowSucceeded: "Succeeded",
		WorkflowFailed:    "Failed",
		WorkflowError:     "Error",
	}[p]
}

func (p WorkflowPhase) Completed() bool {
	switch p {
	case WorkflowSucceeded, WorkflowFailed, WorkflowError:
		return true
	default:
		return false
	}
}
