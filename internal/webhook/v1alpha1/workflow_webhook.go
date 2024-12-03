package v1alpha1

import (
	"context"
	"fmt"

	argoprojiov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argovalidate "github.com/argoproj/argo-workflows/v3/workflow/validate"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:unused
// log is for logging in this package.
var workflowlog = logf.Log.WithName("workflow-validating-webhook")

// SetupWorkflowWebhookWithManager registers the webhook for Workflow in the manager.
func SetupWorkflowWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&argoprojiov1alpha1.Workflow{}).
		WithValidator(&WorkflowCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-argoproj-io-argoproj-v1alpha1-workflow,mutating=false,failurePolicy=fail,sideEffects=None,groups=argoproj.io.argoproj,resources=workflows,verbs=create;update,versions=v1alpha1,name=vworkflow-v1alpha1.kb.io,admissionReviewVersions=v1

// WorkflowCustomValidator struct is responsible for validating the Workflow resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type WorkflowCustomValidator struct {
	ctrlClient client.Client
}

var _ webhook.CustomValidator = &WorkflowCustomValidator{}

func (v *WorkflowCustomValidator) validate(ctx context.Context, workflow *argoprojiov1alpha1.Workflow) (admission.Warnings, error) {
	if err := argovalidate.ValidateWorkflow(
		&workflowTemplateGetter{client: v.ctrlClient, namespace: workflow.GetNamespace(), ctx: ctx},
		&clusterWorkflowTemplateGetter{client: v.ctrlClient, ctx: ctx},
		workflow,
		argovalidate.ValidateOpts{Submit: true},
	); err != nil {
		workflowlog.Error(
			err, "admission request denied",
			"kind", "Workflow",
			"generateName", workflow.GetGenerateName(),
			"name", workflow.GetName(),
			"namespace", workflow.GetNamespace(),
		)
		return admission.Warnings{fmt.Sprintf("workflow failed validation: %s", err)}, err
	}
	return nil, nil
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Workflow.
func (v *WorkflowCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	workflow, ok := obj.(*argoprojiov1alpha1.Workflow)
	if !ok {
		return nil, fmt.Errorf("expected a Workflow object but got %T", obj)
	}
	workflowlog.Info("Validation for Workflow upon creation", "name", workflow.GetName())

	// TODO(user): fill in your validation logic upon object creation.
	return v.validate(ctx, workflow)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Workflow.
func (v *WorkflowCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	workflow, ok := newObj.(*argoprojiov1alpha1.Workflow)
	if !ok {
		return nil, fmt.Errorf("expected a Workflow object for the newObj but got %T", newObj)
	}
	workflowlog.Info("Validation for Workflow upon update", "name", workflow.GetName())

	// TODO(user): fill in your validation logic upon object update.
	return v.validate(ctx, workflow)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Workflow.
func (v *WorkflowCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	workflow, ok := obj.(*argoprojiov1alpha1.Workflow)
	if !ok {
		return nil, fmt.Errorf("expected a Workflow object but got %T", obj)
	}
	workflowlog.Info("Validation for Workflow upon deletion", "name", workflow.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
