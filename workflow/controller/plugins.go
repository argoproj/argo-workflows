package controller

import (
	"fmt"
	"path/filepath"
	"plugin"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
)

func (wfc *WorkflowController) loadPlugins(dir string) error {
	objs, err := wfc.configMapInformer.GetIndexer().ByIndex(indexes.ConfigMapLabelsIndex, indexes.ConfigMapIndexValue(wfc.namespace, "ControllerPlugin"))
	if err != nil {
		return err
	}
	log.WithField("num", len(objs)).Info("loading plugins")
	for _, obj := range objs {
		cm := obj.(*corev1.ConfigMap)
		path := filepath.Join(dir, cm.Data["path"])
		log.WithField("path", path).Info("loading plugin")
		plug, err := plugin.Open(path)
		if err != nil {
			return err
		}
		f, err := plug.Lookup("New")
		if err != nil {
			return err
		}
		newFunc, ok := f.(func(map[string]string) (interface{}, error))
		if !ok {
			return fmt.Errorf("plugin %q does not export `func New() interface{}`", path)
		}
		sym, err := newFunc(cm.Data)
		if err != nil {
			return err
		}
		wfc.plugins = append(wfc.plugins, sym)
	}
	return nil
}
