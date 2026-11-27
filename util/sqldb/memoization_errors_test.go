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

func TestIsDuplicateKey(t *testing.T) {
	for _, err := range []error{&pq.Error{Code: "23505", Message: "clé dupliquée"}, &mysql.MySQLError{Number: 1062, Message: "doppelter Schlüssel"}} {
		assert.True(t, IsDuplicateKey(err))
		assert.True(t, IsDuplicateKey(fmt.Errorf("wrapped: %w", err)))
	}
	assert.False(t, IsDuplicateKey(nil))
	assert.False(t, IsDuplicateKey(errors.New("duplicate key")))
	assert.False(t, IsDuplicateKey(&pq.Error{Code: "42501"}))
}

func TestIsTransientDatabaseError(t *testing.T) {
	for _, code := range []pqerror.Code{"08006", "40001", "40P01", "57P01", "57P02", "57P03", "53300"} {
		assert.True(t, IsTransientError(fmt.Errorf("wrapped: %w", &pq.Error{Code: code})))
	}
	for _, code := range []uint16{1040, 1053, 1205, 1213, 2006, 2013} {
		assert.True(t, IsTransientError(fmt.Errorf("wrapped: %w", &mysql.MySQLError{Number: code})))
	}
	assert.False(t, IsTransientError(nil))
	assert.False(t, IsTransientError(&pq.Error{Code: "42501"}))
	assert.False(t, IsTransientError(&mysql.MySQLError{Number: 1146}))
}
