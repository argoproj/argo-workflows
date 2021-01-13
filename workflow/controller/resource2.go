package controller

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

func (woc *wfOperationCtx) executeResource2(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {

	node := woc.wf.GetNodeByName(nodeName)

	if node != nil && !node.Pending() {
		return node, nil
	}

	un := &unstructured.Unstructured{}
	data, err := yaml.Marshal(tmpl.Resource2)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, un)
	if err != nil {
		return nil, err
	}

	gvk := un.GroupVersionKind()
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)
	if gvr.Empty() {
		return nil, fmt.Errorf("unable to guess group version resource from \"%v\"", gvk)
	}

	node = woc.initializeExecutableNode(
		nodeName,
		wfv1.NodeTypePod,
		templateScope,
		tmpl,
		orgTmpl,
		opts.boundaryID,
		wfv1.NodePending,
		nodeWithGVR(gvr),
	)

	if un.GetName() == "" && un.GetGenerateName() == "" {
		un.SetName(node.ID)
	}

	clusterName := wfv1.ClusterNameOr(tmpl.ClusterName, woc.clusterName())
	namespace := wfv1.NamespaceOr(tmpl.Namespace, woc.wf.Namespace)

	woc.addCoreMetadata(un, nodeName, clusterName, namespace)

	informer, err := woc.controller.resourceInformer(clusterName, namespace, gvr)
	if err != nil {
		return nil, err
	}

	_, exists, err := informer.GetStore().GetByKey(namespace + "/" + un.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to get resource from informer store: %w", err)
	}
	if exists {
		woc.log.Debugf("Skipped resource2 %s (%s) creation: already exists", node.Name, node.ID)
		return node, nil
	}

	dy, err := woc.controller.dynamicInterfaceX(clusterName, namespace)
	if err != nil {
		return node, err
	}

	woc.log.WithFields(log.Fields{"resource": node.Resource, "name": un.GetName()}).Info("creating resource2")

	existing, err := dy.Resource(gvr).Namespace(namespace).Create(ctx, un, metav1.CreateOptions{})
	switch {
	// we additionally check that is it labelled with the workflow
	case apierr.IsAlreadyExists(err) && existing.GetLabels()[common.LabelKeyWorkflow] == un.GetLabels()[common.LabelKeyWorkflow]:
		woc.log.Debugf("Failed resource2 %s (%s) creation: already exists", node.Name, node.ID)
	case err != nil:
		return woc.requeueIfTransientErr(err, node.Name)
	}
	return node, nil
}
