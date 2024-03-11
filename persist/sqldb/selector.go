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
	var clauses []*db.RawExpr
	if namespace != "" {
		clauses = append(clauses, db.Raw("namespace = ?", namespace))
	}
	if name != "" {
		clauses = append(clauses, db.Raw("name = ?", name))
	}
	if namePrefix != "" {
		clauses = append(clauses, db.Raw("name like ?", namePrefix+"%"))
	}
	if !minStartedAt.IsZero() {
		clauses = append(clauses, db.Raw("startedat > ?", minStartedAt))
	}
	if !maxStartedAt.IsZero() {
		clauses = append(clauses, db.Raw("startedat < ?", maxStartedAt))
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
		out += " and " + c.Raw()
		outArgs = append(outArgs, c.Arguments()...)
	}

	out += " order by startedat desc"

	// If we were passed 0 as the limit, then we should load all available archived workflows
	// to match the behavior of the `List` operations in the Kubernetes API
	if limit == 0 {
		limit = -1
		offset = -1
	}
	out += " limit ?"
	outArgs = append(outArgs, limit)
	out += " offset ?"
	outArgs = append(outArgs, offset)
	return
}
