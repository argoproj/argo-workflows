package e2e

import "github.com/argoproj/argo/cmd/argo/commands"

type When struct {
	given *Given
}

func (w *When) SubmitWorkflow() *When {
	commands.SubmitWorkflows([]string{w.given.file}, nil, nil)
	return w
}

func (w *When) WaitForWorkflow() *When {
	commands.WaitWorkflows([]string{w.given.workflowName}, false, true)
	return w
}

func (w *When) DeleteWorkflow() {
	commands.DeleteWorkflow(w.given.workflowName)
}
