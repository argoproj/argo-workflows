package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
	"upper.io/db.v3"
)

func Test_labelsClause(t *testing.T) {
	tests := []struct {
		name         string
		requirements labels.Requirements
		want         db.Compound
	}{
		{"Empty", requirements(""), db.And()},
		{"Equals", requirements("foo=bar"), db.And(db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and value = 'bar'"))},
		{"DoubleEquals", requirements("foo==bar"), db.And(db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and value = 'bar'"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := labelsClause(tt.requirements)
			if assert.NoError(t, err) {
				assert.Equal(t, tt.want.Sentences(), got.Sentences())
			}
		})
	}
}

func requirements(selector string) []labels.Requirement {
	requirements, err := labels.ParseToRequirements(selector)
	if err != nil {
		panic(err)
	}
	return requirements
}
