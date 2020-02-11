package sqldb

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"upper.io/db.v3"
)

func labelsClause(requirements labels.Requirements) (db.Compound, error) {
	var conds []db.Compound
	for _, r := range requirements {
		cond, err := requirementToCondition(r)
		if err != nil {
			return nil, err
		}
		conds = append(conds, cond)
	}
	return db.And(conds...), nil
}

func requirementToCondition(r labels.Requirement) (db.Compound, error) {
	// Should we "sanitize our inputs"? No.
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
	// Valid label values must be 63 characters or less and must be empty or begin and end with an alphanumeric character ([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and alphanumerics between.
	switch r.Operator() {
	case selection.DoesNotExist:
		return db.Raw(fmt.Sprintf("not exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s')", r.Key())), nil
	case selection.Equals, selection.DoubleEquals:
		return db.Raw(fmt.Sprintf("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s' and value = '%s')", r.Key(), r.Values().List()[0])), nil
	case selection.Exists:
		return db.Raw(fmt.Sprintf("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s')", r.Key())), nil
	case selection.In:
		return db.Raw(fmt.Sprintf("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s' and value in ('%s'))", r.Key(), strings.Join(r.Values().List(), "', '"))), nil
	case selection.NotEquals:
		return db.Raw(fmt.Sprintf("not exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s' and value = '%s')", r.Key(), r.Values().List()[0])), nil
	case selection.NotIn:
		return db.Raw(fmt.Sprintf("not exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s' and value in ('%s'))", r.Key(), strings.Join(r.Values().List(), "', '"))), nil
	}
	return nil, fmt.Errorf("operation %v is not supported", r.Operator())
}
