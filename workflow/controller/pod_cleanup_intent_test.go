package controller

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

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	hydratorfake "github.com/argoproj/argo-workflows/v4/workflow/hydrator/fake"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

type cleanupIntentHydrator struct {
	hydrator.Interface
	hydrate func(*wfv1.Workflow) error
}

func (h cleanupIntentHydrator) Hydrate(_ context.Context, wf *wfv1.Workflow) error {
	return h.hydrate(wf)
}

func TestLookupWorkflowForPodCleanup(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf", Namespace: "default"}, Status: wfv1.WorkflowStatus{OffloadNodeStatusVersion: "persisted-version"}}
	client := fakewfclientset.NewClientset(wf)
	reads := 0
	failure := error(nil)
	wfc := &WorkflowController{wfclientset: client, hydrator: cleanupIntentHydrator{Interface: hydratorfake.Noop, hydrate: func(wf *wfv1.Workflow) error {
		reads++
		assert.Equal(t, "persisted-version", wf.Status.OffloadNodeStatusVersion)
		if failure != nil {
			return failure
		}
		wf.Status.Nodes = wfv1.Nodes{"node": {ID: "node"}}
		return nil
	}}}
	_, err := wfc.lookupWorkflowForPodCleanup(ctx, wf.Namespace, wf.Name, false)
	require.NoError(t, err)
	assert.Zero(t, reads)
	got, err := wfc.lookupWorkflowForPodCleanup(ctx, wf.Namespace, wf.Name, true)
	require.NoError(t, err)
	assert.Equal(t, "node", got.Status.Nodes["node"].ID)
	assert.Equal(t, 1, reads)
	failure = fmt.Errorf("offload unavailable")
	_, err = wfc.lookupWorkflowForPodCleanup(ctx, wf.Namespace, wf.Name, true)
	require.ErrorIs(t, err, failure)
	for _, action := range client.Actions() {
		assert.Equal(t, "get", action.GetVerb())
	}
}

func TestGeneratedCleanupCallersDoNotDependOnPodInformer(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, kind := range []string{"daemon", "agent"} {
		t.Run(kind, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Name: "wf", Namespace: "default", UID: "workflow-uid"},
				Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning, StartedAt: metav1.NewTime(time.Unix(1000, 0)), Nodes: wfv1.Nodes{}},
			}
			node := wfv1.NodeStatus{ID: wf.NodeID("wf.daemon"), Name: "wf.daemon", Type: wfv1.NodeTypePod, Phase: wfv1.NodeRunning, Daemoned: new(true), CapturedPodUID: "pod-uid"}
			if kind == "agent" {
				node.Type, node.Phase, node.Daemoned = wfv1.NodeTypeHTTP, wfv1.NodeSucceeded, nil
			}
			wf.Status.Nodes[node.ID] = node
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			podName := woc.getAgentPodName()
			if kind == "daemon" {
				podName = util.GeneratePodName(wf.Name, node.Name, util.GetTemplateFromNode(node), node.ID, util.GetWorkflowPodNameVersion(wf))
			}
			pod := &apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: podName, Namespace: wf.Namespace, UID: "pod-uid", ResourceVersion: "10",
					OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(wf, wfv1.SchemeGroupVersion.WithKind("Workflow"))},
					Annotations:     map[string]string{common.AnnotationKeyNodeID: node.ID, common.AnnotationKeyNodeName: node.Name},
				},
				Status: apiv1.PodStatus{Phase: apiv1.PodPending},
			}
			// The fake watch does not enforce selectors. Expose the Pod through
			// API reactors without emitting a watch event, so this test really
			// exercises generated intent resolution across an informer miss.
			deleted := false
			client := controller.kubeclientset.(*fake.Clientset)
			client.PrependReactor("get", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
				if action.(clienttesting.GetAction).GetName() != pod.Name {
					return false, nil, nil
				}
				if deleted {
					return true, nil, apierr.NewNotFound(schema.GroupResource{Resource: "pods"}, pod.Name)
				}
				return true, pod.DeepCopy(), nil
			})
			client.PrependReactor("delete", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
				if action.(clienttesting.DeleteAction).GetName() != pod.Name {
					return false, nil, nil
				}
				deleted = true
				return true, nil, nil
			})
			cached, err := controller.PodController.GetPod(wf.Namespace, podName)
			require.NoError(t, err)
			require.Nil(t, cached)
			if kind == "daemon" {
				woc.killDaemonedChildren(ctx, "")
			} else {
				woc.markWorkflowPhase(ctx, wfv1.WorkflowSucceeded, "")
			}
			require.Equal(t, 1, controller.PodController.TestingQueueLen())
			_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, woc.wf, metav1.UpdateOptions{})
			require.NoError(t, err)
			require.True(t, controller.PodController.TestingProcessNextItem(ctx)) // logical intent
			require.True(t, controller.PodController.TestingProcessNextItem(ctx)) // UID-bound action
			if kind == "daemon" {
				require.True(t, controller.PodController.TestingProcessNextItem(ctx)) // Pending delete followup
			}
			_, err = controller.kubeclientset.CoreV1().Pods(wf.Namespace).Get(ctx, podName, metav1.GetOptions{})
			assert.True(t, apierr.IsNotFound(err))
		})
	}
}
