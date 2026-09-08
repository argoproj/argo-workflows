package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricNames(t *testing.T) {
	valid := []string{
		"metric",
		"metric_name",
	}
	for _, name := range valid {
		assert.True(t, IsValidMetricName(name), name)
	}
	invalid := []string{
		"metric.this",
		"metric:this",
		"metric[this]",
	}

	for _, name := range invalid {
		assert.False(t, IsValidMetricName(name), name)
	}
}

func TestReservedMetricNames(t *testing.T) {
	assert.True(t, IsReservedMetricName("retry_strategy_terminations_total"))
	assert.False(t, IsReservedMetricName("RETRY_STRATEGY_TERMINATIONS_TOTAL"))
	assert.False(t, IsReservedMetricName("my_company_retry_strategy_terminations_total"))
}
