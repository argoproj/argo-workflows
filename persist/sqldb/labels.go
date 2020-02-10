package sqldb

import (
	"k8s.io/apimachinery/pkg/labels"
	"upper.io/db.v3"
)

func labelsClause(requirements labels.Requirements) db.Cond {
	// nothing
	return db.Cond{}
}
