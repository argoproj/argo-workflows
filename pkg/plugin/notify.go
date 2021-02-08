package plugin

/*
This function can be called by a plugin to notify the controller of a change that happened externally to it.

It is not intended to be implemented by the plugin. Instead, the workflow controller implements it.
*/
type NotifyFunc = func(req NotifyReq, resp *NotifyResp) error

type NotifyReq struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"` // workflow name
}

type NotifyResp struct{}
