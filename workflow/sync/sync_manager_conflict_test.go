package sync

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

const wfWithTwoDBSemaphores = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 name: hello-world-two-db-sems
 namespace: default
spec:
 entrypoint: whalesay
 synchronization:
   semaphores:
     - database:
         key: sem-a
     - database:
         key: sem-b
 templates:
 - name: whalesay
   container:
     image: docker/whalesay:latest
     command: [cowsay]
     args: ["hello world"]
`

// stubLock reuses poisonedLock's inert semaphore implementation, overriding
// just the acquisition path so a test can drive TryAcquire's database retry
// loop into exactly the failure it wants while the transaction itself runs
// against a real database.
type stubLock struct {
	*poisonedLock
	tryAcquireErr error
	tryAcquireN   int
}

var _ semaphore = &stubLock{}

func (s *stubLock) checkAcquire(_ context.Context, _ string, _ *sqldb.SessionProxy) (bool, bool, string) {
	return true, false, ""
}

func (s *stubLock) tryAcquire(_ context.Context, _ string, _ *sqldb.SessionProxy) (bool, string, error) {
	s.tryAcquireN++
	if s.tryAcquireErr != nil {
		return false, "", s.tryAcquireErr
	}
	return true, "", nil
}

// conflictErrForDBType returns the driver error a genuine transaction conflict
// produces, with a message that matches none of IsRetryableSyncError's
// substring fallbacks - retries and classification must work from the driver
// type alone.
func conflictErrForDBType(dbType sqldb.DBType) error {
	if dbType == sqldb.MySQL {
		return &mysql.MySQLError{Number: 1213, Message: "chosen as victim; restart transaction"}
	}
	return &pq.Error{Code: pqerror.TRSerializationFailure, Message: "concurrent update"}
}

// TestIsRetryableSyncErrorTypedConflict pins the delegation to the typed
// driver-error classifier: a genuine conflict is retryable however its message
// is worded, while the substring fallback still catches stringified errors.
func TestIsRetryableSyncErrorTypedConflict(t *testing.T) {
	assert.True(t, IsRetryableSyncError(&pq.Error{Code: pqerror.TRSerializationFailure, Message: "concurrent update"}))
	assert.True(t, IsRetryableSyncError(&pq.Error{Code: pqerror.TRDeadlockDetected, Message: "processes are waiting"}))
	assert.True(t, IsRetryableSyncError(fmt.Errorf("commit: %w", &mysql.MySQLError{Number: 1213, Message: "chosen as victim"})))
	assert.True(t, IsRetryableSyncError(errors.New("ERROR: could not serialize access due to read/write dependencies among transactions")), "substring fallback must still work")
	assert.False(t, IsRetryableSyncError(errors.New("duplicate key value violates unique constraint")))
	assert.False(t, IsRetryableSyncError(nil))
}

// TestTryAcquireSurfacesSerializationConflict drives the database retry loop to
// exhaustion. The typed conflict must be retried, surface to the caller still
// classifiable (the caller, not the sync manager, decides to requeue), and the
// failed attempts must leave the workflow's in-memory synchronization status
// untouched.
func TestTryAcquireSurfacesSerializationConflict(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			info, cleanup, syncConfig, err := createTestDBSession(ctx, t, dbType)
			require.NoError(t, err)
			defer cleanup()

			newManagerWithStubs := func(errB error) (*Manager, *stubLock, *stubLock, *[]string) {
				requeued := []string{}
				syncManager := createLockManager(ctx, info.SessionProxy, &syncConfig, nil, func(key string) {
					requeued = append(requeued, key)
				}, WorkflowExistenceFunc)
				require.NotNil(t, syncManager)
				semA := &stubLock{poisonedLock: newPoisonedLock("sem-a", "")}
				semB := &stubLock{poisonedLock: newPoisonedLock("sem-b", ""), tryAcquireErr: errB}
				syncManager.syncLockMap["default/Database/sem-a"] = semA
				syncManager.syncLockMap["default/Database/sem-b"] = semB
				return syncManager, semA, semB, &requeued
			}

			t.Run("ConflictRetriesThenSurfacesClassifiable", func(t *testing.T) {
				syncManager, semA, semB, requeued := newManagerWithStubs(conflictErrForDBType(dbType))

				wf := wfv1.MustUnmarshalWorkflow(wfWithTwoDBSemaphores)
				acquired, _, _, _, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)

				require.Error(t, err, "an exhausted conflict must surface to the caller")
				assert.True(t, sqldb.IsSerializationConflict(err), "the driver error must reach the caller still classifiable")
				assert.False(t, acquired)
				assert.Equal(t, dbRetryBackoff.Steps, semB.tryAcquireN, "the typed classifier must drive the retry loop: the injected message matches no substring fallback")
				assert.Equal(t, dbRetryBackoff.Steps, semA.tryAcquireN, "each attempt must run the whole lock set")
				assert.Empty(t, *requeued, "the sync manager must not requeue; that decision belongs to the caller")
				// sem-a acquired successfully inside every aborted transaction:
				// the rollback must not leave its Holding entry behind.
				require.NotNil(t, wf.Status.Synchronization)
				require.NotNil(t, wf.Status.Synchronization.Semaphore)
				assert.Empty(t, wf.Status.Synchronization.Semaphore.Holding, "a rolled-back acquisition must not leave a Holding entry")
			})

			t.Run("NonRetryableErrorSurfacesWithoutRetries", func(t *testing.T) {
				syncManager, _, semB, requeued := newManagerWithStubs(errors.New("duplicate key value violates unique constraint"))

				wf := wfv1.MustUnmarshalWorkflow(wfWithTwoDBSemaphores)
				acquired, _, _, _, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)

				require.Error(t, err)
				assert.False(t, sqldb.IsSerializationConflict(err))
				assert.False(t, acquired)
				assert.Equal(t, 1, semB.tryAcquireN, "a non-retryable error must fail fast")
				assert.Empty(t, *requeued)
			})
		})
	}
}
