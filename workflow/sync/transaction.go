package sync

import (
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
)

type transaction struct {
	sessionProxy *sqldb.SessionProxy
}
