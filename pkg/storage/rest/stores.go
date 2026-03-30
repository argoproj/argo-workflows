package rest

import (
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	watchutil "github.com/argoproj/argo-workflows/v4/pkg/storage/watch"
)

var gv = schema.GroupVersion{Group: workflow.Group, Version: workflow.Version}

func newStoreConfig(db *gorm.DB, wm *watchutil.Manager, scheme *runtime.Scheme, resource, kind string, namespaced bool, newFunc func() runtime.Object, newListFunc func() runtime.Object) StoreConfig {
	return StoreConfig{
		DB:           db,
		WatchManager: wm,
		Scheme:       scheme,
		GVR:          gv.WithResource(resource),
		GVK:          gv.WithKind(kind),
		ListGVK:      gv.WithKind(kind + "List"),
		NewFunc:      newFunc,
		NewListFunc:  newListFunc,
		Kind:         kind,
		Namespaced:   namespaced,
	}
}

func newStatusStoreConfig(db *gorm.DB, wm *watchutil.Manager, scheme *runtime.Scheme, resource, kind string, namespaced bool, newFunc func() runtime.Object) StatusStoreConfig {
	return StatusStoreConfig{
		DB:           db,
		WatchManager: wm,
		Scheme:       scheme,
		GVR:          gv.WithResource(resource),
		Kind:         kind,
		Namespaced:   namespaced,
		NewFunc:      newFunc,
	}
}

// NewWorkflowStorage returns storage for workflows and workflows/status.
func NewWorkflowStorage(db *gorm.DB, wm *watchutil.Manager, scheme *runtime.Scheme) map[string]rest.Storage {
	cfg := newStoreConfig(db, wm, scheme, workflow.WorkflowPlural, workflow.WorkflowKind, true,
		func() runtime.Object { return &wfv1.Workflow{} },
		func() runtime.Object { return &wfv1.WorkflowList{} },
	)
	statusCfg := newStatusStoreConfig(db, wm, scheme, workflow.WorkflowPlural, workflow.WorkflowKind, true,
		func() runtime.Object { return &wfv1.Workflow{} },
	)
	return map[string]rest.Storage{
		"workflows":        NewGenericStore(cfg),
		"workflows/status": NewStatusStore(statusCfg),
	}
}

// NewWorkflowTemplateStorage returns storage for workflowtemplates.
func NewWorkflowTemplateStorage(db *gorm.DB, wm *watchutil.Manager, scheme *runtime.Scheme) map[string]rest.Storage {
	cfg := newStoreConfig(db, wm, scheme, workflow.WorkflowTemplatePlural, workflow.WorkflowTemplateKind, true,
		func() runtime.Object { return &wfv1.WorkflowTemplate{} },
		func() runtime.Object { return &wfv1.WorkflowTemplateList{} },
	)
	return map[string]rest.Storage{
		"workflowtemplates": NewGenericStore(cfg),
	}
}

// NewClusterWorkflowTemplateStorage returns storage for clusterworkflowtemplates (cluster-scoped).
func NewClusterWorkflowTemplateStorage(db *gorm.DB, wm *watchutil.Manager, scheme *runtime.Scheme) map[string]rest.Storage {
	cfg := newStoreConfig(db, wm, scheme, workflow.ClusterWorkflowTemplatePlural, workflow.ClusterWorkflowTemplateKind, false,
		func() runtime.Object { return &wfv1.ClusterWorkflowTemplate{} },
		func() runtime.Object { return &wfv1.ClusterWorkflowTemplateList{} },
	)
	return map[string]rest.Storage{
		"clusterworkflowtemplates": NewGenericStore(cfg),
	}
}

