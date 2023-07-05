package metrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunServer(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	m := New(config, config)

	server := func(isDummy bool, shouldBodyBeEmpty bool) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go m.RunServer(ctx, isDummy)

		resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", DefaultMetricsServerPort, DefaultMetricsServerPath))
		assert.NoError(t, err)
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		bodyString := string(bodyBytes)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		if shouldBodyBeEmpty {
			assert.Empty(t, bodyString)
		} else {
			assert.NotEmpty(t, bodyString)
		}
	}

	t.Run("dummy metrics server", func(t *testing.T) {
		server(true, true) // dummy metrics server does not provide any metrics responses
	})

	t.Run("prometheus metrics server", func(t *testing.T) {
		server(false, false) // prometheus metrics server provides responses for any metrics
	})
}
