package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (woc *wfOperationCtx) profile(cluster, namespace string) (*profile, error) {
	return woc.controller.profile(woc.wf.Namespace, cluster, namespace)
}

func (woc *wfOperationCtx) primaryCluster() string {
	return woc.controller.primaryCluster()
}

func (woc *wfOperationCtx) clusterOf(obj metav1.Object) string {
	return woc.controller.clusterOf(obj)
}
