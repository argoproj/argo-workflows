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

func BuildWorkflowSelectorForRawQuery(in string, inArgs []any, tableName, labelTableName string, hasClusterName bool, t dbType, namespace string, name string, namePrefix string, minStartedAt, maxStartedAt time.Time, labelRequirements labels.Requirements, limit, offset int) (out string, outArgs []any, err error) {
	clauses := []*db.RawExpr{
		namespaceEqual(namespace),
		nameEqual(name),
		namePrefixClause(namePrefix),
		startedAtFromClause(minStartedAt),
		startedAtToClause(maxStartedAt),
	}
	for _, r := range labelRequirements {
		q, err := requirementToCondition(t, r, tableName, labelTableName, hasClusterName)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, q)
	}
	out = in
	outArgs = inArgs
	for _, c := range clauses {
		if c == nil || c.Empty() {
			continue
		}
		out += " AND " + c.Raw()
		outArgs = append(outArgs, c.Arguments()...)
	}

	out += " ORDER BY startedat DESC"

	// If we were passed 0 as the limit, then we should load all available archived workflows
	// to match the behavior of the `List` operations in the Kubernetes API
	if limit == 0 {
		limit = -1
		offset = -1
	}
	out += " LIMIT ?"
	outArgs = append(outArgs, limit)
	out += " OFFSET ?"
	outArgs = append(outArgs, offset)
	return
}
