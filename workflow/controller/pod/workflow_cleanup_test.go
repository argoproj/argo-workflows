package pod

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/util/workqueue"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// Record production's delayed retry while making the next attempt immediately
// available. Tests restore the dependency without publishing any new event.
type immediateCleanupRetryQueue struct {
	workqueue.TypedRateLimitingInterface[string]
	delays []time.Duration
}

func (q *immediateCleanupRetryQueue) AddAfter(key string, delay time.Duration) {
	q.delays = append(q.delays, delay)
	q.Add(key)
}

func workflowCleanupFixture() (*wfv1.Workflow, *apiv1.Pod) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "workflow", Namespace: "default", UID: "workflow-uid", Labels: map[string]string{common.LabelKeyCompleted: "true"}},
		Status: wfv1.WorkflowStatus{
			StartedAt: metav1.NewTime(time.Unix(1000, 0)), Phase: wfv1.WorkflowSucceeded,
			Nodes: wfv1.Nodes{"node": {ID: "node", Name: "workflow.daemon", Type: wfv1.NodeTypePod, Phase: wfv1.NodeSucceeded, CapturedPodUID: "actual-pod"}},
		},
	}
	pod := identityTestPod("actual-pod")
	pod.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(wf, wfv1.SchemeGroupVersion.WithKind("Workflow"))}
	pod.Annotations = map[string]string{common.AnnotationKeyNodeID: "node", common.AnnotationKeyNodeName: "workflow.daemon"}
	pod.Status.Phase = apiv1.PodPending
	return wf, pod
}

func enqueueWorkflowCleanup(ctx context.Context, c *Controller, kind string, wf *wfv1.Workflow, pod *apiv1.Pod) {
	if kind == agentCleanup {
		c.QueueAgentDeletion(ctx, wf, pod.Name)
	} else {
		c.QueueDaemonTermination(ctx, wf, wf.Status.Nodes["node"], pod.Name)
	}
}

func TestWorkflowCleanupIntentWaitsForPersistedDisposition(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, kind := range []string{agentCleanup, daemonCleanup} {
		t.Run(kind, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			ready, pod := workflowCleanupFixture()
			persisted := ready.DeepCopy()
			persisted.Status.Phase = wfv1.WorkflowRunning
			persisted.Labels[common.LabelKeyCompleted] = "false"
			persisted.Status.Nodes["node"] = wfv1.NodeStatus{ID: "node", Name: "workflow.daemon", Phase: wfv1.NodeRunning, Daemoned: new(true)}
			client := fake.NewSimpleClientset(pod)
			c := identityTestController(t, client) // no informer Pod is available
			q := &immediateCleanupRetryQueue{TypedRateLimitingInterface: c.workqueue}
			c.workqueue = q
			c.lookupWorkflow = func(_ context.Context, namespace, name string, hydrateNodes bool) (*wfv1.Workflow, error) {
				assert.Equal(t, ready.Namespace, namespace)
				assert.Equal(t, ready.Name, name)
				// Resolver hydrates daemon nodes; the mutation gate only reads ownership for a Pending Pod.
				if hydrateNodes {
					assert.Equal(t, daemonCleanup, kind)
				}
				return persisted.DeepCopy(), nil
			}
			enqueueWorkflowCleanup(ctx, c, kind, ready, pod)
			require.True(t, c.processNextPodCleanupItem(ctx))
			assert.Equal(t, []time.Duration{podCleanupRetryDelay}, q.delays)
			require.Len(t, client.Actions(), 1)
			assert.Equal(t, "get", client.Actions()[0].GetVerb())

			persisted = ready.DeepCopy()
			require.True(t, c.processNextPodCleanupItem(ctx)) // resolver only enqueues
			require.Len(t, client.Actions(), 2)
			assert.Equal(t, "get", client.Actions()[1].GetVerb())
			require.True(t, c.processNextPodCleanupItem(ctx)) // known UID action
			if kind == daemonCleanup {
				require.True(t, c.processNextPodCleanupItem(ctx)) // Pending termination -> UID-bound delete
			}
			_, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			assert.True(t, apierr.IsNotFound(err))
		})
	}
}

