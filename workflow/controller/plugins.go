package controller

import (
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/plugins/rpc"
)

func (wfc *WorkflowController) loadPlugins() error {
	objs, err := wfc.configMapInformer.GetIndexer().ByIndex(indexes.ConfigMapLabelsIndex, indexes.ConfigMapIndexValue(wfc.namespace, "ControllerPlugin"))
	if err != nil {
		return err
	}
	log.WithField("num", len(objs)).Info("loading plugins")
	for _, obj := range objs {
		cm := obj.(*corev1.ConfigMap)
		log.WithField("name", cm.Name).Info("loading plugin")
		plug, err := rpc.New(cm.Data)
		if err != nil {
			return err
		}
		wfc.plugins = append(wfc.plugins, plug)
	}
	return nil
}
