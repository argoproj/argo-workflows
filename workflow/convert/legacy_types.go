package convert

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// Legacy types for parsing manifests with deprecated fields.
// These types include both old (deprecated) and new fields to support
// parsing v3.5 and earlier manifests.

// LegacySynchronization can parse both old (semaphore/mutex) and new (semaphores/mutexes) formats
type LegacySynchronization struct {
	// Deprecated v3.5 and before: singular semaphore
	Semaphore *wfv1.SemaphoreRef `json:"semaphore,omitempty"`
	// Deprecated v3.5 and before: singular mutex
	Mutex *wfv1.Mutex `json:"mutex,omitempty"`
	// v3.6 and after: plural semaphores
	Semaphores []*wfv1.SemaphoreRef `json:"semaphores,omitempty"`
	// v3.6 and after: plural mutexes
	Mutexes []*wfv1.Mutex `json:"mutexes,omitempty"`
}

// ToCurrent converts a LegacySynchronization to the current Synchronization type
func (ls *LegacySynchronization) ToCurrent() *wfv1.Synchronization {
	if ls == nil {
		return nil
	}

	sync := &wfv1.Synchronization{}

	// Copy semaphores to avoid aliasing, then append singular if present
	// Unlike schedules, we honor both singular and plurals here
	sync.Semaphores = make([]*wfv1.SemaphoreRef, len(ls.Semaphores))
	copy(sync.Semaphores, ls.Semaphores)
	if ls.Semaphore != nil {
		sync.Semaphores = append(sync.Semaphores, ls.Semaphore)
	}

	// Copy mutexes to avoid aliasing, then append singular if present
	sync.Mutexes = make([]*wfv1.Mutex, len(ls.Mutexes))
	copy(sync.Mutexes, ls.Mutexes)
	if ls.Mutex != nil {
		sync.Mutexes = append(sync.Mutexes, ls.Mutex)
	}

	return sync
}

// LegacyTemplate wraps Template with legacy synchronization support.
// The explicit Synchronization field intentionally shadows wfv1.Template.Synchronization
// to allow parsing both legacy (singular semaphore/mutex) and current (plural) formats.
// During JSON unmarshaling, the explicit field takes precedence.
type LegacyTemplate struct {
	wfv1.Template
	Synchronization *LegacySynchronization `json:"synchronization,omitempty"`
}

// ToCurrent converts a LegacyTemplate to the current Template type
func (lt *LegacyTemplate) ToCurrent() wfv1.Template {
	tmpl := lt.Template
	if lt.Synchronization != nil {
		tmpl.Synchronization = lt.Synchronization.ToCurrent()
	}
	return tmpl
}

// LegacyWorkflowSpec wraps WorkflowSpec with legacy synchronization support.
// The explicit Synchronization and Templates fields intentionally shadow
// wfv1.WorkflowSpec.Synchronization and wfv1.WorkflowSpec.Templates to allow
// parsing both legacy (singular semaphore/mutex/schedule) and current (plural) formats.
// During JSON unmarshaling, explicit fields take precedence.
type LegacyWorkflowSpec struct {
	wfv1.WorkflowSpec
	Synchronization *LegacySynchronization `json:"synchronization,omitempty"`
	Templates       []LegacyTemplate       `json:"templates,omitempty"`
}

// ToCurrent converts a LegacyWorkflowSpec to the current WorkflowSpec type
func (lws *LegacyWorkflowSpec) ToCurrent() wfv1.WorkflowSpec {
	spec := lws.WorkflowSpec

	// Convert synchronization
	if lws.Synchronization != nil {
		spec.Synchronization = lws.Synchronization.ToCurrent()
	}

	// Convert templates
	if len(lws.Templates) > 0 {
		spec.Templates = make([]wfv1.Template, len(lws.Templates))
		for i, legacyTmpl := range lws.Templates {
			spec.Templates[i] = legacyTmpl.ToCurrent()
		}
	}

	return spec
}

