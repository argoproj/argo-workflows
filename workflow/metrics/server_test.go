package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestServerConfig(t *testing.T) {

	cs := ServerConfigs{}
	cs.Add(1, "/foo", &prometheus.Registry{})

	if assert.Contains(t, cs, 1) {
		assert.Contains(t, cs[1], "/foo")
	}

	cs.Add(1, "/bar", &prometheus.Registry{})

	if assert.Contains(t, cs, 1) {
		assert.Contains(t, cs[1], "/foo")
		assert.Contains(t, cs[1], "/bar")
		assert.Equal(t, []string{"/foo", "/bar"}, cs[1].paths())
	}

	cs.Add(2, "/baz", &prometheus.Registry{})

	if assert.Contains(t, cs, 2) {
		assert.Contains(t, cs[2], "/baz")
	}
}
