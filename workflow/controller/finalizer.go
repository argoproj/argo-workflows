package controller

import (
	"context"
	"fmt"
	"strings"

	mclabels "github.com/argoproj-labs/multi-cluster-kubernetes/api/labels"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func (wfc *WorkflowController) newFinalizerController(ctx context.Context) cache.Controller {
	r := util.InstanceIDRequirement(wfc.Config.InstanceID)
	controller := cache.New(&cache.Config{
		Queue: cache.NewFIFO(cache.MetaNamespaceKeyFunc),
		ListerWatcher: cache.NewFilteredListWatchFromClient(wfc.wfclientset.ArgoprojV1alpha1().RESTClient(), "workflows", wfc.managedNamespace, func(opts *metav1.ListOptions) {
			opts.LabelSelector = r.String()
		}),
		Process: func(obj interface{}) error {
			wf, ok := obj.(*wfv1.Workflow)
			if !ok {
				return fmt.Errorf("unexpected type %T", obj)
			}
			return wfc.finalizeWorkflow(ctx, wf)
		},
		ObjectType: &wfv1.Workflow{},
	})
	return controller
}

func (wfc *WorkflowController) finalizeWorkflow(ctx context.Context, wf *wfv1.Workflow) error {
	if wf.GetDeletionTimestamp() == nil {
		return nil
	}
	log.WithField("namespace", wf.Namespace).
		WithField("workflow", wf.Name).
		Info("finalizing workflow")
	woc := newWorkflowOperationCtx(wf, wfc)
	keys := make(map[string]bool)
	for _, tmpl := range woc.execWf.Spec.Templates {
		if tmpl.Cluster != "" || tmpl.Namespace != "" {
			keys[tmpl.ClusterOr(wfc.cluster())+"/"+tmpl.NamespaceOr(woc.wf.Namespace)] = true
		}
	}
	for key := range keys {
		parts := strings.Split(key, "/")
		cluster, namespace := parts[0], parts[1]
		r := util.InstanceIDRequirement(wfc.Config.InstanceID)
		labelSelector :=
			mclabels.KeyOwnerCluster + "=" + wfc.Config.Cluster + "," +
				mclabels.KeyOwnerNamespace + "=" + woc.wf.Namespace + "," +
				common.LabelKeyWorkflow + "=" + woc.wf.Name + ", " +
				r.String()
		log.WithField("cluster", cluster).
			WithField("namespace", namespace).
			WithField("labelSelector", labelSelector).
			Info("deleting pods")
		err := wfc.kubeclientset.Cluster(cluster).CoreV1().Pods(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return fmt.Errorf("failed to finalize workflow %s/%s: %w", wf.Namespace, wf.Name, err)
		}
	}
	controllerutil.RemoveFinalizer(woc.wf, common.FinalizerName)
	woc.updated = true
	woc.persistUpdates(ctx)
	return nil
}
