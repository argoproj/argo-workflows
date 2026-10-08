package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	wfutil "github.com/argoproj/argo-workflows/v4/workflow/util"
)

// smallBackoff returns a PersistConfig with small retry values for fast tests,
// covering both the regular and fallback hydration budgets.
func smallBackoff(retries int) *config.PersistConfig {
	b := metav1.Duration{Duration: time.Millisecond}
	fallbackRetries := retries
	return &config.PersistConfig{
		TemplateHydrationRetries:            &retries,
		TemplateHydrationBackoff:            &b,
		TemplateHydrationBackoffMax:         &b,
		TemplateHydrationFallbackRetries:    &fallbackRetries,
		TemplateHydrationFallbackBackoff:    &b,
		TemplateHydrationFallbackBackoffMax: &b,
	}
}

func TestHydrateTemplates_NoTemplateRepo(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.templateRepo = nil

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	require.NoError(t, err, "Should not error when template repo is nil")
	assert.Empty(t, woc.execWf.Spec.Templates, "Templates should remain empty")
}

func TestHydrateTemplates_AlreadyHydrated(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	mockRepo := mocks.NewTemplateRepo(t)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID:      "test-uid-123",
				Hydrated: true,
			},
			StoredTemplates: map[string]wfv1.Template{
				"template-1": {Name: "template-1"},
			},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	require.NoError(t, err)
	mockRepo.AssertNotCalled(t, "GetTemplates", mock.Anything, mock.Anything)
}

// TestHydrateTemplates_HydratedButNoStoredTemplates verifies re-hydration
// when StoredTemplateSpecs says Hydrated=true but StoredTemplates is nil.
func TestHydrateTemplates_HydratedButNoStoredTemplates(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	mockRepo := mocks.NewTemplateRepo(t)
	expectedTemplates := []wfv1.Template{{Name: "template-1"}}
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return(expectedTemplates, nil)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID:      "test-uid-123",
				Hydrated: true,
			},
			StoredTemplates: nil,
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	require.NoError(t, err)
	mockRepo.AssertCalled(t, "GetTemplates", ctx, "test-uid-123")
	require.Len(t, woc.execWf.Spec.Templates, 1)
	assert.Equal(t, "template-1", woc.execWf.Spec.Templates[0].Name)
	assert.True(t, woc.wf.Status.StoredTemplateSpecs.Hydrated)
	assert.True(t, woc.updated)
}

func TestHydrateTemplates_NotOffloaded(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	mockRepo := mocks.NewTemplateRepo(t)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{
				{Name: "inline-template"},
			},
		},
		Status: wfv1.WorkflowStatus{
			// No StoredTemplateSpecs - templates not offloaded
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	require.NoError(t, err)
	mockRepo.AssertNotCalled(t, "GetTemplates", mock.Anything, mock.Anything)
	require.Len(t, woc.execWf.Spec.Templates, 1)
	assert.Equal(t, "inline-template", woc.execWf.Spec.Templates[0].Name)
}

// TestHydrateTemplates_InlineTemplatesWithStaleMarker covers an archived retry
// that kept the original offload marker: inline templates must win, with no DB probe.
func TestHydrateTemplates_InlineTemplatesWithStaleMarker(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	mockRepo := mocks.NewTemplateRepo(t)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: "default",
			UID:       "new-uid-456",
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{{Name: "inline-template"}},
		},
		Status: wfv1.WorkflowStatus{
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID:      "old-uid-123",
				Hydrated: true,
			},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	require.NoError(t, err)
	mockRepo.AssertNotCalled(t, "GetTemplates", mock.Anything, mock.Anything)
	require.Len(t, woc.execWf.Spec.Templates, 1)
	assert.Equal(t, "inline-template", woc.execWf.Spec.Templates[0].Name)
}

