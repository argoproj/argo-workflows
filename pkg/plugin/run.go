package plugin

/*
This function is called when controller becomes leader.

It can only be called for Golang plugins, no RPC plugins.

The plugin can implement this to perform async processing.
*/
type RunFunc = func(req RunReq)

type RunReq struct {
	Notify func(namespace, workflowName string) // notify the controller that the workflow needs to be reconciled, e.g. due to async action completing
	Done   <-chan struct{}
}
