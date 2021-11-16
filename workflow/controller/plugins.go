package controller

func (wfc *WorkflowController) getControllerPlugins() []interface{} {
	var out []interface{}
	for _, plug := range wfc.controllerPlugins {
		out = append(out, plug)
	}
	return out
}