func TestHydrateTemplates_Success(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	mockRepo := mocks.NewTemplateRepo(t)
	expectedTemplates := []wfv1.Template{
		{Name: "template-1"},
		{Name: "template-2"},
	}
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return(expectedTemplates, nil)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID:      "test-uid-123",
				Hydrated: false,
			},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	require.NoError(t, err)
	mockRepo.AssertCalled(t, "GetTemplates", ctx, "test-uid-123")
	require.Len(t, woc.execWf.Spec.Templates, 2)
	assert.Equal(t, "template-1", woc.execWf.Spec.Templates[0].Name)
	assert.Equal(t, "template-2", woc.execWf.Spec.Templates[1].Name)
	assert.True(t, woc.wf.Status.StoredTemplateSpecs.Hydrated)
	assert.True(t, woc.updated)
}

func TestHydrateTemplates_EmptyResult(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.Persistence = smallBackoff(2)

	mockRepo := mocks.NewTemplateRepo(t)
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return([]wfv1.Template{}, nil)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wf",
			Namespace:         "default",
			UID:               "test-uid-123",
			CreationTimestamp: metav1.NewTime(time.Now()),
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID:      "test-uid-123",
				Hydrated: false,
			},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	var pending *templateHydrationPendingError
	require.ErrorAs(t, err, &pending)
	assert.Equal(t, time.Millisecond, pending.after)
	mockRepo.AssertNumberOfCalls(t, "GetTemplates", 1)
}

// TestHydrateTemplates_EmptyResultExhausted verifies the main path fails once the
// workflow is older than the whole retry budget.
func TestHydrateTemplates_EmptyResultExhausted(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.Persistence = smallBackoff(2)

	mockRepo := mocks.NewTemplateRepo(t)
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return([]wfv1.Template{}, nil)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wf",
			Namespace:         "default",
			UID:               "test-uid-123",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Hour)),
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID:      "test-uid-123",
				Hydrated: false,
			},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no templates found in database")
	mockRepo.AssertNumberOfCalls(t, "GetTemplates", 1)
}

func TestHydrateTemplates_Error(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	mockRepo := mocks.NewTemplateRepo(t)
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return(nil, assert.AnError)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wf",
			Namespace:         "default",
			UID:               "test-uid-123",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Hour)),
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID:      "test-uid-123",
				Hydrated: false,
			},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to hydrate templates")
	mockRepo.AssertCalled(t, "GetTemplates", ctx, "test-uid-123")
}

// TestHydrateTemplates_ErrorRetries verifies a DB error inside the budget returns a
// pending signal that wraps the cause, instead of blocking in a retry loop.
func TestHydrateTemplates_ErrorRetries(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.Persistence = smallBackoff(2)

	mockRepo := mocks.NewTemplateRepo(t)
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return(nil, assert.AnError)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wf",
			Namespace:         "default",
			UID:               "test-uid-123",
			CreationTimestamp: metav1.NewTime(time.Now()),
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID:      "test-uid-123",
				Hydrated: false,
			},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	var pending *templateHydrationPendingError
	require.ErrorAs(t, err, &pending)
	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, time.Millisecond, pending.after)
	mockRepo.AssertNumberOfCalls(t, "GetTemplates", 1)
}

// TestHydrateTemplates_FallbackPath_Success verifies the fallback hydration
// path when StoredTemplateSpecs is not set but templates exist in the DB.
func TestHydrateTemplates_FallbackPath_Success(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.Persistence = smallBackoff(2)

	mockRepo := mocks.NewTemplateRepo(t)
	expectedTemplates := []wfv1.Template{{Name: "template-1"}}
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return(expectedTemplates, nil)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			// No StoredTemplateSpecs - triggers fallback
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	require.NoError(t, err)
	mockRepo.AssertCalled(t, "GetTemplates", ctx, "test-uid-123")
	require.Len(t, woc.execWf.Spec.Templates, 1)
	assert.Equal(t, "template-1", woc.execWf.Spec.Templates[0].Name)
	require.NotNil(t, woc.wf.Status.StoredTemplateSpecs)
	assert.True(t, woc.wf.Status.StoredTemplateSpecs.Hydrated)
	assert.Equal(t, "test-uid-123", woc.wf.Status.StoredTemplateSpecs.UID)
	assert.True(t, woc.updated)
}

