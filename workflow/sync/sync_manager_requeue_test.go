package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	gosync "sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
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

// stubDBSemaphore satisfies the semaphore interface with canned answers so a
// test can drive TryAcquire's database retry loop into exactly the failure it
// wants, while the transaction itself runs against a real database.
type stubDBSemaphore struct {
	tryAcquireErr error
	tryAcquireN   int
}

func (s *stubDBSemaphore) acquire(_ context.Context, _ string, _ *sqldb.SessionProxy) (bool, error) {
	return true, nil
}

func (s *stubDBSemaphore) reacquire(_ context.Context, _ string, _ *sqldb.SessionProxy) error {
	return nil
}

func (s *stubDBSemaphore) checkAcquire(_ context.Context, _ string, _ *sqldb.SessionProxy) (bool, bool, string) {
	return true, false, ""
}

func (s *stubDBSemaphore) tryAcquire(_ context.Context, _ string, _ *sqldb.SessionProxy) (bool, string, error) {
	s.tryAcquireN++
	if s.tryAcquireErr != nil {
		return false, "", s.tryAcquireErr
	}
	return true, "", nil
}

func (s *stubDBSemaphore) release(_ context.Context, _ string) bool { return true }

func (s *stubDBSemaphore) addToQueue(_ context.Context, _ string, _ int32, _ time.Time) error {
	return nil
}

func (s *stubDBSemaphore) removeFromQueue(_ context.Context, _ string) error { return nil }

func (s *stubDBSemaphore) getCurrentHolders(_ context.Context) ([]string, error) { return nil, nil }

func (s *stubDBSemaphore) getCurrentPending(_ context.Context) ([]string, error) { return nil, nil }

func (s *stubDBSemaphore) getLimit(_ context.Context) int { return 1 }

func (s *stubDBSemaphore) probeWaiting(_ context.Context) {}

func (s *stubDBSemaphore) lock(_ context.Context) bool { return true }

func (s *stubDBSemaphore) unlock(_ context.Context) {}

func conflictErrForDBType(dbType sqldb.DBType) error {
	switch dbType {
	case sqldb.MySQL:
		return &mysql.MySQLError{Number: 1213, Message: "Deadlock found when trying to get lock; try restarting transaction"}
	default:
		return &pq.Error{Code: "40001", Message: "could not serialize access due to read/write dependencies among transactions"}
	}
}

// TestDBLockRetryExhaustion drives TryAcquire's database retry loop to
// exhaustion. A genuine driver conflict must be swallowed and the workflow
// requeued with its in-memory synchronization status rolled back; an error
// that merely matches isRetryableSyncError's substrings must surface.
func TestDBLockRetryExhaustion(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			info, cleanup, syncConfig, err := createTestDBSession(ctx, t, dbType)
			require.NoError(t, err)
			defer cleanup()

			newManagerWithStubs := func(stubB *stubDBSemaphore) (*Manager, *stubDBSemaphore, *[]string, *gosync.Mutex) {
				var mu gosync.Mutex
				requeued := []string{}
				syncManager := createLockManager(ctx, info.SessionProxy, &syncConfig, nil, func(key string) {
					mu.Lock()
					defer mu.Unlock()
					requeued = append(requeued, key)
				}, WorkflowExistenceFunc)
				require.NotNil(t, syncManager)
				stubA := &stubDBSemaphore{}
				syncManager.syncLockMap["default/Database/sem-a"] = stubA
				syncManager.syncLockMap["default/Database/sem-b"] = stubB
				return syncManager, stubA, &requeued, &mu
			}

			t.Run("ConflictRequeuesAndRestoresStatus", func(t *testing.T) {
				stubB := &stubDBSemaphore{tryAcquireErr: conflictErrForDBType(dbType)}
				syncManager, stubA, requeued, mu := newManagerWithStubs(stubB)

				wf := wfv1.MustUnmarshalWorkflow(wfWithTwoDBSemaphores)
				acquired, updated, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)

				require.NoError(t, err, "a retryable conflict that exhausts its retries must not surface as an error")
				assert.False(t, acquired)
				assert.True(t, updated, "the restored waiting status must be reported so the caller persists it")
				assert.Equal(t, dbRetryRequeueMsg, msg)
				assert.Equal(t, "default/Database/sem-b", failedLockName, "the lock whose acquisition failed must be attributed")

				// The retry loop must have run to exhaustion.
				assert.Equal(t, 5, stubA.tryAcquireN)
				assert.Equal(t, 5, stubB.tryAcquireN)

				// Each aborted attempt acquired sem-a and mutated the in-memory
				// status via LockAcquired before failing on sem-b; the rollback
				// of the transaction must roll that back too.
				require.NotNil(t, wf.Status.Synchronization)
				require.NotNil(t, wf.Status.Synchronization.Semaphore)
				assert.Empty(t, wf.Status.Synchronization.Semaphore.Holding,
					"aborted transactions must not leave in-memory holds the database rolled back")

				// prepAcquire queued this workflow against both locks outside the
				// transaction, so both queue rows outlive the abort. The waiting
				// entries are what ReleaseAll walks to remove them, so rolling the
				// status back must not take them with it.
				waiting := []string{}
				for _, w := range wf.Status.Synchronization.Semaphore.Waiting {
					waiting = append(waiting, w.Semaphore)
				}
				assert.ElementsMatch(t, []string{"default/Database/sem-a", "default/Database/sem-b"}, waiting,
					"every lock left with a surviving queue row must be recorded as waiting")

				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, []string{"default/hello-world-two-db-sems"}, *requeued)
			})

			t.Run("NonConflictErrorSurfaces", func(t *testing.T) {
				// Matches isRetryableSyncError's "rollback" substring, so it is
				// retried, but it is not a driver conflict code, so exhausting the
				// retries must fail the acquisition rather than requeue.
				stubB := &stubDBSemaphore{tryAcquireErr: errors.New("unexpected rollback failure")}
				syncManager, stubA, requeued, mu := newManagerWithStubs(stubB)

				wf := wfv1.MustUnmarshalWorkflow(wfWithTwoDBSemaphores)
				acquired, updated, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)

				require.ErrorContains(t, err, "unexpected rollback failure")
				assert.False(t, acquired)
				assert.False(t, updated)
				assert.Empty(t, msg)
				assert.Equal(t, "default/Database/sem-b", failedLockName)
				assert.Equal(t, 5, stubA.tryAcquireN)
				assert.Equal(t, 5, stubB.tryAcquireN)
				assert.Empty(t, wf.Status.Synchronization.Semaphore.Holding)

				mu.Lock()
				defer mu.Unlock()
				assert.Empty(t, *requeued)
			})
		})
	}
}

