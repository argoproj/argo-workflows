package sqldb

import "upper.io/db.v3/lib/sqlbuilder"

type noopChange struct {
}

func (b noopChange) apply(_ sqlbuilder.Database) error {
	return nil
}