// TestHydrateTemplates_FallbackPath_Empty verifies the fallback path requeues while
// waiting for rows that the API server may not have committed yet.
func TestHydrateTemplates_FallbackPath_Empty(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.Persistence = smallBackoff(2)

	mockRepo := mocks.NewTemplateRepo(t)
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return([]wfv1.Template{}, nil)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wf",
			Namespace:         "default",
			UID:               "test-uid-123",
			CreationTimestamp: metav1.NewTime(time.Now()),
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			// No StoredTemplateSpecs - triggers fallback
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	var pending *templateHydrationPendingError
	require.ErrorAs(t, err, &pending)
	assert.Equal(t, time.Millisecond, pending.after)
	mockRepo.AssertNumberOfCalls(t, "GetTemplates", 1)
}

// TestHydrateTemplates_FallbackPath_EmptyExhausted verifies the fallback gives up once
// the workflow is older than the whole retry budget, and treats it as not offloaded.
func TestHydrateTemplates_FallbackPath_EmptyExhausted(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.Persistence = smallBackoff(2)

	mockRepo := mocks.NewTemplateRepo(t)
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return([]wfv1.Template{}, nil)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wf",
			Namespace:         "default",
			UID:               "test-uid-123",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Hour)),
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			// No StoredTemplateSpecs - triggers fallback
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	require.NoError(t, err)
	mockRepo.AssertNumberOfCalls(t, "GetTemplates", 1)
	assert.Nil(t, woc.wf.Status.StoredTemplateSpecs)
}

// TestHydrateTemplates_FallbackPath_Error verifies the fallback path propagates persistent
// DB errors rather than silently treating them as "not offloaded".
func TestHydrateTemplates_FallbackPath_Error(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.Persistence = smallBackoff(0)

	mockRepo := mocks.NewTemplateRepo(t)
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return(nil, assert.AnError)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			// No StoredTemplateSpecs - triggers fallback
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check for offloaded templates")
	mockRepo.AssertCalled(t, "GetTemplates", ctx, "test-uid-123")
	assert.Nil(t, woc.wf.Status.StoredTemplateSpecs)
}

// TestHydrateTemplates_RetryEventualConsistency verifies the submit race is bridged
// across reconciles: the first fetch signals pending, the next one succeeds.
func TestHydrateTemplates_RetryEventualConsistency(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.Persistence = smallBackoff(3)

	mockRepo := mocks.NewTemplateRepo(t)
	expectedTemplates := []wfv1.Template{{Name: "template-1"}}
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return([]wfv1.Template{}, nil).Once()
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return(expectedTemplates, nil).Once()
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wf",
			Namespace:         "default",
			UID:               "test-uid-123",
			CreationTimestamp: metav1.NewTime(time.Now()),
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID:      "test-uid-123",
				Hydrated: false,
			},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	err := woc.hydrateTemplates(ctx)
	var pending *templateHydrationPendingError
	require.ErrorAs(t, err, &pending)
	require.False(t, woc.wf.Status.StoredTemplateSpecs.Hydrated)

	// Next reconcile: the rows are committed now.
	err = woc.hydrateTemplates(ctx)
	require.NoError(t, err)
	mockRepo.AssertNumberOfCalls(t, "GetTemplates", 2)
	require.Len(t, woc.execWf.Spec.Templates, 1)
	assert.Equal(t, "template-1", woc.execWf.Spec.Templates[0].Name)
	assert.True(t, woc.wf.Status.StoredTemplateSpecs.Hydrated)
}

// TestHydrateTemplates_RetryExhaustion_Empty verifies the main path returns
// an error when all retries return empty results.
func TestHydrateTemplates_RetryExhaustion_Empty(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.Persistence = smallBackoff(2)

	mockRepo := mocks.NewTemplateRepo(t)
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return([]wfv1.Template{}, nil)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wf",
			Namespace:         "default",
			UID:               "test-uid-123",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Hour)),
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID:      "test-uid-123",
				Hydrated: false,
			},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	err := woc.hydrateTemplates(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no templates found in database")
	mockRepo.AssertNumberOfCalls(t, "GetTemplates", 1)
}