func TestIsSerializationConflict(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"postgres serialization failure", &pq.Error{Code: "40001"}, true},
		{"postgres deadlock", &pq.Error{Code: "40P01"}, true},
		{"postgres unique violation", &pq.Error{Code: "23505"}, false},
		{"mysql deadlock", &mysql.MySQLError{Number: 1213}, true},
		{"mysql lock wait timeout", &mysql.MySQLError{Number: 1205}, false},
		{"wrapped postgres conflict", fmt.Errorf("transaction failed: %w", &pq.Error{Code: "40001"}), true},
		{"wrapped mysql deadlock", fmt.Errorf("transaction failed: %w", &mysql.MySQLError{Number: 1213}), true},
		{"substring lookalike", errors.New("serialization failure during rollback"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isSerializationConflict(tt.err))
		})
	}
}

// TestSerializationConflictSurvivesTxWith forces a real transaction conflict
// through SessionProxy.TxWith and asserts the driver's typed error reaches the
// caller intact - the property isSerializationConflict depends on. Two
// SERIALIZABLE transactions run the classic write-skew dance: each reads the
// row the other writes. PostgreSQL aborts one at commit with SQLSTATE 40001;
// MySQL's shared read locks turn the crossing updates into a deadlock (1213).
func TestSerializationConflictSurvivesTxWith(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			info, cleanup, _, err := createTestDBSession(ctx, t, dbType)
			require.NoError(t, err)
			defer cleanup()

			sess := info.SessionProxy.Session()
			_, err = sess.SQL().Exec(`CREATE TABLE conflict_probe (id INT PRIMARY KEY, val INT)`)
			require.NoError(t, err)
			_, err = sess.SQL().Exec(`INSERT INTO conflict_probe (id, val) VALUES (1, 0), (2, 0)`)
			require.NoError(t, err)

			var readBarrier gosync.WaitGroup
			readBarrier.Add(2)
			runTx := func(readID, writeID int) error {
				return info.SessionProxy.TxWith(ctx, func(sp *sqldb.SessionProxy) error {
					row, err := sp.Session().SQL().QueryRow(`SELECT val FROM conflict_probe WHERE id = ?`, readID)
					if err != nil {
						readBarrier.Done()
						return err
					}
					var val int
					if scanErr := row.Scan(&val); scanErr != nil {
						readBarrier.Done()
						return scanErr
					}
					readBarrier.Done()
					readBarrier.Wait()
					_, err = sp.Session().SQL().Exec(`UPDATE conflict_probe SET val = val + 1 WHERE id = ?`, writeID)
					return err
				}, &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: false})
			}

			errCh := make(chan error, 2)
			go func() { errCh <- runTx(1, 2) }()
			go func() { errCh <- runTx(2, 1) }()
			errs := []error{<-errCh, <-errCh}

			conflicts := 0
			for _, txErr := range errs {
				if txErr == nil {
					continue
				}
				t.Logf("transaction error (%T): %v", txErr, txErr)
				if isSerializationConflict(txErr) {
					conflicts++
				}
			}
			require.GreaterOrEqual(t, conflicts, 1,
				"at least one transaction must fail with a driver conflict error that survives TxWith, got: %v", errs)
		})
	}
}
