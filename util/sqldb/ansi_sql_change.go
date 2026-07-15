package sqldb

import (
	"context"

	"github.com/upper/db/v4"
)

// AnsiSQLChange represents a straightforward change that is compatible with all database providers.
type AnsiSQLChange string

func (s AnsiSQLChange) Apply(ctx context.Context, session db.Session) error {
	_, err := session.SQL().Exec(string(s))
	return err
}