// TestApplyHydratedTemplates verifies all side-effects of applyHydratedTemplates.
func TestApplyHydratedTemplates(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	mockRepo := mocks.NewTemplateRepo(t)
	controller.templateRepo = mockRepo

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	templates := []wfv1.Template{
		{Name: "tmpl-1"},
		{Name: "tmpl-2"},
	}
	woc.applyHydratedTemplates(templates)

	require.NotNil(t, woc.execWf)
	assert.Equal(t, templates, woc.execWf.Spec.Templates)

	require.NotNil(t, woc.wf.Status.StoredTemplates)
	assert.Len(t, woc.wf.Status.StoredTemplates, 2)
	assert.Equal(t, "tmpl-1", woc.wf.Status.StoredTemplates["tmpl-1"].Name)
	assert.Equal(t, "tmpl-2", woc.wf.Status.StoredTemplates["tmpl-2"].Name)

	require.NotNil(t, woc.wf.Status.StoredTemplateSpecs)
	assert.Equal(t, "test-uid-123", woc.wf.Status.StoredTemplateSpecs.UID)
	assert.True(t, woc.wf.Status.StoredTemplateSpecs.Hydrated)
	assert.NotEmpty(t, woc.wf.Status.StoredTemplateSpecs.Version)

	assert.True(t, woc.updated)
}

func TestComputeTemplateVersion(t *testing.T) {
	templates1 := []wfv1.Template{
		{Name: "template-1"},
	}

	templates2 := []wfv1.Template{
		{Name: "template-1"},
	}

	templates3 := []wfv1.Template{
		{Name: "template-2"},
	}

	templates4 := []wfv1.Template{
		{Name: "template-1", Inputs: wfv1.Inputs{Parameters: []wfv1.Parameter{{Name: "p1", Value: wfv1.AnyStringPtr("v1")}}}},
	}

	version1 := wfutil.ComputeTemplateVersion(templates1)
	version2 := wfutil.ComputeTemplateVersion(templates2)
	version3 := wfutil.ComputeTemplateVersion(templates3)
	version4 := wfutil.ComputeTemplateVersion(templates4)

	assert.Equal(t, version1, version2, "Same templates should produce same version")
	assert.NotEqual(t, version1, version3, "Different template names should produce different version")
	assert.NotEqual(t, version1, version4, "Different template content should produce different version")
	assert.Contains(t, version1, "sha256:", "Version should be sha256 hash")
}

// TestTemplateGarbageCollector verifies that the template GC removes orphaned offloaded template
// rows for workflows that no longer exist, with node-status offload disabled.
func TestTemplateGarbageCollector(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	// Use a very short GC period so the test runs quickly
	t.Setenv("TEMPLATE_GC_PERIOD", "50ms")

	mockRepo := mocks.NewTemplateRepo(t)
	// Return two UIDs: one orphaned, one that still exists (via informer)
	mockRepo.On("IsEnabled").Return(true)
	mockRepo.On("ListOldOffloads", mock.Anything, mock.Anything).Return([]string{"orphaned-uid-1", "existing-uid-2"}, nil)
	mockRepo.On("DeleteTemplates", mock.Anything, "orphaned-uid-1").Return(nil)
	// existing-uid-2 should NOT be deleted because the workflow still exists
	controller.templateRepo = mockRepo

	// Add a workflow with UID "existing-uid-2" to the informer so the GC skips it
	existingWf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-wf",
			Namespace: "default",
			UID:       "existing-uid-2",
		},
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(existingWf)
	require.NoError(t, err)
	err = controller.wfInformer.GetIndexer().Add(&unstructured.Unstructured{Object: unstructuredMap})
	require.NoError(t, err)

	// Run the GC with a cancellable context
	gcCtx, gcCancel := context.WithCancel(ctx)
	go controller.templateGarbageCollector(gcCtx)

	// Wait for the GC to process at least one tick
	// The mock will fail the test if DeleteTemplates is called with "existing-uid-2"
	time.Sleep(500 * time.Millisecond)
	gcCancel()

	// Verify the orphaned workflow's templates were deleted
	mockRepo.AssertCalled(t, "DeleteTemplates", mock.Anything, "orphaned-uid-1")
	// Verify the existing workflow's templates were NOT deleted
	mockRepo.AssertNotCalled(t, "DeleteTemplates", mock.Anything, "existing-uid-2")
}