// LegacyCronWorkflowSpec can parse both old (schedule) and new (schedules) formats
type LegacyCronWorkflowSpec struct {
	WorkflowSpec               LegacyWorkflowSpec     `json:"workflowSpec"`
	Schedule                   string                 `json:"schedule,omitempty"`  // Deprecated v3.5 and before
	Schedules                  []string               `json:"schedules,omitempty"` // v3.6 and after
	ConcurrencyPolicy          wfv1.ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`
	Suspend                    bool                   `json:"suspend,omitempty"`
	StartingDeadlineSeconds    *int64                 `json:"startingDeadlineSeconds,omitempty"`
	SuccessfulJobsHistoryLimit *int32                 `json:"successfulJobsHistoryLimit,omitempty"`
	FailedJobsHistoryLimit     *int32                 `json:"failedJobsHistoryLimit,omitempty"`
	Timezone                   string                 `json:"timezone,omitempty"`
	WorkflowMetadata           *metav1.ObjectMeta     `json:"workflowMetadata,omitempty"`
	StopStrategy               *wfv1.StopStrategy     `json:"stopStrategy,omitempty"`
	When                       string                 `json:"when,omitempty"`
}

// ToCurrent converts a LegacyCronWorkflowSpec to the current CronWorkflowSpec type
func (lcs *LegacyCronWorkflowSpec) ToCurrent() wfv1.CronWorkflowSpec {
	spec := wfv1.CronWorkflowSpec{
		WorkflowSpec:               lcs.WorkflowSpec.ToCurrent(),
		Schedules:                  lcs.Schedules,
		ConcurrencyPolicy:          lcs.ConcurrencyPolicy,
		Suspend:                    lcs.Suspend,
		StartingDeadlineSeconds:    lcs.StartingDeadlineSeconds,
		SuccessfulJobsHistoryLimit: lcs.SuccessfulJobsHistoryLimit,
		FailedJobsHistoryLimit:     lcs.FailedJobsHistoryLimit,
		Timezone:                   lcs.Timezone,
		WorkflowMetadata:           lcs.WorkflowMetadata,
		StopStrategy:               lcs.StopStrategy,
		When:                       lcs.When,
	}

	// Migrate singular schedule to plural if needed
	// Unlike synchronization we didn't honor both singular and plural, singular has priority
	if lcs.Schedule != "" {
		spec.Schedules = []string{lcs.Schedule}
	}

	return spec
}

// LegacyCronWorkflow wraps CronWorkflow with legacy field support
type LegacyCronWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              LegacyCronWorkflowSpec  `json:"spec"`
	Status            wfv1.CronWorkflowStatus `json:"status,omitempty"`
}

// ToCurrent converts a LegacyCronWorkflow to the current CronWorkflow type
func (lcw *LegacyCronWorkflow) ToCurrent() *wfv1.CronWorkflow {
	return &wfv1.CronWorkflow{
		TypeMeta:   lcw.TypeMeta,
		ObjectMeta: lcw.ObjectMeta,
		Spec:       lcw.Spec.ToCurrent(),
		Status:     lcw.Status,
	}
}

// LegacyWorkflow wraps Workflow with legacy field support
type LegacyWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              LegacyWorkflowSpec  `json:"spec"`
	Status            wfv1.WorkflowStatus `json:"status,omitzero"`
}

// ToCurrent converts a LegacyWorkflow to the current Workflow type
func (lw *LegacyWorkflow) ToCurrent() *wfv1.Workflow {
	return &wfv1.Workflow{
		TypeMeta:   lw.TypeMeta,
		ObjectMeta: lw.ObjectMeta,
		Spec:       lw.Spec.ToCurrent(),
		Status:     lw.Status,
	}
}

// LegacyWorkflowTemplate wraps WorkflowTemplate with legacy field support
type LegacyWorkflowTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              LegacyWorkflowSpec `json:"spec"`
}

// ToCurrent converts a LegacyWorkflowTemplate to the current WorkflowTemplate type
func (lwt *LegacyWorkflowTemplate) ToCurrent() *wfv1.WorkflowTemplate {
	return &wfv1.WorkflowTemplate{
		TypeMeta:   lwt.TypeMeta,
		ObjectMeta: lwt.ObjectMeta,
		Spec:       lwt.Spec.ToCurrent(),
	}
}

// LegacyClusterWorkflowTemplate wraps ClusterWorkflowTemplate with legacy field support
type LegacyClusterWorkflowTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              LegacyWorkflowSpec `json:"spec"`
}

// ToCurrent converts a LegacyClusterWorkflowTemplate to the current ClusterWorkflowTemplate type
func (lcwt *LegacyClusterWorkflowTemplate) ToCurrent() *wfv1.ClusterWorkflowTemplate {
	return &wfv1.ClusterWorkflowTemplate{
		TypeMeta:   lcwt.TypeMeta,
		ObjectMeta: lcwt.ObjectMeta,
		Spec:       lcwt.Spec.ToCurrent(),
	}
}
