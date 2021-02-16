package plugins

import (
	"fmt"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/plugins/httptemplate"
	"github.com/argoproj/argo-workflows/v3/plugins/jsonrpc"
)

func New(d config.PluginDef) (interface{}, error) {
	switch d.Package {
	case "httptemplate":
		return httptemplate.New(), nil
	case "jsonrpc":
		return jsonrpc.New(d.Config.Host), nil
	default:
		panic(fmt.Errorf("unknown plugin package %q", d.Package))
	}
}
