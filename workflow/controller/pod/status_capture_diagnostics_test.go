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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestStatusCaptureLegacyHookRequiresPublishedReceipt(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprint(committed), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, p := captureFixture()
			n := wf.Status.Nodes["node"]
			n.CapturedPodUID = ""
			wf.Status.Nodes["node"] = n
			c := identityTestController(t, fake.NewClientset(p))
			c.lookupWorkflow = func(context.Context, string, string, bool) (*wfv1.Workflow, error) { return wf.DeepCopy(), nil }
			calls := 0
			c.SetLegacyPodRecapture(func(context.Context, *apiv1.Pod) (*wfv1.Workflow, error) {
				calls++
				returned := wf.DeepCopy()
				n := returned.Status.Nodes["node"]
				n.CapturedPodUID = string(p.UID)
				returned.Status.Nodes["node"] = n
				if committed {
					wf = returned.DeepCopy()
				}
				return returned, nil
			})
			allowed, err := c.allowPodCleanup(ctx, p)
			require.Equal(t, 1, calls)
			assert.Equal(t, committed, allowed)
			if committed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestStatusCaptureDiagnosticBacklogKeepsRetry(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	const backlog = 8
	workflows := map[string]*wfv1.Workflow{}
	objects := []runtime.Object{}
	keys := []string{}
	for i := range backlog {
		wf, p := captureFixture()
		wf.Name = fmt.Sprintf("legacy-%d", i)
		wf.UID = types.UID(wf.Name)
		p.Name = wf.Name
		p.UID = types.UID(fmt.Sprintf("pod-%d", i))
		p.OwnerReferences[0].Name = wf.Name
		p.OwnerReferences[0].UID = wf.UID
		p.Labels[common.LabelKeyWorkflow] = wf.Name
		n := wf.Status.Nodes["node"]
		n.CapturedPodUID = ""
		wf.Status.Nodes["node"] = n
		workflows[wf.Name] = wf
		objects = append(objects, p)
		keys = append(keys, newPodCleanupKeyWithUID(p.Namespace, p.Name, removeFinalizer, string(p.UID)))
	}
	client := fake.NewClientset(objects...)
	c := identityTestController(t, client)
	clock := captureTimingQueue(t, c)
	hook := logging.NewTestHook()
	c.log = logging.NewTestLogger(logging.Info, logging.Text, hook)
	ctx := logging.WithLogger(t.Context(), c.log)
	deleted := false
	reads := 0
	c.lookupWorkflow = func(_ context.Context, _, name string, _ bool) (*wfv1.Workflow, error) {
		reads++
		if deleted {
			return nil, apierr.NewNotFound(schema.GroupResource{Resource: "workflows"}, name)
		}
		return workflows[name].DeepCopy(), nil
	}
	for _, key := range keys {
		c.workqueue.Add(key)
	}
	for round := range 4 {
		require.Eventually(t, func() bool { return c.workqueue.Len() == backlog }, time.Second, time.Millisecond)
		for range backlog {
			require.True(t, c.processNextPodCleanupItem(ctx))
		}
		assert.Equal(t, backlog*(round+1), reads)
		assert.Zero(t, captureTimingVerbCounts(client)["patch"])
		warnings := 0
		for _, entry := range hook.AllEntries() {
			if entry.Level == logging.Warn {
				warnings++
				assert.Equal(t, CaptureLegacyUnsupported, entry.Fields["captureReason"])
			}
		}
		assert.Equal(t, backlog, warnings, "unchanged holds do not warn on every retry")
		captureTimingWaitScheduled(t, clock)
		if round < 3 {
			clock.Step(podCleanupRetryDelay)
		}
	}
	deleted = true // authoritative owner disappearance, with no Pod event or worker restart
	clock.Step(podCleanupRetryDelay)
	require.Eventually(t, func() bool { return c.workqueue.Len() == backlog }, time.Second, time.Millisecond)
	for range backlog {
		require.True(t, c.processNextPodCleanupItem(ctx))
	}
	assert.Equal(t, backlog, captureTimingVerbCounts(client)["patch"])
	for _, obj := range objects {
		p := obj.(*apiv1.Pod)
		got, err := client.CoreV1().Pods(p.Namespace).Get(ctx, p.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, []string{"example.com/keep"}, got.Finalizers)
	}
	assert.Zero(t, c.cleanupDiagnostics.recent.Len())
}

func TestStatusCaptureDiagnosticReasonChangeAndBound(t *testing.T) {
	hook := logging.NewTestHook()
	c := &Controller{log: logging.NewTestLogger(logging.Info, logging.Text, hook)}
	ctx := logging.WithLogger(t.Context(), c.log)
	c.reportCleanupHold(ctx, "pod", c.log, StatusCaptureHold(CaptureResultPending, "waiting"))
	c.reportCleanupHold(ctx, "pod", c.log, StatusCaptureHold(CaptureResultPending, "waiting"))
	c.reportCleanupHold(ctx, "pod", c.log, StatusCaptureHold(CaptureIdentityConflict, "different UID"))
	warnings := 0
	for _, entry := range hook.AllEntries() {
		if entry.Level == logging.Warn {
			warnings++
		}
	}
	require.Equal(t, 2, warnings)
	assert.Equal(t, CaptureIdentityConflict, hook.LastEntry().Fields["captureReason"])
	for i := range cleanupDiagnosticCapacity + 10 {
		c.cleanupDiagnostics.changed(fmt.Sprint(i), "reason")
	}
	assert.Equal(t, cleanupDiagnosticCapacity, c.cleanupDiagnostics.recent.Len())
	c.cleanupDiagnostics.forget("pod")
	assert.True(t, c.cleanupDiagnostics.changed("pod", "reason"), "lost diagnostics cannot discard work")
}