func TestWorkflowCleanupIntentRejectsStaleIdentity(t *testing.T) {
	for _, kind := range []string{agentCleanup, daemonCleanup} {
		for _, change := range []string{"epoch", "workflowUID", "ownerUID", "ownerName", "ownerKind", "ownerAPIVersion", "nodeID", "nodeName"} {
			if kind == agentCleanup && (change == "nodeID" || change == "nodeName") {
				continue
			}
			t.Run(kind+"/"+change, func(t *testing.T) {
				ctx := logging.TestContext(t.Context())
				wf, pod := workflowCleanupFixture()
				expected := wf.DeepCopy()
				switch change {
				case "epoch":
					wf.Status.StartedAt = metav1.NewTime(wf.Status.StartedAt.Add(time.Second))
				case "workflowUID":
					wf.UID = "new-workflow"
				case "ownerUID":
					pod.OwnerReferences[0].UID = "other-workflow"
				case "ownerName":
					pod.OwnerReferences[0].Name = "other-workflow"
				case "ownerKind":
					pod.OwnerReferences[0].Kind = "Other"
				case "ownerAPIVersion":
					pod.OwnerReferences[0].APIVersion = "other.io/v1"
				case "nodeID":
					pod.Annotations[common.AnnotationKeyNodeID] = "other-node"
				case "nodeName":
					pod.Annotations[common.AnnotationKeyNodeName] = "other-node"
				}
				client := fake.NewSimpleClientset(pod)
				c := identityTestController(t, client)
				c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) { return wf.DeepCopy(), nil }
				enqueueWorkflowCleanup(ctx, c, kind, expected, pod)
				require.True(t, c.processNextPodCleanupItem(ctx))
				assert.Zero(t, c.workqueue.Len())
				require.Len(t, client.Actions(), 1)
				assert.Equal(t, "get", client.Actions()[0].GetVerb())
			})
		}
	}
}

func TestWorkflowCleanupIntentRetriesLookupFailures(t *testing.T) {
	for _, failure := range []string{"podForbidden", "podTimeout", "workflowForbidden", "hydration"} {
		t.Run(failure, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, pod := workflowCleanupFixture()
			client := fake.NewSimpleClientset(pod)
			c := identityTestController(t, client)
			q := &immediateCleanupRetryQueue{TypedRateLimitingInterface: c.workqueue}
			c.workqueue = q
			broken := true
			client.PrependReactor("get", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
				if broken && failure == "podForbidden" {
					return true, nil, apierr.NewForbidden(schema.GroupResource{Resource: "pods"}, pod.Name, fmt.Errorf("temporary RBAC denial"))
				}
				if broken && failure == "podTimeout" {
					return true, nil, apierr.NewTimeoutError("temporary API failure", 1)
				}
				return false, nil, nil
			})
			c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) {
				if broken && failure == "workflowForbidden" {
					return nil, apierr.NewForbidden(schema.GroupResource{Resource: "workflows"}, wf.Name, fmt.Errorf("temporary RBAC denial"))
				}
				if broken && failure == "hydration" {
					return nil, fmt.Errorf("offload unavailable")
				}
				return wf.DeepCopy(), nil
			}
			c.QueueDaemonTermination(ctx, wf, wf.Status.Nodes["node"], pod.Name)
			require.True(t, c.processNextPodCleanupItem(ctx))
			assert.Equal(t, []time.Duration{podCleanupRetryDelay}, q.delays)
			broken = false
			require.True(t, c.processNextPodCleanupItem(ctx))
			key, quit := c.workqueue.Get()
			require.False(t, quit)
			defer c.workqueue.Done(key)
			assert.Equal(t, newPodCleanupKeyWithUID(pod.Namespace, pod.Name, terminateContainers, string(pod.UID)), key)
		})
	}
}

func TestWorkflowCleanupIntentLegacyNodeAndAbsence(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := workflowCleanupFixture()
	delete(pod.Annotations, common.AnnotationKeyNodeID)
	node := wf.Status.Nodes["node"]
	delete(wf.Status.Nodes, "node")
	node.ID = wf.NodeID(node.Name)
	wf.Status.Nodes[node.ID] = node
	client := fake.NewSimpleClientset(pod)
	c := identityTestController(t, client)
	c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) { return wf.DeepCopy(), nil }
	c.QueueDaemonTermination(ctx, wf, node, pod.Name)
	require.True(t, c.processNextPodCleanupItem(ctx))
	key, quit := c.workqueue.Get()
	require.False(t, quit)
	c.workqueue.Done(key)
	assert.Equal(t, newPodCleanupKeyWithUID(pod.Namespace, pod.Name, terminateContainers, string(pod.UID)), key)

	require.NoError(t, client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}))
	c.QueueAgentDeletion(ctx, wf, pod.Name)
	require.True(t, c.processNextPodCleanupItem(ctx))
	assert.Zero(t, c.workqueue.Len(), "authoritative Pod absence completes the intent")
}
