package controller

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// saveMemoizationCache is shared by pod, task-set, DAG and Steps completion.
func (woc *wfOperationCtx) saveMemoizationCache(ctx context.Context, node *wfv1.NodeStatus) error {
	status := node.MemoizationStatus
	if status == nil {
		return nil
	}
	c := woc.controller.cacheFactory.GetCache(status.CacheName)
	return c.Save(ctx, status.Key, node.ID, node.Outputs, status.MaxAge)
}
