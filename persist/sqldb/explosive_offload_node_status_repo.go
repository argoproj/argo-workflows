package sqldb

import (
	"fmt"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var ExplosiveOffloadNodeStatusRepo OffloadNodeStatusRepo = &explosiveOffloadNodeStatusRepo{}

type explosiveOffloadNodeStatusRepo struct {
}

func (n *explosiveOffloadNodeStatusRepo) IsEnabled() bool {
	return false
}

func (n *explosiveOffloadNodeStatusRepo) Save(string, string, wfv1.Nodes) (string, error) {
	return "", fmt.Errorf("offload node status not supported")
}

func (n *explosiveOffloadNodeStatusRepo) Get(string, string, string) (wfv1.Nodes, error) {
	return nil, fmt.Errorf("offload node status not supported")
}

func (n *explosiveOffloadNodeStatusRepo) List(string) (map[PrimaryKey]wfv1.Nodes, error) {
	return nil, fmt.Errorf("offload node status not supported")
}

func (n *explosiveOffloadNodeStatusRepo) Delete(string, string) error {
	return fmt.Errorf("offload node status disabled")
}
