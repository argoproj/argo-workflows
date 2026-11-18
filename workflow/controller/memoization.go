package controller

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// saveMemoizationCache is shared by pod, task-set, DAG and Steps completion.
func (woc *wfOperationCtx) saveMemoizationCache(ctx context.Context, node *wfv1.NodeStatus, tmpl *wfv1.Template) error {
	if node.MemoizationStatus == nil {
		return nil
	}
	resolveInputs := tmpl == nil
	if resolveInputs {
		var err error
		tmpl, err = woc.GetNodeTemplate(ctx, node)
		if err != nil {
			return fmt.Errorf("failed to get node template for cache save: %w", err)
		}
	}
	if tmpl == nil || tmpl.Memoize == nil {
		return fmt.Errorf("node %s has no memoize template", node.ID)
	}
	if resolveInputs {
		// Pod completion retrieves the original template, not its substituted copy.
		ageTemplate := &wfv1.Template{Memoize: &wfv1.Memoize{MaxAge: tmpl.Memoize.MaxAge}}
		if node.Inputs != nil {
			ageTemplate.Inputs = *node.Inputs
		}
		var err error
		tmpl, err = common.SubstituteParams(ctx, ageTemplate, woc.globalParams(), nil)
		if err != nil {
			return err
		}
	}
	c := woc.controller.cacheFactory.GetCache(node.MemoizationStatus.CacheName)
	return c.Save(ctx, node.MemoizationStatus.Key, node.ID, node.Outputs, tmpl.Memoize.MaxAge)
}
