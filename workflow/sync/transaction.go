package sync

import (
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

type transaction struct {
	sessionProxy *sqldb.SessionProxy
}
