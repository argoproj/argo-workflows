package plugin

/*
This function is called on controller start-up.

The plugin can use this for any one time set-up.

If it returns an error, the controller will exit.
*/
type InitFunc = func(req InitReq, resp *InitResp) error

type InitReq struct {
	Name string `json:"name"` // name of the plugin, plugins/enabled/hello.so is named "hello"
}

type InitResp struct {
	Templates []string `json:"templates,omitempty"` // templates this can process
}
