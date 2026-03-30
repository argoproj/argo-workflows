package metrics

import (
	"context"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"

	"github.com/prometheus/client_golang/prometheus"

	envutil "github.com/argoproj/argo-workflows/v4/util/env"
)

const (
	operationDurationDefaultBucketCount = 6
)

func addOperationDurationHistogram(ctx context.Context, m *Metrics) error {
	maxOperationTimeSeconds := envutil.LookupEnvDurationOr(ctx, "MAX_OPERATION_TIME", 30*time.Second).Seconds()
	operationDurationMetricBucketCount := envutil.LookupEnvIntOr(ctx, "OPERATION_DURATION_METRIC_BUCKET_COUNT", operationDurationDefaultBucketCount)
	if operationDurationMetricBucketCount < 1 {
		logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{
			"value":   operationDurationMetricBucketCount,
			"default": operationDurationDefaultBucketCount,
		}).Error(ctx, "Invalid OPERATION_DURATION_METRIC_BUCKET_COUNT value, setting to default")
		operationDurationMetricBucketCount = operationDurationDefaultBucketCount
	}
	bucketWidth := maxOperationTimeSeconds / float64(operationDurationMetricBucketCount)
	// The buckets here are only the 'defaults' and can be overridden with configmap defaults
	return m.CreateBuiltinInstrument(telemetry.InstrumentOperationDurationSeconds,
		telemetry.WithDefaultBuckets(prometheus.LinearBuckets(bucketWidth, bucketWidth, operationDurationMetricBucketCount)))
}

func (m *Metrics) OperationCompleted(ctx context.Context, durationSeconds float64) {
	m.RecordOperationDurationSeconds(ctx, durationSeconds)
}
