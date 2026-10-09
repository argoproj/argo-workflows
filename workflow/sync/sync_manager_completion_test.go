//go:build !windows

package sync

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	syncdb "github.com/argoproj/argo-workflows/v4/util/sync/db"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestCompletedDatabaseHolderCleanupAfterManagerRestart(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
			info, cleanup, cfg, err := createTestDBSession(ctx, t, dbType)
			require.NoError(t, err)
			defer func() {
				cancel()
				cleanup()
			}()
			getLimit := func(context.Context, string) (int, error) { return 1, nil }
			newManager := func(workflowExists WorkflowExists) *Manager {
				// These managers represent successive processes, so drive their
				// housekeeping explicitly. Starting overlapping DB background
				// loops in this process would also overlap upper/db's global
				// logger setup. The existing database helper already migrates
				// storage and records a live heartbeat for this controller name.
				manager := createLockManager(ctx, nil, &cfg, getLimit, func(string) {}, workflowExists)
				manager.dbInfo = info
				manager.queries = syncdb.NewSyncQueries(info.SessionProxy, info.Config)
				return manager
			}
			old := newManager(func(string) bool { return true })
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "completed", Namespace: "default", UID: "completed-uid", CreationTimestamp: metav1.Now()},
				Spec: wfv1.WorkflowSpec{Synchronization: &wfv1.Synchronization{
					Mutexes: []*wfv1.Mutex{{Name: "completed-restart", Database: true}},
				}},
				Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning},
			}
			acquired, _, _, _, err := old.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
			require.NoError(t, err)
			require.True(t, acquired)
			require.NotNil(t, wf.Status.Synchronization)
			wf.Status.Phase = wfv1.WorkflowError
			wf.Labels = map[string]string{common.LabelKeyCompleted: "true"}
			// The completion is persisted, but the process exits before its
			// informer can release the database hold. Its controller name stays
			// alive after restart, so heartbeat expiry cannot recover this hold.

			var completionAware atomic.Bool
			workflowExists := func(key string) bool {
				if key != "default/completed" {
					return true
				}
				return !completionAware.Load() || wf.Labels[common.LabelKeyCompleted] != "true" || !wf.Status.Phase.Completed()
			}
			restarted := newManager(workflowExists)
			require.Equal(t, old.dbInfo.Config.ControllerName, restarted.dbInfo.Config.ControllerName)
			// The controller initializes its manager from Running Workflows;
			// this retained Error object is absent from that startup list.
			_, err = restarted.Initialize(ctx, nil)
			require.NoError(t, err)
			require.Empty(t, restarted.syncLockMap)
			restarted.ReleaseAll(ctx, wf.DeepCopy())
			require.Empty(t, restarted.syncLockMap, "a completion Add cannot release a lock absent from the new manager")

			contender := wf.DeepCopy()
			contender.Name, contender.UID = "contender", "contender-uid"
			contender.Status, contender.Labels = wfv1.WorkflowStatus{}, nil
			acquired, _, _, _, err = restarted.TryAcquire(ctx, contender, "", contender.Spec.Synchronization)
			require.NoError(t, err)
			require.False(t, acquired, "demand initializes the lock, but its old database hold still blocks admission")
			restarted.CheckWorkflowExistence(ctx)
			acquired, _, _, _, err = restarted.TryAcquire(ctx, contender, "", contender.Spec.Synchronization)
			require.NoError(t, err)
			require.False(t, acquired, "object existence alone keeps a retained completed Workflow's hold")

			// The controller's completion-aware callback lets the existing
			// periodic cleanup retire the hold without deleting Workflow history.
			completionAware.Store(true)
			restarted.CheckWorkflowExistence(ctx)
			acquired, _, _, _, err = restarted.TryAcquire(ctx, contender, "", contender.Spec.Synchronization)
			require.NoError(t, err)
			require.True(t, acquired)
			restarted.ReleaseAll(ctx, contender)
		})
	}
}
