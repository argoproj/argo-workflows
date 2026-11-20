package cron

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/pkg/sync"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/metrics"
)

// tombstoneCronWf loads the CronWorkflow from the issue's reproduction.
func tombstoneCronWf(t *testing.T) *unstructured.Unstructured {
	t.Helper()
	var cronWf v1alpha1.CronWorkflow
	v1alpha1.MustUnmarshal("@testdata/tombstone-wf.yaml", &cronWf)
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&cronWf)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: obj}
}

// newTestCronController builds the parts of a Controller that the informer
// handler and processNextCronItem need. The CronWorkflow informer is backed
// by a fake dynamic client and is never started: the test drives its index
// and the event handler directly, the way client-go does after a relist.
func newTestCronController(ctx context.Context, t *testing.T) *Controller {
	t.Helper()
	testMetrics, err := metrics.New(ctx, telemetry.TestScopeName, telemetry.TestScopeName, &telemetry.MetricsConfig{}, metrics.Callbacks{})
	require.NoError(t, err)
	cc := &Controller{
		cron:        newCronFacade(),
		keyLock:     sync.NewKeyLock(),
		wfClientset: fake.NewClientset(),
		metrics:     testMetrics,
		cronWfQueue: workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]()),
		logger:      logging.RequireLoggerFromContext(ctx),
	}
	cc.cronWfInformer = dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		cronWorkflowResyncPeriod, cc.managedNamespace,
		func(options *v1.ListOptions) { cronWfInformerListOptionsFunc(options, cc.instanceID) },
	).ForResource(schema.GroupVersionResource{Group: workflow.Group, Version: workflow.Version, Resource: workflow.CronWorkflowPlural})
	return cc
}

// TestTombstoneDeleteRemovesScheduledJob covers #17442: a delete that arrives
// as a cache.DeletedFinalStateUnknown tombstone - the watch was down long
// enough that the API server answered 410 Gone and the relist had to deliver
// it - must still reach DeleteFunc, enqueue the key, and get the scheduled
// job removed. Before the fix the FilterFunc rejected the tombstone for not
// being an *unstructured, client-go's FilteringResourceEventHandler.OnDelete
// returned early, and the job kept submitting Workflows for the deleted
// CronWorkflow until the controller restarted.
func TestTombstoneDeleteRemovesScheduledJob(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cc := newTestCronController(ctx, t)
	require.NoError(t, cc.addCronWorkflowInformerHandler(ctx))

	// Schedule the job the way the informer would: the CronWorkflow is in
	// the index and its key is queued.
	un := tombstoneCronWf(t)
	key := un.GetNamespace() + "/" + un.GetName()
	require.NoError(t, cc.cronWfInformer.Informer().GetIndexer().Add(un))
	cc.cronWfQueue.Add(key)
	require.True(t, cc.processNextCronItem(ctx))
	require.Contains(t, cc.cron.entryIDs, key, "the CronWorkflow should be scheduled by now")

	// The watch gap ends: the object has left the index and client-go
	// delivers the delete as a tombstone.
	require.NoError(t, cc.cronWfInformer.Informer().GetIndexer().Delete(un))
	// the handler addCronWorkflowInformerHandler registered, built the same
	// way so the test can call it directly
	handler := cc.cronWorkflowEventHandler(ctx)
	handler.OnDelete(cache.DeletedFinalStateUnknown{Key: key, Obj: un})

	require.Equal(t, 1, cc.cronWfQueue.Len(), "the tombstone delete must be enqueued, or the scheduled job is never removed")

	require.True(t, cc.processNextCronItem(ctx))
	assert.NotContains(t, cc.cron.entryIDs, key, "the scheduled job must be removed when the CronWorkflow is gone")
}

