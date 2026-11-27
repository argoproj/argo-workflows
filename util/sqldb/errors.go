package sqldb

import (
	"errors"

	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

// mysqlErLockDeadlock is ER_LOCK_DEADLOCK: the transaction was chosen as the
// deadlock victim and rolled back. go-sql-driver/mysql exports no error-number
// constants. ER_LOCK_WAIT_TIMEOUT (1205) is deliberately not treated as a
// conflict: it only rolls back the statement, not the transaction, and it
// signals a held-too-long lock rather than a retryable ordering conflict.
const mysqlErLockDeadlock = 1213

// IsSerializationConflict reports whether err is a transaction conflict
// reported by the database driver itself: PostgreSQL serialization_failure
// (40001) or deadlock_detected (40P01), or MySQL ER_LOCK_DEADLOCK (1213).
// These are the errors where the database aborted the transaction because of
// concurrent activity and the whole transaction can safely be retried.
func IsSerializationConflict(err error) bool {
	if pq.As(err, pqerror.TRSerializationFailure, pqerror.TRDeadlockDetected) != nil {
		return true
	}
	return errors.Is(err, &mysql.MySQLError{Number: mysqlErLockDeadlock})
}
