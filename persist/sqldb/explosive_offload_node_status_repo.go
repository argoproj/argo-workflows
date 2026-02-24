package sqldb

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

var (
	ExplosiveOffloadNodeStatusRepo OffloadNodeStatusRepo = &explosiveOffloadNodeStatusRepo{}
	ErrOffloadNotSupported                               = fmt.Errorf("offload node status is not supported")
)

type explosiveOffloadNodeStatusRepo struct{}

func (n *explosiveOffloadNodeStatusRepo) IsEnabled() bool {
	return false
}

func (n *explosiveOffloadNodeStatusRepo) Save(context.Context, string, string, wfv1.Nodes) (string, error) {
	return "", ErrOffloadNotSupported
}

func (n *explosiveOffloadNodeStatusRepo) Get(context.Context, string, string) (wfv1.Nodes, error) {
	return nil, ErrOffloadNotSupported
}

func (n *explosiveOffloadNodeStatusRepo) List(context.Context, string) (map[UUIDVersion]wfv1.Nodes, error) {
	return nil, ErrOffloadNotSupported
}

func (n *explosiveOffloadNodeStatusRepo) Delete(context.Context, string, string) error {
	return ErrOffloadNotSupported
}

func (n *explosiveOffloadNodeStatusRepo) ListOldOffloads(context.Context, string) (map[string][]string, error) {
	return nil, ErrOffloadNotSupported
}