// TestCronWorkflowFilterFunc pins what the informer filter lets through. The
// tombstone row is the fix for #17442; the others guard the behaviour around
// it, in particular that a completed CronWorkflow is still filtered out.
func TestCronWorkflowFilterFunc(t *testing.T) {
	live := tombstoneCronWf(t)
	completed := live.DeepCopy()
	completed.SetLabels(map[string]string{common.LabelKeyCronWorkflowCompleted: "true"})
	key := live.GetNamespace() + "/" + live.GetName()

	tests := []struct {
		name  string
		event func(handler cache.ResourceEventHandler)
		want  int
	}{
		{
			name:  "Add of a live CronWorkflow",
			event: func(h cache.ResourceEventHandler) { h.OnAdd(live, false) },
			want:  1,
		},
		{
			name:  "Add of a completed CronWorkflow",
			event: func(h cache.ResourceEventHandler) { h.OnAdd(completed, false) },
			want:  0,
		},
		{
			name:  "Delete of a live CronWorkflow",
			event: func(h cache.ResourceEventHandler) { h.OnDelete(live) },
			want:  1,
		},
		{
			name:  "Update of a live CronWorkflow",
			event: func(h cache.ResourceEventHandler) { h.OnUpdate(live, live) },
			want:  1,
		},
		{
			name:  "Delete of a completed CronWorkflow",
			event: func(h cache.ResourceEventHandler) { h.OnDelete(completed) },
			want:  0,
		},
		{
			name: "Delete delivered as a tombstone",
			event: func(h cache.ResourceEventHandler) {
				h.OnDelete(cache.DeletedFinalStateUnknown{Key: key, Obj: live})
			},
			want: 1,
		},
		{
			name:  "Delete of something that is not an object",
			event: func(h cache.ResourceEventHandler) { h.OnDelete("not an object") },
			want:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			cc := newTestCronController(ctx, t)
			handler := cc.cronWorkflowEventHandler(ctx)
			tt.event(handler)
			assert.Equal(t, tt.want, cc.cronWfQueue.Len())
		})
	}
}

// TestRunSkipsSubmissionAfterCronWorkflowDeleted is the backstop from
// #17442: a job firing after its CronWorkflow left the informer - the window
// between a tombstone delete arriving and the queued key being processed -
// must not submit a Workflow. With the CronWorkflow still in the index the
// run goes ahead as before.
func TestRunSkipsSubmissionAfterCronWorkflowDeleted(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	var cronWf v1alpha1.CronWorkflow
	v1alpha1.MustUnmarshal("@testdata/tombstone-wf.yaml", &cronWf)
	un := tombstoneCronWf(t)

	newWoc := func(t *testing.T, exists bool) (*cronWfOperationCtx, *fake.Clientset) {
		t.Helper()
		cs := fake.NewClientset()
		testMetrics, err := metrics.New(ctx, telemetry.TestScopeName, telemetry.TestScopeName, &telemetry.MetricsConfig{}, metrics.Callbacks{})
		require.NoError(t, err)
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
		if exists {
			require.NoError(t, indexer.Add(un))
		}
		woc := &cronWfOperationCtx{
			wfClientset:   cs,
			wfClient:      cs.ArgoprojV1alpha1().Workflows(cronWf.Namespace),
			cronWfIf:      cs.ArgoprojV1alpha1().CronWorkflows(cronWf.Namespace),
			cronWf:        cronWf.DeepCopy(),
			log:           logging.RequireLoggerFromContext(ctx),
			metrics:       testMetrics,
			cronWfIndexer: indexer,
		}
		return woc, cs
	}

	t.Run("Skipped when the CronWorkflow is gone", func(t *testing.T) {
		woc, cs := newWoc(t, false)
		woc.run(ctx, time.Now())
		assert.Empty(t, cs.Actions(), "no submit and no status patch for a deleted CronWorkflow")
	})

	t.Run("Runs when the CronWorkflow is present", func(t *testing.T) {
		woc, cs := newWoc(t, true)
		woc.run(ctx, time.Now())
		wsl, err := cs.ArgoprojV1alpha1().Workflows(cronWf.Namespace).List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.Len(t, wsl.Items, 1, "a live CronWorkflow still submits its Workflow")
	})

	t.Run("Runs when the index cannot be consulted", func(t *testing.T) {
		woc, cs := newWoc(t, false)
		woc.cronWfIndexer = erroringIndexer{Indexer: woc.cronWfIndexer}
		woc.run(ctx, time.Now())
		wsl, err := cs.ArgoprojV1alpha1().Workflows(cronWf.Namespace).List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.Len(t, wsl.Items, 1, "an indexer error must not suppress a run")
	})
}

// erroringIndexer fails GetByKey, standing in for an index that cannot be
// consulted; the embedded Indexer satisfies the rest of the interface.
type erroringIndexer struct {
	cache.Indexer
}

func (erroringIndexer) GetByKey(string) (any, bool, error) {
	return nil, false, errors.New("index unavailable")
}
