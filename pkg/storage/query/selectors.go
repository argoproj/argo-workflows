package query

import (
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"gorm.io/gorm"
)

// ApplyLabelSelector adds WHERE conditions to the query based on a label selector.
// It uses EXISTS/NOT EXISTS subqueries on the resource_labels table.
func ApplyLabelSelector(db *gorm.DB, sel labels.Selector) *gorm.DB {
	if sel == nil || sel.Empty() {
		return db
	}
	requirements, _ := sel.Requirements()
	for _, req := range requirements {
		db = applyLabelRequirement(db, req)
	}
	return db
}

func applyLabelRequirement(db *gorm.DB, req labels.Requirement) *gorm.DB {
	key := req.Key()
	values := req.Values()

	switch req.Operator() {
	case selection.Equals, selection.DoubleEquals:
		val, _ := values.PopAny()
		return db.Where(
			"EXISTS (SELECT 1 FROM resource_labels WHERE resource_labels.resource_id = resource_records.id AND resource_labels.key = ? AND resource_labels.value = ?)",
			key, val,
		)
	case selection.NotEquals:
		val, _ := values.PopAny()
		return db.Where(
			"NOT EXISTS (SELECT 1 FROM resource_labels WHERE resource_labels.resource_id = resource_records.id AND resource_labels.key = ? AND resource_labels.value = ?)",
			key, val,
		)
	case selection.In:
		vals := values.UnsortedList()
		return db.Where(
			"EXISTS (SELECT 1 FROM resource_labels WHERE resource_labels.resource_id = resource_records.id AND resource_labels.key = ? AND resource_labels.value IN ?)",
			key, vals,
		)
	case selection.NotIn:
		vals := values.UnsortedList()
		return db.Where(
			"NOT EXISTS (SELECT 1 FROM resource_labels WHERE resource_labels.resource_id = resource_records.id AND resource_labels.key = ? AND resource_labels.value IN ?)",
			key, vals,
		)
	case selection.Exists:
		return db.Where(
			"EXISTS (SELECT 1 FROM resource_labels WHERE resource_labels.resource_id = resource_records.id AND resource_labels.key = ?)",
			key,
		)
	case selection.DoesNotExist:
		return db.Where(
			"NOT EXISTS (SELECT 1 FROM resource_labels WHERE resource_labels.resource_id = resource_records.id AND resource_labels.key = ?)",
			key,
		)
	}
	return db
}

// ApplyFieldSelector adds WHERE conditions based on a field selector.
// Only metadata.name and metadata.namespace are supported.
func ApplyFieldSelector(db *gorm.DB, sel fields.Selector) *gorm.DB {
	if sel == nil {
		return db
	}
	for _, req := range sel.Requirements() {
		switch req.Field {
		case "metadata.name":
			db = applyFieldCondition(db, "name", req.Operator, req.Value)
		case "metadata.namespace":
			db = applyFieldCondition(db, "namespace", req.Operator, req.Value)
		}
	}
	return db
}

func applyFieldCondition(db *gorm.DB, column string, op selection.Operator, value string) *gorm.DB {
	switch op {
	case selection.Equals, selection.DoubleEquals:
		return db.Where(fmt.Sprintf("%s = ?", column), value)
	case selection.NotEquals:
		return db.Where(fmt.Sprintf("%s != ?", column), value)
	}
	return db
}
