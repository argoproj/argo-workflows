package metrics

import (
	"context"
	"strings"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func addPodPendingCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentPodPendingCount)
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
		m.AddInt(ctx, telemetry.InstrumentPodPendingCount.Name(), 1, telemetry.InstAttribs{
			{Name: telemetry.AttribPodPendingReason, Value: splitReason[0]},
			{Name: telemetry.AttribPodNamespace, Value: namespace},
		})
	}
}
