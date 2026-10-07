//go:build !windows

package sync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

const wfWithMemoryAndDatabaseMutex = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-dual-mutex
  namespace: default
spec:
  entrypoint: whalesay
  synchronization:
    mutexes:
      - name: shared
      - name: shared
        database: true
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

// A workflow holding an in-memory and a database mutex of the same name (the
// recommended way to migrate from one to the other) must be at the front of
// both queues before it can acquire either. When several waiters share a
// priority and a creation timestamp - which Kubernetes stores at second
// resolution - each queue must break the tie the same way, or the heap can
// favour one waiter while the database favours another and neither ever runs.
func TestDualMemoryAndDatabaseMutexTiedQueueOrder(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
			defer cancel()
			info, deferfn, cfg, err := createTestDBSession(ctx, t, dbType)
			require.NoError(t, err)
			defer deferfn()

			syncMgr := createLockManager(ctx, info.SessionProxy, &cfg, func(_ context.Context, _ string) (int, error) { return 1, nil }, func(string) {}, WorkflowExistenceFunc)
			require.NotNil(t, syncMgr)

			created := metav1.Time{Time: time.Now().Add(-time.Minute).Truncate(time.Second)}
			newWf := func(name string) *wfv1.Workflow {
				wf := wfv1.MustUnmarshalWorkflow(wfWithMemoryAndDatabaseMutex)
				wf.Name = name
				wf.CreationTimestamp = created
				return wf
			}
			wfA, wfB, wfC := newWf("wf-a"), newWf("wf-b"), newWf("wf-c")

			checkCanAcquire(ctx, t, syncMgr, wfA)
			checkCannotAcquire(ctx, t, syncMgr, wfB)
			checkCannotAcquire(ctx, t, syncMgr, wfC)

			// Releasing the holder removes it from the in-memory heap, which
			// reorders tied entries unless the key breaks the tie.
			syncMgr.Release(ctx, wfA, wfA.Name, wfA.Spec.Synchronization)
			checkCannotAcquire(ctx, t, syncMgr, wfC)
			checkCanAcquire(ctx, t, syncMgr, wfB)

			syncMgr.Release(ctx, wfB, wfB.Name, wfB.Spec.Synchronization)
			checkCanAcquire(ctx, t, syncMgr, wfC)
		})
	}
}
