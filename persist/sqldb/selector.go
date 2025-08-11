package sqldb

import (
	"time"

	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v3/server/utils"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
)

func BuildArchivedWorkflowSelector(selector db.Selector, tableName, labelTableName string, t sqldb.DBType, options utils.ListOptions, count bool) (db.Selector, error) {
	selector = selector.
		And(namespaceEqual(options.Namespace)).
		And(nameEqual(options.Name)).
		And(namePrefixClause(options.NamePrefix)).
		And(startedAtFromClause(options.MinStartedAt)).
		And(startedAtToClause(options.MaxStartedAt)).
		And(createdAfterClause(options.CreatedAfter)).
		And(finishedBeforeClause(options.FinishedBefore))

	selector, err := labelsClause(selector, t, options.LabelRequirements, tableName, labelTableName, true)
	if err != nil {
		return nil, err
	}
	if count {
		return selector, nil
	}
	// If we were passed 0 as the limit, then we should load all available archived workflows
	// to match the behavior of the `List` operations in the Kubernetes API
	if options.Limit == 0 {
		options.Limit = -1
		options.Offset = -1
	}
	return selector.
		OrderBy("-startedat").
		Limit(options.Limit).
		Offset(options.Offset), nil
}

func BuildWorkflowSelector(in string, inArgs []any, tableName, labelTableName string, t sqldb.DBType, options utils.ListOptions, count bool) (out string, outArgs []any, err error) {
	var clauses []*db.RawExpr
	if options.Namespace != "" {
		clauses = append(clauses, db.Raw("namespace = ?", options.Namespace))
	}
	if options.Name != "" {
		nameFilter := options.NameFilter
		if nameFilter == "" {
			nameFilter = "Exact"
		}
		if nameFilter == "Exact" {
			clauses = append(clauses, db.Raw("name = ?", options.Name))
		}
		if nameFilter == "Contains" {
			clauses = append(clauses, db.Raw("name like ?", "%"+options.Name+"%"))
		}
		if nameFilter == "Prefix" {
			clauses = append(clauses, db.Raw("name like ?", options.Name+"%"))
		}
	}
	if !options.CreatedAfter.IsZero() {
		clauses = append(clauses, db.Raw("json_extract(workflow, '$.metadata.creationTimestamp') >= ?", options.CreatedAfter.Format(time.RFC3339)))
	}
	if !options.FinishedBefore.IsZero() {
		clauses = append(clauses, db.Raw("finishedat <= ?", options.FinishedBefore))
	}
	if options.NamePrefix != "" {
		clauses = append(clauses, db.Raw("name like ?", options.NamePrefix+"%"))
	}
	if !options.MinStartedAt.IsZero() {
		clauses = append(clauses, db.Raw("startedat >= ?", options.MinStartedAt))
	}
	if !options.MaxStartedAt.IsZero() {
		clauses = append(clauses, db.Raw("startedat <= ?", options.MaxStartedAt))
	}
	for _, r := range options.LabelRequirements {
		q, err := requirementToCondition(t, r, tableName, labelTableName, false)
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
	if count {
		return out, outArgs, nil
	}
	if options.StartedAtAscending {
		out += " order by startedat asc"
	} else {
		out += " order by startedat desc"
	}

	// If we were passed 0 as the limit, then we should load all available archived workflows
	// to match the behavior of the `List` operations in the Kubernetes API
	if options.Limit == 0 {
		options.Limit = -1
		options.Offset = -1
	}
	out += " limit ?"
	outArgs = append(outArgs, options.Limit)
	out += " offset ?"
	outArgs = append(outArgs, options.Offset)
	return out, outArgs, nil
}
