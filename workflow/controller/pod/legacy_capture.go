package pod

import (
	"context"

	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// LegacyPodRecapture commits a newly verified, unchanged legacy result. Its
// return value alone never permits cleanup; the Pod worker rereads the owner.
type LegacyPodRecapture func(context.Context, *apiv1.Pod) (*wfv1.Workflow, error)

// SetLegacyPodRecapture installs the completed-result verifier before Run.
func (c *Controller) SetLegacyPodRecapture(recapture LegacyPodRecapture) {
	c.recaptureLegacy = recapture
}

func (c *Controller) ensureCapturedDisposition(ctx context.Context, pod *apiv1.Pod, state *podCleanupState) error {
	if capturedDisposition(pod, state) == nil {
		return nil
	}
	if c.recaptureLegacy == nil || state.node == nil || state.node.CapturedPodUID != "" ||
		!state.workflow.Status.Fulfilled() || state.workflow.Labels[common.LabelKeyCompleted] != "true" ||
		!state.node.Phase.Fulfilled(state.node.TaskResultSynced) || state.workflow.Status.IsTaskResultIncomplete(state.node.ID) {
		return capturedDisposition(pod, state)
	}
	// The verifier requires exactly one node. Reject an already incompatible
	// snapshot before starting another read/hydrate transaction; retry still
	// rereads authoritative state and no result or receipt is changed here.
	if len(state.workflow.Status.Nodes) != 1 {
		return StatusCaptureHold(CaptureLegacyUnsupported, "legacy capture supports only a completed single-node successful Workflow")
	}
	if _, err := c.recaptureLegacy(ctx, pod); err != nil {
		return err
	}
	// Recheck the published representation, not the callback's in-memory object
	// or an unreferenced offload write. The mutation still uses the observed Pod
	// UID/resourceVersion and the existing shared barrier.
	observed, err := c.observePodCleanup(ctx, pod, true)
	if err != nil {
		return err
	}
	if observed.orphan || observed.ignored {
		return StatusCaptureHold(CaptureIdentityConflict, "workflow ownership changed during legacy capture; retry cleanup")
	}
	*state = *observed
	return capturedDisposition(pod, state)
}
