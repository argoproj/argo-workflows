package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"google.golang.org/grpc/metadata"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/expr/argoexpr"
	exprenv "github.com/argoproj/argo-workflows/v3/util/expr/env"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	jsonutil "github.com/argoproj/argo-workflows/v3/util/json"
	"github.com/argoproj/argo-workflows/v3/util/labels"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/creator"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

type Operation struct {
	//nolint: containedctx
	ctx               context.Context
	eventRecorder     record.EventRecorder
	instanceIDService instanceid.Service
	events            []wfv1.WorkflowEventBinding
	env               map[string]interface{}
}

// Context returns the context associated with this operation
func (o *Operation) Context() context.Context {
	return o.ctx
}

func NewOperation(ctx context.Context, instanceIDService instanceid.Service, eventRecorder record.EventRecorder, events []wfv1.WorkflowEventBinding, namespace, discriminator string, payload *wfv1.Item) (*Operation, error) {
	env, err := expressionEnvironment(ctx, namespace, discriminator, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow template expression environment: %w", err)
	}
	return &Operation{
		ctx:               ctx,
		eventRecorder:     eventRecorder,
		instanceIDService: instanceIDService,
		events:            events,
		env:               env,
	}, nil
}

// not to be converted with sutils, parent calling function should handle this
// responsibility
func (o *Operation) Dispatch(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx)

	logger.Debug(ctx, "Executing event dispatch")

	data, _ := json.MarshalIndent(o.env, "", "  ")
	logger.Debug(ctx, string(data))

	var errs []error
	for _, event := range o.events {
		err := waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
			_, err := o.dispatch(ctx, event)
			return !errorsutil.IsTransientErr(ctx, err), err
		})
		if err != nil {
			logger.WithError(err).WithFields(logging.Fields{"namespace": event.Namespace, "event": event.Name}).Error(ctx, "failed to dispatch from event")
			o.eventRecorder.Event(&event, corev1.EventTypeWarning, "WorkflowEventBindingError", "failed to dispatch event: "+err.Error())
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to dispatch event: %v", errs)
	}
	return nil
}

func (o *Operation) dispatch(ctx context.Context, wfeb wfv1.WorkflowEventBinding) (*wfv1.Workflow, error) {
	logger := logging.RequireLoggerFromContext(ctx)

	selector := wfeb.Spec.Event.Selector
	matched, err := argoexpr.EvalBool(selector, o.env)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate workflow template expression: %w", err)
	}
	logger.WithFields(logging.Fields{"namespace": wfeb.Namespace, "event": wfeb.Name, "selector": selector, "matched": matched}).Debug(ctx, "Selector evaluation")
	submit := wfeb.Spec.Submit
	if matched && submit != nil {
		//nolint: contextcheck
		client := auth.GetWfClient(o.ctx)
		ref := wfeb.Spec.Submit.WorkflowTemplateRef
		var tmpl wfv1.WorkflowSpecHolder
		var err error
		if ref.ClusterScope {
			tmpl, err = client.ArgoprojV1alpha1().ClusterWorkflowTemplates().Get(ctx, ref.Name, metav1.GetOptions{})
		} else {
			tmpl, err = client.ArgoprojV1alpha1().WorkflowTemplates(wfeb.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get workflow template: %w", err)
		}
		err = o.instanceIDService.Validate(tmpl)
		if err != nil {
			return nil, fmt.Errorf("failed to validate workflow template instanceid: %w", err)
		}
		wf := common.NewWorkflowFromWorkflowTemplate(tmpl.GetName(), ref.ClusterScope)
		o.instanceIDService.Label(wf)
		err = o.populateWorkflowMetadata(wf, &submit.ObjectMeta)
		if err != nil {
			return nil, err
		}

		if wf.Name == "" {
			wf.SetName(wf.GetGenerateName() + util.RandSuffix())
		}

		// users will always want to know why a workflow was submitted,
		// so we label with creator (which is a standard) and the name of the triggering event
		//nolint: contextcheck
		creator.LabelCreator(o.ctx, wf)
		labels.Label(wf, common.LabelKeyWorkflowEventBinding, wfeb.Name)
		if submit.Arguments != nil {
			for _, p := range submit.Arguments.Parameters {
				if p.ValueFrom == nil {
					return nil, fmt.Errorf("malformed workflow template parameter \"%s\": valueFrom is nil", p.Name)
				}
				program, err := expr.Compile(p.ValueFrom.Event, expr.Env(o.env))
				if err != nil {
					return nil, fmt.Errorf("failed to compile workflow template parameter %s expression: %w", p.Name, err)
				}
				result, err := expr.Run(program, o.env)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate workflow template parameter \"%s\" expression: %w", p.Name, err)
				}
				data, err := json.Marshal(result)
				if err != nil {
					return nil, fmt.Errorf("failed to convert result to JSON \"%s\" expression: %w", p.Name, err)
				}
				wf.Spec.Arguments.Parameters = append(wf.Spec.Arguments.Parameters, wfv1.Parameter{Name: p.Name, Value: wfv1.AnyStringPtr(wfv1.Item{Value: data})})
			}
		}
		wf, err = client.ArgoprojV1alpha1().Workflows(wfeb.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create workflow: %w", err)
		}
		return wf, nil
	}
	return nil, nil
}

