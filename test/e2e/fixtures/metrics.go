package fixtures

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
)

// MetricBaseline stores baseline metric values for comparison
type MetricBaseline struct {
	t                 *testing.T
	httpClient        func() *httpexpect.Expect
	baselines         map[string]float64
	expectedIncreases map[string]float64
}

// NewMetricBaseline creates a new metric baseline tracker
func NewMetricBaseline(t *testing.T, httpClient func() *httpexpect.Expect) *MetricBaseline {
	return &MetricBaseline{
		t:                 t,
		httpClient:        httpClient,
		baselines:         make(map[string]float64),
		expectedIncreases: make(map[string]float64),
	}
}

// getCurrentMetricValues fetches current values for the specified metrics
func (mb *MetricBaseline) getCurrentMetricValues(metricsMap map[string]float64) map[string]float64 {
	mb.t.Helper()

	body := mb.httpClient().GET("").
		Expect().
		Status(200).
		Body().
		Raw()

	values := make(map[string]float64)
	for metric := range metricsMap {
		values[metric] = parseMetricValue(body, metric)
	}

	return values
}

// CaptureBaseline captures baselines for all metrics in the expected increases map
// and stores the expected increases for later use with ExpectIncrease()
func (mb *MetricBaseline) CaptureBaseline(expectedIncreases map[string]float64) *MetricBaseline {
	mb.t.Helper()

	// Capture current metric values and store them as baselines
	currentValues := mb.getCurrentMetricValues(expectedIncreases)

	for metric := range expectedIncreases {
		value := currentValues[metric]
		mb.baselines[metric] = value
		mb.t.Logf("Baseline for %s: %f", metric, value)
	}

	// Store the expected increases for later use
	mb.expectedIncreases = expectedIncreases

	return mb
}

// ExpectIncrease checks that the metrics have increased by the amounts
// specified in the map passed to CaptureBaseline()
func (mb *MetricBaseline) ExpectIncrease() {
	mb.t.Helper()

	if len(mb.expectedIncreases) == 0 {
		mb.t.Fatal("No expected increases stored. Call CaptureBaseline() first.")
	}

	currentValues := mb.getCurrentMetricValues(mb.expectedIncreases)

	for metric, expectedIncrease := range mb.expectedIncreases {
		baseline := mb.baselines[metric] // defaults to 0 if not found
		currentValue := currentValues[metric]
		actualIncrease := currentValue - baseline

		mb.t.Logf("Metric %s: baseline=%f, current=%f, expected_increase=%f, actual_increase=%f",
			metric, baseline, currentValue, expectedIncrease, actualIncrease)

		assert.InDelta(mb.t, expectedIncrease, actualIncrease, 0.001,
			"Expected %s to increase by %f, but it increased by %f (from %f to %f)", metric, expectedIncrease, actualIncrease, baseline, currentValue)
	}
}

// parseMetricValue extracts the numeric value from a prometheus metric line
// Returns 0 if the metric is not found
func parseMetricValue(body, metricPattern string) float64 {
	// Escape special regex characters in the metric pattern, but keep the spaces
	// We'll look for lines that match the pattern and extract the value
	lines := strings.SplitSeq(body, "\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if this line matches our metric pattern
		if strings.Contains(line, metricPattern) {
			// Extract the value using regex
			// Prometheus format: metric_name{labels} value
			re := regexp.MustCompile(`\s+([0-9.]+)$`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				value, err := strconv.ParseFloat(matches[1], 64)
				if err == nil {
					return value
				}
			}
		}
	}

	return 0 // Default to 0 if metric not found
}