// NewCronWorkflowStorage returns storage for cronworkflows and cronworkflows/status.
func NewCronWorkflowStorage(db *gorm.DB, wm *watchutil.Manager, scheme *runtime.Scheme) map[string]rest.Storage {
	cfg := newStoreConfig(db, wm, scheme, workflow.CronWorkflowPlural, workflow.CronWorkflowKind, true,
		func() runtime.Object { return &wfv1.CronWorkflow{} },
		func() runtime.Object { return &wfv1.CronWorkflowList{} },
	)
	statusCfg := newStatusStoreConfig(db, wm, scheme, workflow.CronWorkflowPlural, workflow.CronWorkflowKind, true,
		func() runtime.Object { return &wfv1.CronWorkflow{} },
	)
	return map[string]rest.Storage{
		"cronworkflows":        NewGenericStore(cfg),
		"cronworkflows/status": NewStatusStore(statusCfg),
	}
}

// NewWorkflowTaskSetStorage returns storage for workflowtasksets and workflowtasksets/status.
func NewWorkflowTaskSetStorage(db *gorm.DB, wm *watchutil.Manager, scheme *runtime.Scheme) map[string]rest.Storage {
	cfg := newStoreConfig(db, wm, scheme, workflow.WorkflowTaskSetPlural, workflow.WorkflowTaskSetKind, true,
		func() runtime.Object { return &wfv1.WorkflowTaskSet{} },
		func() runtime.Object { return &wfv1.WorkflowTaskSetList{} },
	)
	statusCfg := newStatusStoreConfig(db, wm, scheme, workflow.WorkflowTaskSetPlural, workflow.WorkflowTaskSetKind, true,
		func() runtime.Object { return &wfv1.WorkflowTaskSet{} },
	)
	return map[string]rest.Storage{
		"workflowtasksets":        NewGenericStore(cfg),
		"workflowtasksets/status": NewStatusStore(statusCfg),
	}
}

// NewWorkflowTaskResultStorage returns storage for workflowtaskresults.
func NewWorkflowTaskResultStorage(db *gorm.DB, wm *watchutil.Manager, scheme *runtime.Scheme) map[string]rest.Storage {
	cfg := newStoreConfig(db, wm, scheme, "workflowtaskresults", workflow.WorkflowTaskResultKind, true,
		func() runtime.Object { return &wfv1.WorkflowTaskResult{} },
		func() runtime.Object { return &wfv1.WorkflowTaskResultList{} },
	)
	return map[string]rest.Storage{
		"workflowtaskresults": NewGenericStore(cfg),
	}
}

// NewWorkflowArtifactGCTaskStorage returns storage for workflowartifactgctasks and workflowartifactgctasks/status.
func NewWorkflowArtifactGCTaskStorage(db *gorm.DB, wm *watchutil.Manager, scheme *runtime.Scheme) map[string]rest.Storage {
	cfg := newStoreConfig(db, wm, scheme, workflow.WorkflowArtifactGCTaskPlural, workflow.WorkflowArtifactGCTaskKind, true,
		func() runtime.Object { return &wfv1.WorkflowArtifactGCTask{} },
		func() runtime.Object { return &wfv1.WorkflowArtifactGCTaskList{} },
	)
	statusCfg := newStatusStoreConfig(db, wm, scheme, workflow.WorkflowArtifactGCTaskPlural, workflow.WorkflowArtifactGCTaskKind, true,
		func() runtime.Object { return &wfv1.WorkflowArtifactGCTask{} },
	)
	return map[string]rest.Storage{
		"workflowartifactgctasks":        NewGenericStore(cfg),
		"workflowartifactgctasks/status": NewStatusStore(statusCfg),
	}
}

// NewWorkflowEventBindingStorage returns storage for workfloweventbindings.
func NewWorkflowEventBindingStorage(db *gorm.DB, wm *watchutil.Manager, scheme *runtime.Scheme) map[string]rest.Storage {
	cfg := newStoreConfig(db, wm, scheme, workflow.WorkflowEventBindingPlural, workflow.WorkflowEventBindingKind, true,
		func() runtime.Object { return &wfv1.WorkflowEventBinding{} },
		func() runtime.Object { return &wfv1.WorkflowEventBindingList{} },
	)
	return map[string]rest.Storage{
		"workfloweventbindings": NewGenericStore(cfg),
	}
}