func (o *Operation) populateWorkflowMetadata(wf *wfv1.Workflow, metadata *metav1.ObjectMeta) error {
	if len(metadata.Name) > 0 {
		evalName, err := o.evaluateStringExpression(metadata.Name, "name")
		if err != nil {
			return err
		}
		wf.SetName(evalName)
	}
	if len(metadata.GenerateName) > 0 {
		evalName, err := o.evaluateStringExpression(metadata.GenerateName, "generateName")
		if err != nil {
			return err
		}
		wf.GenerateName = evalName
	}
	for labelKey, labelValue := range metadata.Labels {
		evalLabel, err := o.evaluateStringExpression(labelValue, fmt.Sprintf("label \"%s\"", labelKey))
		if err != nil {
			return err
		}
		// This is invariant code, but it's a convenient way to only initialize labels if there are actually labels
		// defined. Given that there will likely be few user defined labels this shouldn't affect performance at all.
		if wf.Labels == nil {
			wf.Labels = map[string]string{}
		}
		wf.Labels[labelKey] = evalLabel
	}
	for annotationKey, annotationValue := range metadata.Annotations {
		evalAnnotation, err := o.evaluateStringExpression(annotationValue, fmt.Sprintf("annotation \"%s\"", annotationKey))
		if err != nil {
			return err
		}
		// See labels comment above.
		if wf.Annotations == nil {
			wf.Annotations = map[string]string{}
		}
		wf.Annotations[annotationKey] = evalAnnotation
	}
	return nil
}

func (o *Operation) evaluateStringExpression(statement string, errorInfo string) (string, error) {
	env := exprenv.GetFuncMap(o.env)
	program, err := expr.Compile(statement, expr.Env(env))
	if err != nil {
		return "", fmt.Errorf("failed to evaluate workflow %s expression: %w", errorInfo, err)
	}
	result, err := expr.Run(program, env)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate workflow %s expression: %w", errorInfo, err)
	}

	v, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("workflow %s expression must evaluate to a string, not a %T", errorInfo, result)
	}
	return v, nil
}

func expressionEnvironment(ctx context.Context, namespace, discriminator string, payload *wfv1.Item) (map[string]interface{}, error) {
	src := map[string]interface{}{
		"namespace":     namespace,
		"discriminator": discriminator,
		"metadata":      metaData(ctx),
		"payload":       payload,
	}
	return jsonutil.Jsonify(src)
}

func metaData(ctx context.Context) map[string]interface{} {
	meta := make(map[string]interface{})
	md, _ := metadata.FromIncomingContext(ctx)
	for k, v := range md {
		// only allow headers `X-`  headers, e.g. `X-Github-Action`
		// otherwise, deny, e.g. deny `authorization` as this would leak security credentials
		if strings.HasPrefix(k, "x-") {
			meta[k] = v
		}
	}
	return meta
}
