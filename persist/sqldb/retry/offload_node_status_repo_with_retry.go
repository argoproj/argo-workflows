package retry

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util/errors"
)

// should be <10s
// Retry	Seconds
// 1	0.10
// 2	0.30
// 3	0.70
// 4	1.50
// 5	3.10
var readRetry = wait.Backoff{Steps: 5, Duration: 100 * time.Millisecond, Factor: 2}

// needs to be long
// http://backoffcalculator.com/?attempts=5&rate=2&interval=1
// Retry	Seconds
// 1	1.00
// 2	3.00
// 3	7.00
// 4	15.00
// 5	31.00
var writeRetry = wait.Backoff{Steps: 5, Duration: 1 * time.Second, Factor: 2}

type offloadNodeStatusRepoWithRetry struct {
	delegate sqldb.OffloadNodeStatusRepo
}

func WithRetry(delegate sqldb.OffloadNodeStatusRepo) sqldb.OffloadNodeStatusRepo {
	return &offloadNodeStatusRepoWithRetry{delegate}
}

func (o *offloadNodeStatusRepoWithRetry) Save(uid, namespace string, nodes wfv1.Nodes) (string, error) {
	var version string
	err := wait.ExponentialBackoff(writeRetry, func() (bool, error) {
		var err error
		version, err = o.delegate.Save(uid, namespace, nodes)
		return done(err), err
	})
	return version, err
}

func (o *offloadNodeStatusRepoWithRetry) Get(uid, version string) (wfv1.Nodes, error) {
	var nodes wfv1.Nodes
	err := wait.ExponentialBackoff(readRetry, func() (bool, error) {
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
	err := wait.ExponentialBackoff(readRetry, func() (bool, error) {
		var err error
		nodes, err = o.delegate.List(namespace)
		return done(err), err
	})
	return nodes, err
}

func (o *offloadNodeStatusRepoWithRetry) ListOldOffloads(namespace string) ([]sqldb.UUIDVersion, error) {
	var versions []sqldb.UUIDVersion
	err := wait.ExponentialBackoff(readRetry, func() (bool, error) {
		var err error
		versions, err = o.delegate.ListOldOffloads(namespace)
		return done(err), err
	})
	return versions, err
}

func (o *offloadNodeStatusRepoWithRetry) Delete(uid, version string) error {
	err := wait.ExponentialBackoff(writeRetry, func() (bool, error) {
		err := o.delegate.Delete(uid, version)
		return done(err), err
	})
	return err
}

func (o *offloadNodeStatusRepoWithRetry) IsEnabled() bool {
	return o.delegate.IsEnabled()
}

var _ sqldb.OffloadNodeStatusRepo = &offloadNodeStatusRepoWithRetry{}
