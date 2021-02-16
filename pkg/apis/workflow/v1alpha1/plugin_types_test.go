package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

type httpTemplate struct {
	Foo string `json:"foo"`
}

func TestPluginTemplate(t *testing.T) {
	t.Run("Get", func(t *testing.T) {
		p := PluginTemplate{Value: json.RawMessage(`{"http": {"foo": "bar"}}`)}
		h := &httpTemplate{}
		err := p.UnmarshalTo(h)
		assert.NoError(t, err)
		assert.NotEmpty(t, h)
	})
	t.Run("Unmarshall", func(t *testing.T) {
		w := &Workflow{}
		err := yaml.Unmarshal([]byte(`
spec:
  templates:
   - name: main
     plugin: {}
`), w)
		assert.NoError(t, err)
		tmpl := w.GetTemplateByName("main")
		assert.Equal(t, TemplateTypePlugin, tmpl.GetType())
	})
}
