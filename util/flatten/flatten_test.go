package flatten

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
)

func Test_flatten(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want map[string]string
	}{
		{"Empty", workflowarchive.ListArchivedWorkflowsRequest{}, map[string]string{}},
		{"NotEmpty", workflowarchive.ListArchivedWorkflowsRequest{ListOptions: &metav1.ListOptions{
			LabelSelector:       "foo=55",
			FieldSelector:       "bar=66",
			Watch:               false,
			AllowWatchBookmarks: true,
			ResourceVersion:     "11",
			TimeoutSeconds:      ptr.To(int64(22)),
			Limit:               33,
			Continue:            "44",
		}}, map[string]string{
			"listOptions.allowWatchBookmarks": "true",
			"listOptions.continue":            "44",
			"listOptions.fieldSelector":       "bar=66",
			"listOptions.labelSelector":       "foo=55",
			"listOptions.limit":               "33",
			"listOptions.resourceVersion":     "11",
			"listOptions.timeoutSeconds":      "22",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Flatten(tt.in))
		})
	}
}
