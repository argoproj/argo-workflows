package sqldb

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/upper/db/v4"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// ListWorkflowsLabelKeys returns distinct name from argo_archived_workflows_labels table
// SELECT DISTINCT name FROM argo_archived_workflows_labels
func (r *workflowArchive) ListWorkflowsLabelKeys() (*wfv1.LabelKeys, error) {
	var archivedWfLabels []archivedWorkflowLabelRecord

	err := r.session.SQL().
		Select(db.Raw("DISTINCT name")).
		From(archiveLabelsTableName).
		All(&archivedWfLabels)
	if err != nil {
		return nil, err
	}
	labelKeys := make([]string, len(archivedWfLabels))
	for i, md := range archivedWfLabels {
		labelKeys[i] = md.Key
	}

	return &wfv1.LabelKeys{Items: labelKeys}, nil
}

// ListWorkflowsLabelValues returns distinct value from argo_archived_workflows_labels table
// SELECT DISTINCT value FROM argo_archived_workflows_labels WHERE name=labelkey
func (r *workflowArchive) ListWorkflowsLabelValues(key string) (*wfv1.LabelValues, error) {
	var archivedWfLabels []archivedWorkflowLabelRecord
	err := r.session.SQL().
		Select(db.Raw("DISTINCT value")).
		From(archiveLabelsTableName).
		Where(db.Cond{"name": key}).
		All(&archivedWfLabels)
	if err != nil {
		return nil, err
	}
	labels := make([]string, len(archivedWfLabels))
	for i, md := range archivedWfLabels {
		labels[i] = md.Value
	}

	return &wfv1.LabelValues{Items: labels}, nil
}

func labelsClause(selector db.Selector, t dbType, requirements labels.Requirements) (db.Selector, error) {
	for _, req := range requirements {
		cond, err := requirementToCondition(t, req)
		if err != nil {
			return nil, err
		}
		selector = selector.And(cond)
	}
	return selector, nil
}

func requirementToCondition(t dbType, r labels.Requirement) (*db.RawExpr, error) {
	// Should we "sanitize our inputs"? No.
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
	// Valid label values must be 63 characters or less and must be empty or begin and end with an alphanumeric character ([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and alphanumerics between.
	// https://kb.objectrocket.com/postgresql/casting-in-postgresql-570#string+to+integer+casting
	switch r.Operator() {
	case selection.DoesNotExist:
		return db.Raw(fmt.Sprintf("not exists (select 1 from %s where clustername = %s.clustername and uid = %s.uid and name = '%s')", archiveLabelsTableName, archiveTableName, archiveTableName, r.Key())), nil
	case selection.Equals, selection.DoubleEquals:
		return db.Raw(fmt.Sprintf("exists (select 1 from %s where clustername = %s.clustername and uid = %s.uid and name = '%s' and value = '%s')", archiveLabelsTableName, archiveTableName, archiveTableName, r.Key(), r.Values().List()[0])), nil
	case selection.In:
		return db.Raw(fmt.Sprintf("exists (select 1 from %s where clustername = %s.clustername and uid = %s.uid and name = '%s' and value in ('%s'))", archiveLabelsTableName, archiveTableName, archiveTableName, r.Key(), strings.Join(r.Values().List(), "', '"))), nil
	case selection.NotEquals:
		return db.Raw(fmt.Sprintf("not exists (select 1 from %s where clustername = %s.clustername and uid = %s.uid and name = '%s' and value = '%s')", archiveLabelsTableName, archiveTableName, archiveTableName, r.Key(), r.Values().List()[0])), nil
	case selection.NotIn:
		return db.Raw(fmt.Sprintf("not exists (select 1 from %s where clustername = %s.clustername and uid = %s.uid and name = '%s' and value in ('%s'))", archiveLabelsTableName, archiveTableName, archiveTableName, r.Key(), strings.Join(r.Values().List(), "', '"))), nil
	case selection.Exists:
		return db.Raw(fmt.Sprintf("exists (select 1 from %s where clustername = %s.clustername and uid = %s.uid and name = '%s')", archiveLabelsTableName, archiveTableName, archiveTableName, r.Key())), nil
	case selection.GreaterThan:
		i, err := strconv.Atoi(r.Values().List()[0])
		if err != nil {
			return nil, err
		}
		return db.Raw(fmt.Sprintf("exists (select 1 from %s where clustername = %s.clustername and uid = %s.uid and name = '%s' and cast(value as %s) > %d)", archiveLabelsTableName, archiveTableName, archiveTableName, r.Key(), t.intType(), i)), nil
	case selection.LessThan:
		i, err := strconv.Atoi(r.Values().List()[0])
		if err != nil {
			return nil, err
		}
		return db.Raw(fmt.Sprintf("exists (select 1 from %s where clustername = %s.clustername and uid = %s.uid and name = '%s' and cast(value as %s) < %d)", archiveLabelsTableName, archiveTableName, archiveTableName, r.Key(), t.intType(), i)), nil
	}
	return nil, fmt.Errorf("operation %v is not supported", r.Operator())
}
