package controller

func (woc *wfOperationCtx) profile(cluster string) (*profile, error) {
	return woc.controller.profile(cluster)
}
