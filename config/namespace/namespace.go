package namespace

import (
	corev1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util/slice"
)

type Roles []Role

func (c Roles) Find(namespace string) Role {
	for _, x := range c {
		if x.Namespace == namespace {
			return x
		}
	}
	return Role{}
}

type Role struct {
	// Which namespace the rules apply to.
	Namespace string `json:"namespace"`
	Rules     Rules  `json:"rules"`
}

func (r Role) IsEmpty() bool {
	return r.Namespace == ""
}

type Rules []Rule

func (r Rules) Allow(clusterName wfv1.ClusterName, namespace string) bool {
	for _, x := range r {
		if (slice.ContainsString(x.ClusterNames, clusterName) || slice.ContainsString(x.ClusterNames, wfv1.ClusterAll)) &&
			(slice.ContainsString(x.Namespaces, namespace) || slice.ContainsString(x.Namespaces, corev1.NamespaceAll)) {
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
