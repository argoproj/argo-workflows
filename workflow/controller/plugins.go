package controller

import (
	"github.com/argoproj/argo-workflows/v3/workflow/controller/plugins/rpc"
)

func (wfc *WorkflowController) getControllerPlugins() ([]interface{}, error) {
	cms, err := wfc.getConfigMaps(wfc.namespace, "ControllerPlugin")
	if err != nil {
		return nil, err
	}
	plugs := make([]interface{}, len(cms))
	for i, cm := range cms {
		plug, err := rpc.New(cm.Data)
		if err != nil {
			return nil, err
		}
		plugs[i] = plug
	}
	return plugs, nil
}
