package controller

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func (woc *wfOperationCtx) executeResource2(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {

	node := woc.wf.GetNodeByName(nodeName)

	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePod, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	} else if !node.Pending() {
		return node, nil
	}

	tmpl = tmpl.DeepCopy()

	un := &unstructured.Unstructured{}
	data, err := yaml.Marshal(tmpl.Resource2)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, un)
	if err != nil {
		return nil, err
	}

	if un.GetName() == "" && un.GetGenerateName() == "" {
		un.SetName(node.ID)
	}

	clusterName := wfv1.ClusterNameOr(tmpl.ClusterName, woc.clusterName())
	namespace := wfv1.NamespaceOr(tmpl.Namespace, woc.wf.Namespace)

	woc.addCoreMetadata(un, nodeName, clusterName, namespace)

	gvr, _ := meta.UnsafeGuessKindToResource(un.GroupVersionKind())

	informer, err := woc.controller.resourceInformer(clusterName, gvr, namespace)
	if err != nil {
		return nil, err
	}

	_, exists, err := informer.GetStore().Get(cache.ExplicitKey(namespace + "/" + un.GetName()))
	if err != nil {
		return nil, fmt.Errorf("failed to get resource from informer store: %w", err)
	}
	if exists {
		woc.log.Debugf("Skipped resource2 %s (%s) creation: already exists", node.Name, node.ID)
		return node, nil
	}

	dy, err := woc.controller.dynamicInterfaceX(clusterName, gvr, namespace)
	if err != nil {
		return node, err
	}

	woc.log.WithFields(log.Fields{"gvr": gvr, "name": un.GetName()}).Info("creating resource2")

	_, err = dy.Namespace(namespace).Create(ctx, un, metav1.CreateOptions{})
	switch {
	case apierr.IsAlreadyExists(err):
		woc.log.Debugf("Failed resource2 %s (%s) creation: already exists", node.Name, node.ID)
	case err != nil:
		return woc.requeueIfTransientErr(err, node.Name)
	}
	return node, err
}
