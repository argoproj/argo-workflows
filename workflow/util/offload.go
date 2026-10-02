package util

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/retry"
	waitutil "github.com/argoproj/argo-workflows/v4/util/wait"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v4/workflow/validate"
)

// TemplateRepo is the interface used by offload-aware submission. Kept as a minimal local
// interface so this package does not need to couple to persist/sqldb directly.
type TemplateRepo interface {
	SaveTemplates(ctx context.Context, uid, namespace string, templates []wfv1.Template) error
}

// ComputeTemplateVersion hashes the name-sorted canonical form of a template set so version
// hashes are stable irrespective of storage/insertion order. We clone and sort rather than
// mutate the caller's slice, and hash in template-name order to match the deterministic
// order GetTemplates now returns.
func ComputeTemplateVersion(templates []wfv1.Template) string {
	sorted := make([]wfv1.Template, len(templates))
	copy(sorted, templates)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	hasher := sha256.New()
	for _, tmpl := range sorted {
		jsonBytes, _ := json.Marshal(tmpl)
		hasher.Write(jsonBytes)
	}
	return fmt.Sprintf("sha256:%x", hasher.Sum(nil))
}

// CRD array limits mirrored from the Workflow CRD. They fail the raw k8s create
// for non-offloaded specs; the offload path bypasses them by creating a stub.
const (
	maxDAGTasksPerTemplate  = 200
	maxTemplatesPerWorkflow = 200
)

// ShouldOffloadTemplates reports whether inline templates are large enough to warrant offloading.
// Templates at or below minSize stay in etcd; minSize <= 0 uses the configured default. Size alone
// is not enough to decide: the Workflow CRD caps dag.tasks and spec.templates array items, so a
// structurally oversized spec (e.g. an 800-task DAG at ~110KiB) cannot pass the raw create.
func ShouldOffloadTemplates(templates []wfv1.Template, minSize int) bool {
	if len(templates) == 0 {
		return false
	}
	if len(templates) > maxTemplatesPerWorkflow {
		return true
	}
	for i := range templates {
		if t := templates[i]; t.DAG != nil && len(t.DAG.Tasks) > maxDAGTasksPerTemplate {
			return true
		}
	}
	if minSize <= 0 {
		minSize = config.DefaultTemplateOffloadMinSize
	}
	jsonBytes, err := json.Marshal(templates)
	if err != nil {
		return true // cannot measure; offloading is the safe default for etcd size limits
	}
	return len(jsonBytes) > minSize
}

// SubmitWorkflowWithOffload validates and creates a workflow, offloading its templates to the
// database first (when a templateRepo is configured and the spec is large enough — see
// ShouldOffloadTemplates) so the etcd object stays a small stub. minOffloadSize is the minimum
// serialized template size in bytes above which templates offload; <= 0 uses the default.
func SubmitWorkflowWithOffload(ctx context.Context, wfIf v1alpha1.WorkflowInterface, wfClientset wfclientset.Interface, templateRepo TemplateRepo, namespace string, wf *wfv1.Workflow, wfDefaults *wfv1.Workflow, opts *wfv1.SubmitOpts, minOffloadSize int) (*wfv1.Workflow, error) {
	err := ApplySubmitOpts(wf, opts)
	if err != nil {
		return nil, err
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates())

	err = validate.Workflow(ctx, wftmplGetter, cwftmplGetter, wf, wfDefaults, validate.Opts{Submit: true})
	if err != nil {
		return nil, err
	}
	switch {
	case opts.DryRun:
		// Return the full spec unmodified - templates must NOT be stripped for a dry-run.
		return wf, nil
	case opts.ServerDryRun:
		createWf, createErr := CreateServerDryRun(ctx, wf, wfClientset)
		if createErr != nil {
			return nil, createErr
		}
		return createWf, createErr
	default:
		return CreateWorkflowWithOffload(ctx, wfIf, wfClientset, templateRepo, wf, minOffloadSize)
	}
}

// CreateWorkflowWithOffload creates a workflow, offloading its templates to the database first
// (when a templateRepo is configured and the spec is large enough — see ShouldOffloadTemplates)
// so the etcd object stays a small stub. The caller must have already validated the workflow;
// minOffloadSize is the minimum serialized template size above which templates offload (<=0 = default).
func CreateWorkflowWithOffload(ctx context.Context, wfIf v1alpha1.WorkflowInterface, wfClientset wfclientset.Interface, templateRepo TemplateRepo, wf *wfv1.Workflow, minOffloadSize int) (*wfv1.Workflow, error) {
	// Warn loudly when offloading is unavailable but the spec risks the 2MB API / 1MB etcd
	// limits. Measure the payload BEFORE stripping templates so we report the true size the
	// caller intended to submit.
	jsonBytes, _ := json.Marshal(wf)
	if templateRepo == nil && len(jsonBytes) > 1*1024*1024 {
		logging.RequireLoggerFromContext(ctx).WithField("payloadBytes", len(jsonBytes)).WithField("templateCount", len(wf.Spec.Templates)).Error(ctx, "Template offloading is NOT enabled in this mode and the workflow spec is large — it will exceed the 2MB API / 1MB etcd limits. Connect via the Argo Server (--argo-server) to enable template offloading.")
	}

	// Offload templates (when a templateRepo was passed and the templates are large enough) so
	// the etcd object stays a small stub.
	shouldOffload := templateRepo != nil && ShouldOffloadTemplates(wf.Spec.Templates, minOffloadSize)
	var originalTemplates []wfv1.Template
	if shouldOffload {
		originalTemplates = wf.Spec.Templates
		wf.Spec.Templates = nil
	}

	var runWf *wfv1.Workflow
	err := waitutil.Backoff(retry.DefaultRetry(ctx), func() (bool, error) {
		var createErr error
		runWf, createErr = wfIf.Create(ctx, wf, metav1.CreateOptions{})
		return !errorsutil.IsTransientErr(ctx, createErr), createErr
	})
	if err != nil {
		return nil, err
	}

	if shouldOffload {
		// Save templates to the DB first, then set StoredTemplateSpecs and update.
		// The fallback hydration path finds them by UID even before the marker lands.
		saveErr := templateRepo.SaveTemplates(ctx, string(runWf.UID), runWf.Namespace, originalTemplates)
		if saveErr != nil {
			if delErr := wfIf.Delete(ctx, runWf.Name, metav1.DeleteOptions{}); delErr != nil && !apierr.IsNotFound(delErr) {
				return nil, fmt.Errorf("failed to save templates: %w; also failed to clean up stub %s: %w", saveErr, runWf.Name, delErr)
			}
			return nil, fmt.Errorf("failed to save templates: %w", saveErr)
		}
		runWf.Status.StoredTemplateSpecs = &wfv1.TemplateSpecReference{
			UID:      string(runWf.UID),
			Version:  ComputeTemplateVersion(originalTemplates),
			Hydrated: false,
		}
		if _, updErr := wfClientset.ArgoprojV1alpha1().Workflows(runWf.Namespace).Update(ctx, runWf, metav1.UpdateOptions{}); updErr != nil {
			logging.RequireLoggerFromContext(ctx).WithError(updErr).WithField("wf", runWf.Name).Warn(ctx, "Failed to update workflow status with StoredTemplateSpecs after offload")
		}
	}
	return runWf, nil
}
