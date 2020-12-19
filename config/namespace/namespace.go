package namespace

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util/slice"
)

type Namespaces []Namespace

func (c Namespaces) Find(namespace string) Namespace {
	for _, x := range c {
		if x.Name == namespace {
			return x
		}
	}
	return Namespace{}
}

type Namespace struct {
	// Which namespace the rules apply to.
	Name  string `json:"name"`
	Rules Rules  `json:"rules"`
}

type Rules []Rule

func (r Rules) Allow(clusterName wfv1.ClusterName, namespace string) bool {
	for _, x := range r {
		if (slice.ContainsString(x.ClusterNames, clusterName) || slice.ContainsString(x.ClusterNames, "*")) &&
			(slice.ContainsString(x.Namespaces, namespace) || slice.ContainsString(x.Namespaces, "*")) {
			return true
		}
	}
	return false
}

type Rule struct {
	// Which clusters. Empty string means any.
	ClusterNames []wfv1.ClusterName `json:"clusterNames"`
	// Which namespaces. Empty string means any.
	Namespaces []string `json:"namespaces"`
}
