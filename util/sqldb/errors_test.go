package sqldb

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
	"github.com/stretchr/testify/assert"
)

func TestIsSerializationConflict(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"postgres serialization failure", &pq.Error{Code: pqerror.TRSerializationFailure, Message: "could not serialize access"}, true},
		{"postgres deadlock detected", &pq.Error{Code: pqerror.TRDeadlockDetected, Message: "deadlock detected"}, true},
		{"postgres wrapped", fmt.Errorf("commit: %w", &pq.Error{Code: pqerror.TRSerializationFailure}), true},
		{"postgres unrelated code", &pq.Error{Code: "23505", Message: "duplicate key value"}, false},
		{"mysql deadlock", &mysql.MySQLError{Number: 1213, Message: "Deadlock found when trying to get lock; try restarting transaction"}, true},
		{"mysql wrapped", fmt.Errorf("commit: %w", &mysql.MySQLError{Number: 1213}), true},
		{"mysql lock wait timeout", &mysql.MySQLError{Number: 1205, Message: "Lock wait timeout exceeded; try restarting transaction"}, false},
		{"plain error mentioning deadlock", errors.New("deadlock detected"), false},
		{"plain error mentioning serialization", errors.New("could not serialize access"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsSerializationConflict(tt.err))
		})
	}
}
