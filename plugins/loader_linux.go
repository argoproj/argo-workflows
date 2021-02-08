package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"

	log "github.com/sirupsen/logrus"
)

var Plugins = make(map[string]plugin.Plugin)

func Load() error {
	err := filepath.Walk("./plugins/enabled", func(path string, f os.FileInfo, err error) error {
		if err != nil || f.IsDir() {
			return err
		}
		log.Infof("loading plugin %q", path)
		p, err := plugin.Open(path)
		if err != nil {
			return fmt.Errorf("plugin %q failed to open: %w", path, err)
		}
		Plugins[path] = *p
		return nil
	})
	return err
}
