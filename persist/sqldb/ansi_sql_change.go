package sqldb

import (
	"github.com/upper/db/v4"
)

// represent a straight forward change that is compatible with all database providers
type ansiSQLChange string

func (s ansiSQLChange) apply(session db.Session) error {
	_, err := session.SQL().Exec(string(s))
	return err
}
