package common

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metric struct {
	prometheus.Metric
	LastUpdated time.Time
}
