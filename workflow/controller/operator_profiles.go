package controller

func (woc *wfOperationCtx) profile(cluster, namespace string, act role) (*profile, error) {
	return woc.controller.profile(woc.wf.Namespace, cluster, namespace, act)
}
