package namespace

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestNamespace_IsEmpty(t *testing.T) {
	assert.True(t, Namespace{}.IsEmpty())
	assert.False(t, Namespace{Name: "my-ns"}.IsEmpty())
}

func TestRules_Allow(t *testing.T) {
	for _, tt := range []struct {
		name        string
		rules       Rules
		clusterName wfv1.ClusterName
		namespace   string
		want        bool
	}{
		{"no rules", Rules{}, "", "", false},
		{"one empty rules", Rules{{}}, "", "", false},
		{"any    cluster, no   namespace", Rules{{ClusterNames: []wfv1.ClusterName{""}, Namespaces: []string{}}}, "", "", false},
		{"no    cluster, any   namespace", Rules{{ClusterNames: []wfv1.ClusterName{}, Namespaces: []string{""}}}, "", "", false},
		{"any   cluster, any   namespace", Rules{{ClusterNames: []wfv1.ClusterName{""}, Namespaces: []string{""}}}, "", "", true},
		{"other cluster, any   namespace", Rules{{ClusterNames: []wfv1.ClusterName{"o"}, Namespaces: []string{""}}}, "", "", false},
		{"any   cluster, other namespace", Rules{{ClusterNames: []wfv1.ClusterName{""}, Namespaces: []string{"o"}}}, "", "", false},
		{"other cluster, other namespace", Rules{{ClusterNames: []wfv1.ClusterName{"o"}, Namespaces: []string{"o"}}}, "", "", false},
		{"this  cluster, other namespace", Rules{{ClusterNames: []wfv1.ClusterName{"c"}, Namespaces: []string{"o"}}}, "c", "", false},
		{"other cluster, this  namespace", Rules{{ClusterNames: []wfv1.ClusterName{"o"}, Namespaces: []string{"n"}}}, "c", "n", false},
		{"this  cluster, this  namespace", Rules{{ClusterNames: []wfv1.ClusterName{"c"}, Namespaces: []string{"n"}}}, "c", "n", true},
		{"two  clusters, one   namespace", Rules{{ClusterNames: []wfv1.ClusterName{"c", "d"}, Namespaces: []string{"n"}}}, "c", "n", true},
		{"one   cluster, two  namespaces", Rules{{ClusterNames: []wfv1.ClusterName{"c"}, Namespaces: []string{"n", "m"}}}, "c", "n", true},
		{"two  clusters, two  namespaces", Rules{{ClusterNames: []wfv1.ClusterName{"c", "d"}, Namespaces: []string{"n", "m"}}}, "c", "n", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.rules.Allow(tt.clusterName, tt.namespace))
		})
	}
}
