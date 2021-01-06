package controller

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfretry "github.com/argoproj/argo/workflow/util/retry"
)

// RetryTweak is a 2nd order function interface for tweaking the retry
type RetryTweak = func(retryStrategy wfv1.RetryStrategy, nodes wfv1.Nodes, tmpl *wfv1.Template)

// RetryOnDifferentHost append affinity with fail host to template
func RetryOnDifferentHost(retryNodeName string) RetryTweak {
	return func(retryStrategy wfv1.RetryStrategy, nodes wfv1.Nodes, tmpl *wfv1.Template) {
		if retryStrategy.Affinity == nil {
			return
		}
		hostNames := wfretry.GetFailHosts(nodes, retryNodeName)
		hostLabel := "kubernetes.io/hostname"
		if hostLabel != "" && len(hostNames) > 0 {
			tmpl.Affinity = wfretry.AddHostnamesToAffinity(hostLabel, hostNames, tmpl.Affinity)
		}
	}
}
