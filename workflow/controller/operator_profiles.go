package controller

func (woc *wfOperationCtx) profile(cluster, namespace string) (*profile, error) {
	return woc.controller.profile(woc.wf.Namespace, cluster, namespace)
}
