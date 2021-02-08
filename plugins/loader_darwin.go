package plugins

import (
	"plugin"
)

var Plugins = make(map[string]plugin.Plugin)

func Load() error {
	return nil
}
