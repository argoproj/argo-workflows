package retry

import (
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util/errors"
	"github.com/argoproj/argo/util/retry"
)

type offloadNodeStatusRepoWithRetry struct {
	delegate sqldb.OffloadNodeStatusRepo
}

func WithRetry(delegate sqldb.OffloadNodeStatusRepo) sqldb.OffloadNodeStatusRepo {
	return &offloadNodeStatusRepoWithRetry{delegate}
}

func (o *offloadNodeStatusRepoWithRetry) Save(uid, namespace string, nodes wfv1.Nodes) (string, error) {
	var version string
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		var err error
		version, err = o.delegate.Save(uid, namespace, nodes)
		return done(err), err
	})
	return version, err
}

func (o *offloadNodeStatusRepoWithRetry) Get(uid, version string) (wfv1.Nodes, error) {
	var nodes wfv1.Nodes
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		var err error
		nodes, err = o.delegate.Get(uid, version)
		return done(err), err
	})
	return nodes, err
}

func done(err error) bool {
	return err == nil || !errors.IsTransientErr(err)
}

func (o *offloadNodeStatusRepoWithRetry) List(namespace string) (map[sqldb.UUIDVersion]wfv1.Nodes, error) {
	var nodes map[sqldb.UUIDVersion]wfv1.Nodes
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		var err error
		nodes, err = o.delegate.List(namespace)
		return done(err), err
	})
	return nodes, err
}

func (o *offloadNodeStatusRepoWithRetry) ListOldOffloads(namespace string) ([]sqldb.UUIDVersion, error) {
	var versions []sqldb.UUIDVersion
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		var err error
		versions, err = o.delegate.ListOldOffloads(namespace)
		return done(err), err
	})
	return versions, err
}

func (o *offloadNodeStatusRepoWithRetry) Delete(uid, version string) error {
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		err := o.delegate.Delete(uid, version)
		return done(err), err
	})
	return err
}

func (o *offloadNodeStatusRepoWithRetry) IsEnabled() bool {
	return o.delegate.IsEnabled()
}

var _ sqldb.OffloadNodeStatusRepo = &offloadNodeStatusRepoWithRetry{}