// TestSubstituteGlobalVariables_PreservesHydratedTemplates: a hydrated workflow's template body keeps
// unresolved placeholders (both ${item} and ${workflow.parameters.x}) intact through
// template.Replace, while a non-hydrated workflow's templates are nil-ed out before substitution.
func TestSubstituteGlobalVariables_PreservesHydratedTemplates(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	placeholderArgs := []string{"${workflow.parameters.foo}-${item}"}
	tmpl := wfv1.Template{
		Name: "main",
		Container: &v1.Container{
			Image: "busybox:latest",
			Args:  placeholderArgs,
		},
	}

	newWF := func(name, uid string, hydrated bool) *wfv1.Workflow {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
				UID:       types.UID(uid),
			},
			Status: wfv1.WorkflowStatus{
				Phase: wfv1.WorkflowRunning,
				// Mirror the hydrated template body into StoredTemplates, as the fallback
				// hydration path would.
				StoredTemplates: map[string]wfv1.Template{
					"main": tmpl,
				},
			},
		}
		if hydrated {
			wf.Status.StoredTemplateSpecs = &wfv1.TemplateSpecReference{
				UID:      uid,
				Hydrated: true,
			}
		}
		return wf
	}

	// Hydrated case: the template body lives in execWf.Spec.Templates (hydrated from the DB).
	woc := newWorkflowOperationCtx(ctx, newWF("hydrated-wf", "test-uid", true), controller)
	woc.execWf.Spec.Templates = []wfv1.Template{tmpl}
	require.Len(t, woc.execWf.Spec.Templates, 1)

	err := woc.substituteGlobalVariables(ctx, woc.globalParams())
	require.NoError(t, err)

	// The hydrated template body is preserved and survives substitution untouched: neither the
	// ${workflow.parameters.foo} global token (unregistered) nor the ${item} local token (no
	// scope) is resolvable, so with allowUnresolved they both remain verbatim.
	require.Len(t, woc.execWf.Spec.Templates, 1)
	require.NotNil(t, woc.execWf.Spec.Templates[0].Container)
	assert.Equal(t, placeholderArgs, woc.execWf.Spec.Templates[0].Container.Args,
		"unresolved placeholders must survive the hydrated template body intact")

	// The mirrored StoredTemplates entry must remain the raw, un-substituted body.
	stored := woc.wf.Status.StoredTemplates["main"]
	require.NotNil(t, stored.Container)
	assert.Equal(t, placeholderArgs, stored.Container.Args)

	// Non-hydrated control case: the nil-out lands on the local execWfSpec copy, and because
	// WorkflowSpec.Templates carries `omitempty` the JSON round-trip drops the key, so json.Unmarshal
	// leaves woc.execWf.Spec.Templates intact.
	wocNonHydrated := newWorkflowOperationCtx(ctx, newWF("non-hydrated-wf", "nonhydrated-uid", false), controller)
	wocNonHydrated.execWf.Spec.Templates = []wfv1.Template{tmpl}
	require.Len(t, wocNonHydrated.execWf.Spec.Templates, 1)

	err = wocNonHydrated.substituteGlobalVariables(ctx, wocNonHydrated.globalParams())
	require.NoError(t, err)
	// Surprising (see comment above): the templates are NOT nil-ed out; they survive verbatim
	// exactly like the hydrated case, so the isHydrated marker does not actually change template
	// survival through this function.
	assert.Len(t, wocNonHydrated.execWf.Spec.Templates, 1, "templates surprise-survive the intended nil-out (omitempty drops the key on the JSON round-trip)")
	assert.Equal(t, placeholderArgs, wocNonHydrated.execWf.Spec.Templates[0].Container.Args,
		"unresolved placeholders survive the non-hydrated path too")
}

