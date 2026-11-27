package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
)

type recordingMemoizationDB struct {
	memodb.MemoizationDB
	load func(string, string, string) (*memodb.CacheRecord, error)
	save func(string, string, string, string, *wfv1.Outputs, string) error
}

func (*recordingMemoizationDB) IsEnabled() bool { return true }
func (q *recordingMemoizationDB) Load(_ context.Context, ns, name, key string) (*memodb.CacheRecord, error) {
	return q.load(ns, name, key)
}
func (q *recordingMemoizationDB) Save(_ context.Context, ns, name, key, nodeID string, outputs *wfv1.Outputs, maxAge string) error {
	return q.save(ns, name, key, nodeID, outputs, maxAge)
}

func TestSQLMemoizationOperate(t *testing.T) {
	for _, result := range []string{"hit", "miss", "error", "missing-key"} {
		t.Run(result, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx)
			defer cancel()
			wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-cached.yaml")
			wf.Namespace = "workflow-namespace"
			if result == "missing-key" {
				wf.Spec.Templates[0].Memoize.Key = ""
			}
			calls := 0
			controller.cacheFactory.SetQueries(ctx, &recordingMemoizationDB{load: func(ns, name, key string) (*memodb.CacheRecord, error) {
				calls++
				assert.Equal(t, "default", ns, "cache stays in the controller namespace")
				assert.Equal(t, "whalesay-cache", name)
				assert.Equal(t, "hi-there-world", key)
				switch result {
				case "hit":
					return &memodb.CacheRecord{NodeID: "cached-node", Outputs: `{"parameters":[{"name":"hello","value":"SQL output"}]}`, CreatedAt: time.Now()}, nil
				case "error":
					return nil, errors.New("SQL unavailable")
				default:
					return nil, nil
				}
			}})
			_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Create(ctx, wf, metav1.CreateOptions{})
			require.NoError(t, err)
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)
			if result == "missing-key" {
				assert.Zero(t, calls, "a missing key must not be generated automatically")
			} else {
				assert.Equal(t, 1, calls)
			}
			require.Len(t, woc.wf.Status.Nodes, 1)
			for _, node := range woc.wf.Status.Nodes {
				switch result {
				case "hit":
					assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
					require.NotNil(t, node.MemoizationStatus)
					assert.True(t, node.MemoizationStatus.Hit)
					require.NotNil(t, node.Outputs)
					assert.Equal(t, "SQL output", node.Outputs.Parameters[0].Value.String())
				case "miss":
					assert.Equal(t, wfv1.NodePending, node.Phase)
					assert.False(t, node.MemoizationStatus.Hit)
				case "error":
					assert.Equal(t, wfv1.NodeError, node.Phase)
					assert.Contains(t, node.Message, "SQL unavailable")
				case "missing-key":
					assert.Equal(t, wfv1.NodeError, node.Phase)
					assert.Contains(t, node.Message, "invalid cache key")
				}
			}
			for _, action := range controller.kubeclientset.(*fake.Clientset).Actions() {
				if action.Matches("get", "configmaps") {
					assert.NotEqual(t, "whalesay-cache", action.(k8stesting.GetAction).GetName(), "SQL memoization must not read the ConfigMap cache")
				}
				assert.False(t, action.Matches("create", "configmaps"), "SQL memoization must not create a ConfigMap cache")
			}
		})
	}
}

func TestSaveMemoizationCache(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-cached.yaml")
	wf.Spec.Templates[0].Memoize.MaxAge = "{{inputs.parameters.cacheAge}}"
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	node := &wfv1.NodeStatus{ID: "node", Outputs: &wfv1.Outputs{}, MemoizationStatus: &wfv1.MemoizationStatus{Key: "key", CacheName: "cache", MaxAge: "1h"}}
	saveErr := errors.New("save failed")
	controller.cacheFactory.SetQueries(ctx, &recordingMemoizationDB{save: func(ns, name, key, id string, outputs *wfv1.Outputs, maxAge string) error {
		assert.Equal(t, "default", ns)
		assert.Equal(t, "cache", name)
		assert.Equal(t, "key", key)
		assert.Equal(t, "node", id)
		assert.Same(t, node.Outputs, outputs)
		assert.Equal(t, "1h", maxAge)
		return saveErr
	}})
	require.ErrorIs(t, woc.saveMemoizationCache(ctx, node), saveErr)
	// The template may no longer be available; completion uses persisted status only.
	node.TemplateRef = &wfv1.TemplateRef{Name: "missing-template", Template: "main"}
	require.ErrorIs(t, woc.saveMemoizationCache(ctx, node), saveErr)
	require.NoError(t, woc.saveMemoizationCache(ctx, &wfv1.NodeStatus{}))
}

func TestSQLMemoizationTaskSetSave(t *testing.T) {
	for _, fails := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "error"}[fails], func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := wfv1.MustUnmarshalWorkflow("@testdata/taskset/reconcile-task-set-with-memoization-workflow.yaml")
			var taskSet wfv1.WorkflowTaskSet
			wfv1.MustUnmarshal("@testdata/taskset/reconcile-task-set-with-memoization-taskset.yaml", &taskSet)
			cancel, controller := newController(ctx, wf)
			defer cancel()
			require.NoError(t, controller.wfTaskSetInformer.Informer().GetIndexer().Add(&taskSet))
			calls := 0
			controller.cacheFactory.SetQueries(ctx, &recordingMemoizationDB{save: func(ns, name, key, id string, outputs *wfv1.Outputs, maxAge string) error {
				calls++
				assert.Equal(t, "default", ns)
				assert.Equal(t, "cache-demo-1", name)
				assert.Equal(t, "cache-demo-1", key)
				assert.Equal(t, "10s", maxAge)
				require.NotNil(t, outputs)
				if fails {
					return errors.New("save failed")
				}
				return nil
			}})
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			require.NoError(t, woc.reconcileTaskSet(ctx))
			assert.Equal(t, 1, calls)
			for _, node := range woc.wf.Status.Nodes {
				if fails {
					assert.Equal(t, wfv1.NodeError, node.Phase)
					assert.Equal(t, "save failed", node.Message)
				} else {
					assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
				}
			}
		})
	}
}

func TestMemoizationMissingCacheConfig(t *testing.T) {
	for _, cache := range []*wfv1.Cache{nil, {}} {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx)
		wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-cached.yaml")
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		tmpl := &wf.Spec.Templates[0]
		tmpl.Memoize.Cache = cache
		_, node, err := woc.handleMemoization(ctx, wf.Name, nil, tmpl, "local/"+wf.Name, &wfv1.WorkflowStep{Template: tmpl.Name}, "", nil, false)
		require.ErrorContains(t, err, "memoize.cache.configMap.name is required")
		require.NotNil(t, node)
		assert.Equal(t, wfv1.NodeError, node.Phase)
		cancel()
	}
}
