package common

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metric struct {
	Metric      prometheus.Metric
	LastUpdated time.Time
}
