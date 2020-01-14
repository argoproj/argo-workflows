package sqldb

import (
	"fmt"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var ExplosiveOffloadNodeStatusRepo = &explosiveOffloadNodeStatusRepo{}

type explosiveOffloadNodeStatusRepo struct {
}

func (n *explosiveOffloadNodeStatusRepo) IsEnabled() bool {
	return false
}

func (n *explosiveOffloadNodeStatusRepo) Save(*v1alpha1.Workflow) error {
	return fmt.Errorf("offload node status not supported")
}

func (n *explosiveOffloadNodeStatusRepo) Get(string, string, string) (*v1alpha1.Workflow, error) {
	return nil, fmt.Errorf("offload node status not supported")
}

func (n *explosiveOffloadNodeStatusRepo) List(string) (v1alpha1.Workflows, error) {
	return nil, fmt.Errorf("offload node status not supported")
}

func (n *explosiveOffloadNodeStatusRepo) Delete(string, string) error {
	return fmt.Errorf("offload node status disabled")
}
