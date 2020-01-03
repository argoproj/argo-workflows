package sqldb

import (
	"fmt"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var NullWorkflowAchive = &nullWorkflowAchieve{}

type nullWorkflowAchieve struct {
}

func (r *nullWorkflowAchieve) ArchiveWorkflow(*wfv1.Workflow) error {
	return nil
}

func (r *nullWorkflowAchieve) ListWorkflows(string, int, int) ([]wfv1.Workflow, error) {
	return []wfv1.Workflow{}, nil
}

func (r *nullWorkflowAchieve) GetWorkflow(string, string) (*wfv1.Workflow, error) {
	return nil, fmt.Errorf("getting archived workflows not supported")
}

func (r *nullWorkflowAchieve) DeleteWorkflow(namespace string, uid string) error {
	return fmt.Errorf("deleting archived workflows not supported")
}
