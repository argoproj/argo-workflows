package axdb

import (
	"github.com/gocql/gocql"
	"strings"
)

func IgnoreErrors(msg string) bool {
	if strings.Contains(msg, "is a duplicate of existing index") ||
		strings.Contains(msg, "Index ") && strings.Contains(msg, "already exists") ||
		// index to be dropped doesn't exist, it's OK
		strings.Contains(msg, "Index") && strings.Contains(msg, "could not be found in any of the tables of keyspace") ||
		// the column to be dropped doesn't exist, it's OK
		strings.Contains(msg, "Column") && strings.Contains(msg, "was not found in table") ||
		// the column to be added has existed, it's OK
		strings.Contains(msg, "Invalid column name") && strings.Contains(msg, "because it conflicts with an existing column") {
		return true
	}
	return false
}

func GetAXDBErrCodeFromDBError(err error) int {
	if err != nil {
		switch err.(type) {
		case gocql.RequestError:
			realErr := err.(gocql.RequestError)
			// 2xx: problem validating the request
			if realErr.Code() >= 0x2000 {
				if realErr.Code() == 0x2200 && IgnoreErrors(realErr.Message()) {
					return RestStatusOK
				} else {
					return RestStatusInvalid
				}
			} else if realErr.Code() >= 0x1000 { // 1xx: problem during request execution
				return RestStatusInternalError
			} else {
				// protocol, credential error
				return RestStatusDenied
			}
		default:
			// for safety reason, we will return InternalError, it could cause unnecessary retry, but no hurt.
			// this code path isn't supposed to touch.
			return RestStatusInternalError
		}
	} else {
		return RestStatusOK
	}
}
