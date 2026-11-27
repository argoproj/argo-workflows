package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestName(t *testing.T) {
	for _, tt := range []struct {
		name                       string
		pool                       *wfv1.Pool
		workflowTemplate, template string
		want                       string
	}{
		{"explicit", &wfv1.Pool{Name: "mine"}, "wft", "tmpl", "mine"},
		{"template", &wfv1.Pool{}, "", "Tmpl", "tmpl"},
		{"workflow template", &wfv1.Pool{}, "my.wft", "tmpl", "my-wft-tmpl"},
		{"cut", &wfv1.Pool{}, "a-very-long-workflow-template-name", "and-template", "a-very-long-workflow-template-name"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := Name(tt.pool, tt.workflowTemplate, tt.template)
			assert.Equal(t, tt.want, got)
			assert.LessOrEqual(t, len(got), NameMaxLength)
		})
	}
}
