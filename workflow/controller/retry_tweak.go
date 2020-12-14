package controller

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfutil "github.com/argoproj/argo/workflow/util"
)

type RetryTweak interface {
	RetryTweak(wfv1.RetryStrategy, wfv1.Nodes, *wfv1.Template)
}

type RetryOnDifferentHost struct {
	retryNodeName string
}

// RetryOnDifferentHost append affinity with fail host to template
func (r *RetryOnDifferentHost) RetryTweak(retryStrategy wfv1.RetryStrategy, allNodes wfv1.Nodes, tmpl *wfv1.Template) {
	hostNames := wfutil.GetFailHosts(allNodes, r.retryNodeName)
	hostLabel := retryStrategy.ScheduleOnDifferentHostNodesLabel
	if hostLabel != nil && len(hostNames) > 0 {
		tmpl.Affinity = wfutil.AddHostnamesToAffinity(*hostLabel, hostNames, tmpl.Affinity)
	}
}
