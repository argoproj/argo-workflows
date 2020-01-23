package sqldb

import "upper.io/db.v3/lib/sqlbuilder"

// represent a straight forward change that is compatible with all database providers
type ansiSQLChange string

func (s ansiSQLChange) apply(session sqlbuilder.Database) error {
	_, err := session.Exec(string(s))
	return err
}
