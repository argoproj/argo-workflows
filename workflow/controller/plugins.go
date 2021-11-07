package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"sigs.k8s.io/yaml"

	log "github.com/sirupsen/logrus"
)

type pluginManifest struct {
	Path string                 `json:"path,omitempty"`
	Spec map[string]interface{} `json:"spec,omitempty"`
}

func (wfc *WorkflowController) loadPlugins(dir string) error {
	log.WithField("dir", dir).Info("loading plugins")
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		path := filepath.Join(dir, f.Name())
		if !strings.HasSuffix(path, ".yaml") {
			continue
		}
		log.WithField("path", path).Info("loading plugin")
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return err
		}
		spec := &pluginManifest{}
		if err := yaml.Unmarshal(data, spec); err != nil {
			return err
		}
		plug, err := plugin.Open(filepath.Join(dir, spec.Path))
		if err != nil {
			return err
		}
		f, err := plug.Lookup("New")
		if err != nil {
			return err
		}
		newFunc, ok := f.(func(map[string]interface{}) (interface{}, error))
		if !ok {
			return fmt.Errorf("plugin %q does not export `func New() interface{}`", path)
		}
		sym, err := newFunc(spec.Spec)
		if err != nil {
			return err
		}
		wfc.plugins = append(wfc.plugins, sym)
	}
	return nil
}
