package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

const (
	workflowPodCleanupKeyPrefix = "workflow-pod-cleanup:"
	podCleanupRetryDelay        = 30 * time.Second
	agentCleanup                = "agent"
	daemonCleanup               = "daemon"
)

// A generated-name cleanup intent is not a Pod UID authorization. It may bind
// to a Pod only while its Workflow execution still requests this disposition.
type workflowPodCleanupIntent struct {
	Namespace    string      `json:"namespace"`
	PodName      string      `json:"podName"`
	WorkflowName string      `json:"workflowName"`
	WorkflowUID  types.UID   `json:"workflowUID"`
	StartedAt    metav1.Time `json:"startedAt"`
	Kind         string      `json:"kind"`
	NodeID       string      `json:"nodeID,omitempty"`
	NodeName     string      `json:"nodeName,omitempty"`
}

func (i workflowPodCleanupIntent) valid() bool {
	return i.Namespace != "" && i.PodName != "" && i.WorkflowName != "" && i.WorkflowUID != "" && !i.StartedAt.IsZero() &&
		(i.Kind == agentCleanup || i.Kind == daemonCleanup && i.NodeID != "" && i.NodeName != "")
}

// QueueDaemonTermination retains the stop intent even when the Pod informer has
// not observed the Pod. It waits for the stopped node to be persisted.
func (c *Controller) QueueDaemonTermination(ctx context.Context, wf *wfv1.Workflow, node wfv1.NodeStatus, podName string) {
	c.queueWorkflowPodCleanupIntent(ctx, workflowPodCleanupIntent{
		Namespace: wf.Namespace, PodName: podName, WorkflowName: wf.Name,
		WorkflowUID: wf.UID, StartedAt: wf.Status.StartedAt, Kind: daemonCleanup,
		NodeID: node.ID, NodeName: node.Name,
	})
}

// QueueAgentDeletion retains a completion intent independently of the Pod
// informer and waits for the Workflow's completion to be persisted.
func (c *Controller) QueueAgentDeletion(ctx context.Context, wf *wfv1.Workflow, podName string) {
	c.queueWorkflowPodCleanupIntent(ctx, workflowPodCleanupIntent{
		Namespace: wf.Namespace, PodName: podName, WorkflowName: wf.Name,
		WorkflowUID: wf.UID, StartedAt: wf.Status.StartedAt, Kind: agentCleanup,
	})
}

func (c *Controller) queueWorkflowPodCleanupIntent(ctx context.Context, intent workflowPodCleanupIntent) {
	if !intent.valid() {
		c.log.WithField("podName", intent.PodName).Warn(ctx, "cannot queue workflow pod cleanup without execution identity")
		return
	}
	data, err := json.Marshal(intent)
	if err != nil {
		c.log.WithError(err).Warn(ctx, "cannot encode workflow pod cleanup intent")
		return
	}
	c.workqueue.Add(workflowPodCleanupKeyPrefix + string(data))
}

func (c *Controller) processWorkflowPodCleanupIntent(ctx context.Context, key string) {
	var intent workflowPodCleanupIntent
	if err := json.Unmarshal([]byte(strings.TrimPrefix(key, workflowPodCleanupKeyPrefix)), &intent); err != nil || !intent.valid() {
		c.log.WithField("key", key).Warn(ctx, "ignoring invalid workflow pod cleanup intent")
		c.workqueue.Forget(key)
		return
	}
	retry, err := c.resolveWorkflowPodCleanup(ctx, intent)
	if err != nil {
		c.log.WithError(err).WithField("podName", intent.PodName).Warn(ctx, "failed to resolve workflow pod cleanup intent")
	}
	if retry || err != nil {
		// Include permission, hydration and not-yet-persisted disposition
		// failures: completion may prevent any further Workflow/Pod events.
		c.workqueue.AddAfter(key, podCleanupRetryDelay)
		return
	}
	c.workqueue.Forget(key)
}

func (c *Controller) resolveWorkflowPodCleanup(ctx context.Context, intent workflowPodCleanupIntent) (bool, error) {
	// Read the candidate before the Workflow. A retry that creates a replacement
	// between these reads is then visible in the authoritative execution check.
	pod, err := c.kubeclientset.CoreV1().Pods(intent.Namespace).Get(ctx, intent.PodName, metav1.GetOptions{})
	if apierr.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if c.lookupWorkflow == nil {
		return false, fmt.Errorf("workflow cleanup lookup is unavailable")
	}
	wf, err := c.lookupWorkflow(ctx, intent.Namespace, intent.WorkflowName, intent.Kind == daemonCleanup)
	if apierr.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if wf == nil {
		return false, fmt.Errorf("workflow cleanup lookup returned no workflow")
	}
	if wf.Namespace != intent.Namespace || wf.Name != intent.WorkflowName || wf.UID != intent.WorkflowUID || !wf.Status.StartedAt.Equal(&intent.StartedAt) {
		return false, nil
	}
	owner := metav1.GetControllerOf(pod)
	if owner == nil || owner.Kind != workflow.WorkflowKind || owner.APIVersion != workflow.APIVersion || owner.Name != intent.WorkflowName || owner.UID != intent.WorkflowUID {
		return false, nil
	}
	if pod.Namespace != intent.Namespace || !c.matchesInstance(pod) || !c.matchesInstance(wf) {
		return false, nil
	}
	if pod.UID == "" {
		return false, fmt.Errorf("workflow cleanup pod has no UID")
	}
	var action podCleanupAction
	if intent.Kind == agentCleanup {
		if !wf.Status.Fulfilled() || wf.Labels[common.LabelKeyCompleted] != "true" {
			return true, nil
		}
		action = deletePod
	} else {
		node, ok := wf.Status.Nodes[intent.NodeID]
		if !ok {
			return true, nil
		}
		if node.ID != intent.NodeID || node.Name != intent.NodeName {
			return false, nil
		}
		podNodeID := pod.Annotations[common.AnnotationKeyNodeID]
		if podNodeID == "" {
			podNodeID = wf.ResolveNodeID(pod.Annotations[common.AnnotationKeyNodeName])
		}
		if podNodeID != intent.NodeID || (pod.Annotations[common.AnnotationKeyNodeName] != "" && pod.Annotations[common.AnnotationKeyNodeName] != intent.NodeName) {
			return false, nil
		}
		if node.Phase != wfv1.NodeSucceeded || node.IsDaemoned() {
			return true, nil
		}
		if statusCaptureEnabled() && node.CapturedPodUID != string(pod.UID) {
			return true, nil
		}
		action = terminateContainers
	}
	// Resolving never mutates a Pod. All effects pass through the ordinary
	// UID-bound path, which rejects replacements and guards PATCH/DELETE.
	c.queuePodForCleanup(ctx, intent.Namespace, intent.PodName, action, string(pod.UID))
	return false, nil
}
