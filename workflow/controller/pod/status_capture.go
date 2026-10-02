package pod

import (
	"context"
	"fmt"
	"os"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

type podCleanupState struct {
	workflow *wfv1.Workflow
	node     *wfv1.NodeStatus
	orphan   bool
	ignored  bool
}

func statusCaptureEnabled() bool {
	return os.Getenv(common.EnvVarPodStatusCaptureFinalizer) == "true"
}

func terminalPod(pod *apiv1.Pod) bool {
	return pod.Status.Phase == apiv1.PodSucceeded || pod.Status.Phase == apiv1.PodFailed
}

func (c *Controller) matchesInstance(obj metav1.Object) bool {
	instanceID := ""
	if c.config != nil {
		instanceID = c.config.InstanceID
	}
	requirement := util.InstanceIDRequirement(instanceID)
	return requirement.Matches(labels.Set(obj.GetLabels()))
}

// SetWorkflowHydrator installs the controller-owned decoder before Run. It
// hydrates the API object already read for this observation, using exactly its
// published node reference; it does not cache or reuse state across work items.
func (c *Controller) SetWorkflowHydrator(hydrate func(context.Context, *wfv1.Workflow) error) {
	c.hydrateWorkflow = hydrate
}

// Read ownership before node storage. An absent/replaced/deleting owner needs
// no node hydration; a cache miss or storage/read error is never an orphan.
func (c *Controller) observePodCleanup(ctx context.Context, pod *apiv1.Pod, needNodes bool) (*podCleanupState, error) {
	state := &podCleanupState{}
	if !c.matchesInstance(pod) {
		state.ignored = true
		return state, nil
	}
	owner := metav1.GetControllerOf(pod)
	if owner == nil || owner.Kind != workflow.WorkflowKind || owner.APIVersion != workflow.APIVersion || owner.UID == "" {
		return nil, StatusCaptureHold(CaptureIdentityConflict, "status capture owner identity is unavailable")
	}
	if label, ok := pod.Labels[common.LabelKeyWorkflow]; ok && label != owner.Name {
		state.ignored = true
		return state, nil
	}
	if c.lookupWorkflow == nil {
		return nil, fmt.Errorf("workflow cleanup lookup is unavailable")
	}
	read := func(hydrate bool) error {
		wf, err := c.lookupWorkflow(ctx, pod.Namespace, owner.Name, hydrate)
		if apierr.IsNotFound(err) && !hydrate {
			state.orphan = true
			return nil
		}
		if err != nil {
			return err
		}
		if wf == nil || wf.Name != owner.Name || wf.Namespace != pod.Namespace {
			return fmt.Errorf("workflow cleanup lookup returned a different object")
		}
		state.workflow = wf
		if wf.UID != owner.UID {
			state.orphan = true
			return nil
		}
		if !c.matchesInstance(wf) {
			state.ignored = true
		} else if wf.DeletionTimestamp != nil {
			state.orphan = true
		}
		return nil
	}
	if err := read(false); err != nil {
		return nil, err
	}
	if state.orphan || state.ignored || !needNodes || pod.Labels[common.LabelKeyComponent] != "" {
		return state, nil
	}
	if state.workflow.Status.CompressedNodes != "" || state.workflow.Status.IsOffloadNodeStatus() {
		if c.hydrateWorkflow != nil {
			// Ownership and the referenced node version belong to this one
			// authoritative read. The mutation worker will read again.
			if err := c.hydrateWorkflow(ctx, state.workflow); err != nil {
				return nil, err
			}
		} else {
			// Compatibility for callers without a local hydrator: the second
			// authoritative response must revalidate its ownership as before.
			if err := read(true); err != nil {
				return nil, err
			}
		}
		if state.orphan || state.ignored {
			return state, nil
		}
		if state.workflow.Status.CompressedNodes != "" || state.workflow.Status.IsOffloadNodeStatus() {
			return nil, fmt.Errorf("workflow cleanup node status was not hydrated")
		}
	}
	nodeID := pod.Annotations[common.AnnotationKeyNodeID]
	if nodeID == "" {
		name := pod.Annotations[common.AnnotationKeyNodeName]
		if name == "" {
			return nil, StatusCaptureHold(CaptureIdentityConflict, "status capture node identity is unavailable")
		}
		nodeID = state.workflow.ResolveNodeID(name)
	}
	node, err := state.workflow.Status.Nodes.Get(nodeID)
	if err != nil {
		return nil, StatusCaptureHold(CaptureResultPending, "status capture node has not been persisted")
	}
	if node.ID != nodeID || (pod.Annotations[common.AnnotationKeyNodeName] != "" && node.Name != pod.Annotations[common.AnnotationKeyNodeName]) {
		return nil, StatusCaptureHold(CaptureIdentityConflict, "status capture node identity does not match pod")
	}
	state.node = node
	return state, nil
}

func restartDisposition(pod *apiv1.Pod, state *podCleanupState) bool {
	return state.node != nil && pod.Status.Phase == apiv1.PodFailed && state.node.Phase == wfv1.NodePending && state.node.RestartingPodUID == string(pod.UID)
}

func capturedDisposition(pod *apiv1.Pod, state *podCleanupState) error {
	if state.node == nil {
		return StatusCaptureHold(CaptureResultPending, "status capture node is unavailable")
	}
	if restartDisposition(pod, state) {
		return nil
	}
	if state.node.CapturedPodUID != string(pod.UID) {
		if state.node.CapturedPodUID != "" {
			return StatusCaptureHold(CaptureIdentityConflict, "status capture identifies a different Pod UID")
		}
		if state.node.CapturedPodUID == "" && state.node.Phase.Fulfilled(state.node.TaskResultSynced) && state.workflow.Status.Fulfilled() && state.workflow.Labels[common.LabelKeyCompleted] == "true" {
			return StatusCaptureHold(CaptureLegacyUnsupported, "status capture provenance is missing; no supported legacy capture proof")
		}
		return StatusCaptureHold(CaptureResultPending, "status capture has not been persisted for this pod UID")
	}
	if !state.node.Phase.Fulfilled(state.node.TaskResultSynced) || state.workflow.Status.IsTaskResultIncomplete(state.node.ID) {
		return StatusCaptureHold(CaptureResultPending, "status capture result or task synchronization is pending")
	}
	return nil
}

// This gate covers every queue action that could remove a present status
// barrier, including completion labeling. Pods that never had the barrier and
// feature-disabled controllers retain their existing cleanup behavior.
func (c *Controller) allowPodCleanup(ctx context.Context, pod *apiv1.Pod) (bool, error) {
	if !statusCaptureEnabled() || !hasOurFinalizer(pod.Finalizers) {
		return true, nil
	}
	state, err := c.observePodCleanup(ctx, pod, terminalPod(pod))
	if err != nil {
		return false, err
	}
	if state.ignored {
		return false, nil
	}
	if state.orphan || !terminalPod(pod) {
		// No terminal execution result exists for nonterminal deletion. This
		// includes Pending cancellation and controller-directed termination.
		return true, nil
	}
	// Agent results live in the WorkflowTaskSet and have no Pod node. The
	// completed owner disposition is its existing deletion contract.
	if pod.Labels[common.LabelKeyComponent] == "agent" {
		if state.workflow.Status.Fulfilled() && state.workflow.Labels[common.LabelKeyCompleted] == "true" {
			return true, nil
		}
		return false, StatusCaptureHold(CaptureResultPending, "agent cleanup is awaiting workflow completion")
	}
	if err := c.ensureCapturedDisposition(ctx, pod, state); err != nil {
		return false, err
	}
	return true, nil
}
