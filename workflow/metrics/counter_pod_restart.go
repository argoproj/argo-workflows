package metrics

import (
	"context"
	"regexp"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

// conditionRegex extracts the condition from pod status messages like
// "The node had condition: [DiskPressure]" -> "DiskPressure"
var conditionRegex = regexp.MustCompile(`\[(\w+)\]`)

func addPodRestartCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentPodRestartsTotal)
}

// RecordPodRestart records a metric when a pod is automatically restarted
// due to an infrastructure failure.
func (m *Metrics) RecordPodRestart(ctx context.Context, reason, message, namespace string) {
	condition := extractConditionFromMessage(message)

	opts := []telemetry.PodRestartsTotalOption{}
	if condition != "" {
		opts = append(opts, telemetry.WithPodRestartCondition(condition))
	}

	m.AddPodRestartsTotal(ctx, 1, reason, namespace, opts...)
}

// extractConditionFromMessage extracts the node condition from the pod status message.
// For example, "The node had condition: [DiskPressure]" returns "DiskPressure".
// Returns empty string if no condition can be extracted.
func extractConditionFromMessage(message string) string {
	matches := conditionRegex.FindStringSubmatch(message)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}
