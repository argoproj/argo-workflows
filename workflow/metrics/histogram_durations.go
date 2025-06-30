package metrics

import (
	"context"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	envutil "github.com/argoproj/argo-workflows/v3/util/env"
)

const (
	operationDurationDefaultBucketCount = 6
)

func addOperationDurationHistogram(ctx context.Context, m *Metrics) error {
	maxOperationTimeSeconds := envutil.LookupEnvDurationOr(ctx, "MAX_OPERATION_TIME", 30*time.Second).Seconds()
	operationDurationMetricBucketCount := envutil.LookupEnvIntOr(ctx, "OPERATION_DURATION_METRIC_BUCKET_COUNT", operationDurationDefaultBucketCount)
	if operationDurationMetricBucketCount < 1 {
		log.Errorf("Invalid OPERATION_DURATION_METRIC_BUCKET_COUNT value of %d, setting to default %d", operationDurationMetricBucketCount, operationDurationDefaultBucketCount)
		operationDurationMetricBucketCount = operationDurationDefaultBucketCount
	}
	bucketWidth := maxOperationTimeSeconds / float64(operationDurationMetricBucketCount)
	// The buckets here are only the 'defaults' and can be overridden with configmap defaults
	return m.CreateBuiltinInstrument(telemetry.InstrumentOperationDurationSeconds,
		telemetry.WithDefaultBuckets(prometheus.LinearBuckets(bucketWidth, bucketWidth, operationDurationMetricBucketCount)))
}

func (m *Metrics) OperationCompleted(ctx context.Context, durationSeconds float64) {
	m.Record(ctx, telemetry.InstrumentOperationDurationSeconds.Name(), durationSeconds, telemetry.InstAttribs{})
}
