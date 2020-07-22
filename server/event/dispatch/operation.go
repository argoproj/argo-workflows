package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/antonmedv/expr"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/util/labels"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/creator"
)

type Operation struct {
	ctx               context.Context
	instanceIDService instanceid.Service
	events            []wfv1.WorkflowEvent
	env               map[string]interface{}
}

func NewOperation(ctx context.Context, instanceIDService instanceid.Service, events []wfv1.WorkflowEvent, discriminator string, payload *wfv1.Item) (*Operation, error) {
	env, err := expressionEnvironment(ctx, discriminator, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow template expression environment: %w", err)
	}
	return &Operation{
		ctx:               ctx,
		instanceIDService: instanceIDService,
		events:            events,
		env:               env,
	}, nil
}

func (o *Operation) Execute() {
	log.Debug("Executing event dispatch")

	data, _ := json.MarshalIndent(o.env, "", "  ")
	log.Debugln(string(data))

	for _, event := range o.events {
		err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
			_, err := o.submitWorkflowsFromEvent(event)
			return err == nil, err
		})
		if err != nil {
			log.WithError(err).WithFields(log.Fields{"namespace": event.Namespace, "event": event.Name}).Error("failed to submit workflows from event")
		}
	}
}

func (o *Operation) submitWorkflowsFromEvent(event wfv1.WorkflowEvent) (*wfv1.Workflow, error) {
	result, err := expr.Eval(event.Spec.Expression, o.env)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate workflow template expression: %w", err)
	}
	matched, boolExpr := result.(bool)
	log.WithFields(log.Fields{
		"namespace":  event.Namespace,
		"event":      event.Name,
		"expression": event.Spec.Expression,
		"matched":    matched,
		"boolExpr":   boolExpr,
	}).Debug("Expression evaluation")

	if !boolExpr {
		return nil, errors.New("malformed workflow template expression: did not evaluate to boolean")
	} else if matched {
		client := auth.GetWfClient(o.ctx)
		tmpl, err := client.ArgoprojV1alpha1().WorkflowTemplates(event.Namespace).Get(event.Spec.WorkflowTemplate.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get workflow template: %w", err)
		}
		err = o.instanceIDService.Validate(tmpl)
		if err != nil {
			return nil, fmt.Errorf("failed to validate workflow template instanceid: %w", err)
		}
		wf := common.NewWorkflowFromWorkflowTemplate(tmpl.Name, tmpl.Spec.WorkflowMetadata, false)
		o.instanceIDService.Label(wf)
		// users will always want to know why a workflow was submitted,
		// so we label with creator (which is a standard) and the name of the triggering event
		creator.Label(o.ctx, wf)
		labels.Label(wf, common.LabelKeyWorkflowEvent, event.Name)
		for _, p := range event.Spec.WorkflowTemplate.Parameters {
			if p.ValueFrom == nil {
				return nil, fmt.Errorf("malformed workflow template parameter \"%s\": validFrom is nil", p.Name)
			}
			result, err := expr.Eval(p.ValueFrom.Expression, o.env)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate workflow template parameter \"%s\" expression: %w", p.Name, err)
			}
			intOrString := intstr.Parse(fmt.Sprintf("%v", result))
			wf.Spec.Arguments.Parameters = append(wf.Spec.Arguments.Parameters, wfv1.Parameter{Name: p.Name, Value: &intOrString})
		}
		wf, err = client.ArgoprojV1alpha1().Workflows(tmpl.Namespace).Create(wf)
		if err != nil {
			return nil, fmt.Errorf("failed to create workflow: %w", err)
		}
		return wf, nil
	}
	return nil, nil
}

func expressionEnvironment(ctx context.Context, discriminator string, payload *wfv1.Item) (map[string]interface{}, error) {
	src := map[string]interface{}{
		"discriminator": discriminator,
		"metadata":      metaData(ctx),
		"payload":       payload,
	}
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	env := make(map[string]interface{})
	return env, json.Unmarshal(data, &env)
}

func metaData(ctx context.Context) map[string]interface{} {
	meta := map[string]interface{}{
		"claimSet": auth.GetClaimSet(ctx),
	}
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
