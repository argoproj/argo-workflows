package controller

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// TestHydrateTemplates_RehydrationCycleDoesNotDirtyWorkflow: across two reconcile cycles the second
// must not mark the workflow updated (no etcd write), and the version-keyed cache must skip the
// second database fetch.
func TestHydrateTemplates_RehydrationCycleDoesNotDirtyWorkflow(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	mockRepo := mocks.NewTemplateRepo(t)
	controller.templateRepo = mockRepo
	controller.templateCache = newTemplateCache(1 << 20)

	expected := []wfv1.Template{{Name: "template-1"}, {Name: "template-2"}}
	// Exactly ONE database fetch across both cycles: cycle 2 must be served
	// from the version-keyed cache (mockery fails the test on a second call).
	mockRepo.EXPECT().GetTemplates(mock.Anything, mock.Anything).Return(expected, nil).Once()

	// ---- cycle 1: informer object = server-created stub -----------------------
	// marker exists with Hydrated=false (server sets it post-Create), no rows
	// hydrated yet, no inline templates.
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-wf", Namespace: "default", UID: "test-uid-123",
		},
		Spec: wfv1.WorkflowSpec{Templates: []wfv1.Template{}},
		Status: wfv1.WorkflowStatus{
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID: "test-uid-123", Hydrated: false,
			},
		},
	}
	woc1 := newWorkflowOperationCtx(ctx, wf, controller)
	require.NoError(t, woc1.hydrateTemplates(ctx))
	require.True(t, woc1.updated, "first hydration (nil/false->true transition) must mark the workflow updated")
	require.NotEmpty(t, woc1.execWf.Spec.Templates)
	require.True(t, woc1.wf.Status.StoredTemplateSpecs.Hydrated)
	require.NotEmpty(t, woc1.wf.Status.StoredTemplateSpecs.Version)

	// ---- simulate persistUpdates' strip-then-restore --------------------------
	// The etcd object after cycle 1 carries stripped spec.templates + nil
	// StoredTemplates, with the hydrated marker surviving.
	stripped := woc1.wf.DeepCopy()
	stripped.Spec.Templates = nil
	stripped.Status.StoredTemplates = nil

	// ---- cycle 2: same informer object, nothing real changed ------------------
	woc2 := newWorkflowOperationCtx(ctx, stripped, controller)
	require.NoError(t, woc2.hydrateTemplates(ctx))
	require.False(t, woc2.updated,
		"steady-state re-hydration must not dirty the workflow (no etcd write, no self-sustaining requeue)")
	require.NotEmpty(t, woc2.execWf.Spec.Templates, "templates must still hydrate for execution")
	require.Len(t, woc2.execWf.Spec.Templates, 2)
	require.NotNil(t, woc2.wf.Status.StoredTemplateSpecs)
	require.Equal(t, woc1.wf.Status.StoredTemplateSpecs.Version, woc2.wf.Status.StoredTemplateSpecs.Version,
		"re-hydrated content must hash to the same version")
}

// TestTemplateCache_VersionKeyedLookup covers the cache contract directly:
// hit on matching version, miss on version change, FIFO byte-budget eviction,
// and nil-receiver safety.
func TestTemplateCache_VersionKeyedLookup(t *testing.T) {
	c := newTemplateCache(64) // tiny budget to exercise eviction
	templates := []wfv1.Template{{Name: "t1"}}

	c.put("uid-a", "sha256:aaa", templates)
	got, ok := c.get("uid-a", "sha256:aaa")
	require.True(t, ok)
	require.Len(t, got, 1)

	// version change -> miss (e.g. retry re-offload changed the rows)
	_, ok = c.get("uid-a", "sha256:bbb")
	require.False(t, ok)

	// unknown uid -> miss
	_, ok = c.get("uid-x", "sha256:aaa")
	require.False(t, ok)

	// eviction: oversize a single entry beyond the budget -> previous entries evicted
	big := make([]wfv1.Template, 64)
	for i := range big {
		big[i] = wfv1.Template{Name: "big"}
	}
	c.put("uid-big", "sha256:big", big)
	require.LessOrEqual(t, c.size(), int64(64+1<<20), "cache must respect its byte budget (with slack for the newest entry)")

	// nil receiver is a no-op (controllers constructed without the config path)
	var nilCache *templateCache
	nilCache.put("u", "v", templates)
	_, ok = nilCache.get("u", "v")
	require.False(t, ok)
	nilCache.evict("u")
	require.Zero(t, nilCache.size())
}
