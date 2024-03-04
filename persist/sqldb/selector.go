package sqldb

import (
	"time"

	"github.com/upper/db/v4"
	"k8s.io/apimachinery/pkg/labels"
)

func BuildWorkflowSelector(selector db.Selector, tableName, labelTableName string, hasClusterName bool, t dbType, namespace string, name string, namePrefix string, minStartedAt, maxStartedAt time.Time, labelRequirements labels.Requirements, limit, offset int) (db.Selector, error) {
	// If we were passed 0 as the limit, then we should load all available archived workflows
	// to match the behavior of the `List` operations in the Kubernetes API
	if limit == 0 {
		limit = -1
		offset = -1
	}

	selector = selector.
		And(namespaceEqual(namespace)).
		And(nameEqual(name)).
		And(namePrefixClause(namePrefix)).
		And(startedAtFromClause(minStartedAt)).
		And(startedAtToClause(maxStartedAt))

	selector, err := labelsClause(selector, t, labelRequirements, tableName, labelTableName, hasClusterName)
	if err != nil {
		return nil, err
	}
	return selector.
		OrderBy("-startedat").
		Limit(limit).
		Offset(offset), nil
}