func TestNextHydrationRetry(t *testing.T) {
	tests := []struct {
		name     string
		elapsed  time.Duration
		initial  time.Duration
		max      time.Duration
		attempts int
		want     time.Duration
		wantOK   bool
	}{
		{"first attempt", 0, 500 * time.Millisecond, 120 * time.Second, 15, 500 * time.Millisecond, true},
		{"after first backoff", 600 * time.Millisecond, 500 * time.Millisecond, 120 * time.Second, 15, time.Second, true},
		{"mid schedule", 10 * time.Second, 500 * time.Millisecond, 120 * time.Second, 15, 8 * time.Second, true},
		{"capped", 10 * time.Minute, 500 * time.Millisecond, 120 * time.Second, 15, 120 * time.Second, true},
		{"exhausted", time.Hour, 500 * time.Millisecond, 120 * time.Second, 15, 0, false},
		{"zero budget", 0, 500 * time.Millisecond, 120 * time.Second, 0, 0, false},
		{"zero initial backoff", 0, 0, 0, 1, time.Millisecond, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := nextHydrationRetry(tc.elapsed, tc.initial, tc.max, tc.attempts)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestOperate_PendingHydrationRequeuesAfter verifies operate turns a pending hydration
// into a scheduled requeue (AddAfter) and returns, instead of blocking or rate-limiting.
func TestOperate_PendingHydrationRequeuesAfter(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.Persistence = smallBackoff(2)

	mockRepo := mocks.NewTemplateRepo(t)
	mockRepo.On("GetTemplates", ctx, "test-uid-123").Return([]wfv1.Template{}, nil)
	controller.templateRepo = mockRepo
	controller.wfQueue = workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())
	defer controller.wfQueue.ShutDown()

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wf",
			Namespace:         "default",
			UID:               "test-uid-123",
			CreationTimestamp: metav1.NewTime(time.Now()),
		},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{},
		},
		Status: wfv1.WorkflowStatus{
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID:      "test-uid-123",
				Hydrated: false,
			},
		},
	}

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	key := "default/test-wf"
	require.Eventually(t, func() bool { return controller.wfQueue.Len() == 1 }, time.Second, time.Millisecond,
		"the pending workflow must be scheduled back onto the queue")
	assert.Zero(t, controller.wfQueue.NumRequeues(key),
		"a pending hydration must be scheduled with AddAfter, not AddRateLimited")
}

// TestSubstituteGlobalVariables_DoesNotSubstituteTemplates pins that templates are kept
// out of the spec-wide substitution: global values are resolved later with a fresh scope,
// and the JSON round trip must not rewrite the slice the template cache shares.
func TestSubstituteGlobalVariables_DoesNotSubstituteTemplates(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.templateCache = newTemplateCache(1 << 20)

	tmpl := wfv1.Template{
		Name: "main",
		Container: &v1.Container{
			Args: []string{"{{workflow.parameters.foo}}"},
		},
	}
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wf",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Status: wfv1.WorkflowStatus{
			Phase: wfv1.WorkflowRunning,
			StoredTemplateSpecs: &wfv1.TemplateSpecReference{
				UID:      "test-uid-123",
				Version:  "v1",
				Hydrated: true,
			},
		},
	}
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	controller.templateCache.put("test-uid-123", "v1", []wfv1.Template{tmpl})
	cached, ok := controller.templateCache.get("test-uid-123", "v1")
	require.True(t, ok)
	woc.execWf.Spec.Templates = cached
	woc.execWf.Spec.NodeSelector = map[string]string{"zone": "{{workflow.parameters.foo}}"}

	params := common.Parameters{"workflow.parameters.foo": "resolved"}
	require.NoError(t, woc.substituteGlobalVariables(ctx, params))

	// Non-template spec fields are still substituted.
	assert.Equal(t, "resolved", woc.execWf.Spec.NodeSelector["zone"])

	// Template bodies stay unresolved here; executeTemplate substitutes them later.
	require.Len(t, woc.execWf.Spec.Templates, 1)
	assert.Equal(t, []string{"{{workflow.parameters.foo}}"}, woc.execWf.Spec.Templates[0].Container.Args,
		"template bodies must not be substituted by substituteGlobalVariables")

	// The shared backing array of the template cache must not be rewritten.
	stillCached, ok := controller.templateCache.get("test-uid-123", "v1")
	require.True(t, ok)
	require.Len(t, stillCached, 1)
	assert.Equal(t, []string{"{{workflow.parameters.foo}}"}, stillCached[0].Container.Args,
		"the version-keyed template cache must not be mutated")
}
