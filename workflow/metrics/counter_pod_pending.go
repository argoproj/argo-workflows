package metrics

import (
	"context"
	"strings"
)

const (
	namePodPending = `pod_pending_count`
)

func addPodPendingCounter(_ context.Context, m *Metrics) error {
	return m.createInstrument(int64Counter,
		namePodPending,
		"Total number of pods that started pending by reason",
		"{pod}",
		withAsBuiltIn(),
	)
}

func (m *Metrics) ChangePodPending(ctx context.Context, reason, namespace string) {
	// Reason strings have a lot of stuff that would result in insane cardinatlity
	// so we just take everything from before the first :
	splitReason := strings.Split(reason, `:`)
	switch splitReason[0] {
	case "PodInitializing":
		// Drop these, they are uninteresting and usually short
		// the pod_phase metric can cope with this being visible
		return
	default:
		m.addInt(ctx, namePodPending, 1, instAttribs{
			{name: labelPodPendingReason, value: splitReason[0]},
			{name: labelPodNamespace, value: namespace},
		})
	}
}
