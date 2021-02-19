package plugins

import (
	"fmt"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/plugins/http"
)

func New(d config.Plugin) (interface{}, error) {
	switch d.Package {
	case "http":
		return http.New(d.Config.URL), nil
	default:
		panic(fmt.Errorf("unknown plugin package %q", d.Package))
	}
}
