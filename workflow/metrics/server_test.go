package metrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisableMetricsServer(t *testing.T) {
	config := ServerConfig{
		Enabled: false,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	m := New(config, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go m.RunServer(ctx, false)
	_, err := http.Get(fmt.Sprintf("http://localhost:%d%s", DefaultMetricsServerPort, DefaultMetricsServerPath))
	assert.Contains(t, err.Error(), "connection refused") // expect that the metrics server not to start
}

func TestSameMetricsServer(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	m := New(config, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go m.RunServer(ctx, false)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", DefaultMetricsServerPort, DefaultMetricsServerPath))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	bodyString := string(bodyBytes)
	assert.NotEmpty(t, bodyString)
}

func TestOwnMetricsServer(t *testing.T) {
	metricsConfig := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	telemetryConfig := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    9091,
	}
	m := New(metricsConfig, telemetryConfig)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go m.RunServer(ctx, false)
	mresp, merr := http.Get(fmt.Sprintf("http://localhost:%d%s", DefaultMetricsServerPort, DefaultMetricsServerPath))
	tresp, terr := http.Get(fmt.Sprintf("http://localhost:%d%s", 9091, DefaultMetricsServerPath))

	assert.NoError(t, merr)
	assert.NoError(t, terr)
	assert.Equal(t, http.StatusOK, mresp.StatusCode)
	assert.Equal(t, http.StatusOK, tresp.StatusCode)

	defer mresp.Body.Close()
	defer tresp.Body.Close()

	mbodyBytes, err := io.ReadAll(mresp.Body)
	tbodyBytes, err := io.ReadAll(tresp.Body)
	assert.NoError(t, err)

	mbodyString := string(mbodyBytes)
	tbodyString := string(tbodyBytes)
	assert.NotEmpty(t, mbodyString)
	assert.NotEmpty(t, tbodyString)
}

func TestDummyMetricsServer(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	m := New(config, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go m.RunServer(ctx, true)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d%s", DefaultMetricsServerPort, DefaultMetricsServerPath))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	bodyString := string(bodyBytes)

	assert.Empty(t, bodyString) // expect the dummy metrics server to provide no metrics responses
}
