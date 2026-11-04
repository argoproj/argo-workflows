package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

func poolAttribs(namespace, name, hash string) attribute.Set {
	return attribute.NewSet(
		attribute.String("namespace", namespace),
		attribute.String("pool_name", name),
		attribute.String("pool_hash", hash),
	)
}

func TestPoolGauges(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	m, te, err := createTestMetrics(ctx, &telemetry.MetricsConfig{}, Callbacks{})
	require.NoError(t, err)
	samples := []PoolGaugeSample{
		{Namespace: "default", Name: "build", Hash: "abc", Backlog: 3, Claimed: 2, Ready: 4},
		{Namespace: "ns2", Name: "build", Hash: "def", Backlog: 0, Claimed: 1, Ready: 1},
	}
	require.NoError(t, m.RegisterPoolGauges(func(_ context.Context) []PoolGaugeSample { return samples }))

	for _, s := range samples {
		attribs := poolAttribs(s.Namespace, s.Name, s.Hash)
		backlog, gaugeErr := te.GetInt64GaugeValue(ctx, telemetry.InstrumentPoolBacklog.Name(), &attribs)
		require.NoError(t, gaugeErr)
		assert.Equal(t, s.Backlog, backlog, "%s backlog", s.Hash)
		claimed, gaugeErr := te.GetInt64GaugeValue(ctx, telemetry.InstrumentPoolClaimedTasks.Name(), &attribs)
		require.NoError(t, gaugeErr)
		assert.Equal(t, s.Claimed, claimed, "%s claimed", s.Hash)
		ready, gaugeErr := te.GetInt64GaugeValue(ctx, telemetry.InstrumentPoolReadyPods.Name(), &attribs)
		require.NoError(t, gaugeErr)
		assert.Equal(t, s.Ready, ready, "%s ready", s.Hash)
	}

	// A pool the callback no longer reports has no series at the next scrape.
	samples = samples[1:]
	gone := poolAttribs("default", "build", "abc")
	for _, name := range []string{
		telemetry.InstrumentPoolBacklog.Name(),
		telemetry.InstrumentPoolClaimedTasks.Name(),
		telemetry.InstrumentPoolReadyPods.Name(),
	} {
		_, gaugeErr := te.GetInt64GaugeValue(ctx, name, &gone)
		require.Error(t, gaugeErr, name)
	}
	kept := poolAttribs("ns2", "build", "def")
	ready, err := te.GetInt64GaugeValue(ctx, telemetry.InstrumentPoolReadyPods.Name(), &kept)
	require.NoError(t, err)
	assert.Equal(t, int64(1), ready)
}

func TestPoolGaugesNoPools(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	m, te, err := createTestMetrics(ctx, &telemetry.MetricsConfig{}, Callbacks{})
	require.NoError(t, err)
	require.NoError(t, m.RegisterPoolGauges(func(_ context.Context) []PoolGaugeSample { return nil }))

	attribs := poolAttribs("default", "build", "abc")
	_, err = te.GetInt64GaugeValue(ctx, telemetry.InstrumentPoolBacklog.Name(), &attribs)
	require.Error(t, err)
}
