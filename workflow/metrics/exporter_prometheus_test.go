//go:build !windows

package metrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDisablePrometheusServer(t *testing.T) {
	config := Config{
		Enabled: false,
		Path:    defaultPrometheusServerPath,
		Port:    defaultPrometheusServerPort,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := New(ctx, TestScopeName, &config, Callbacks{})
	if assert.NoError(t, err) {
		go m.RunPrometheusServer(ctx, false)
		time.Sleep(1 * time.Second) // to confirm that the server doesn't start, even if we wait
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", defaultPrometheusServerPort, defaultPrometheusServerPath))
		if resp != nil {
			defer resp.Body.Close()
		}

		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "connection refused") // expect that the metrics server not to start
		}
	}
}

func TestPrometheusServer(t *testing.T) {
	config := Config{
		Enabled: true,
		Path:    defaultPrometheusServerPath,
		Port:    defaultPrometheusServerPort,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := New(ctx, TestScopeName, &config, Callbacks{})
	if assert.NoError(t, err) {
		go m.RunPrometheusServer(ctx, false)
		time.Sleep(1 * time.Second)
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", defaultPrometheusServerPort, defaultPrometheusServerPath))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		bodyString := string(bodyBytes)
		assert.NotEmpty(t, bodyString)
	}
}

func TestDummyPrometheusServer(t *testing.T) {
	config := Config{
		Enabled: true,
		Path:    defaultPrometheusServerPath,
		Port:    defaultPrometheusServerPort,
		Secure:  false,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := New(ctx, TestScopeName, &config, Callbacks{})
	if assert.NoError(t, err) {
		go m.RunPrometheusServer(ctx, true)
		time.Sleep(1 * time.Second)
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", defaultPrometheusServerPort, defaultPrometheusServerPath))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		bodyString := string(bodyBytes)

		assert.Empty(t, bodyString) // expect the dummy metrics server to provide no metrics responses
	}
}
