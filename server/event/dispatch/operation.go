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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/creator"
)

type Operation struct {
	ctx                       context.Context
	instanceIDService         instanceid.Service
	workflowTemplateKeyLister cache.KeyLister
	payload                   *wfv1.Item
	discriminator             string
}

func NewOperation(ctx context.Context, instanceIDService instanceid.Service, workflowTemplateKeyLister cache.KeyLister, discriminator string, payload *wfv1.Item) Operation {
	return Operation{
		ctx:                       ctx,
		instanceIDService:         instanceIDService,
		workflowTemplateKeyLister: workflowTemplateKeyLister,
		payload:                   payload,
		discriminator:             discriminator,
	}
}

func (o *Operation) Execute() {
	log.Debug("Executing event dispatch")
	o.submitWorkflowsFromWorkflowTemplates()
}

func (o *Operation) submitWorkflowsFromWorkflowTemplates() {
	for _, key := range o.workflowTemplateKeyLister.ListKeys() {
		namespace, name, _ := cache.SplitMetaNamespaceKey(key)
		err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
			err := o.submitWorkflowsFromWorkflowTemplate(namespace, name)
			return err == nil, err
		})
		if err != nil {
			log.WithError(err).WithFields(log.Fields{"namespace": namespace, "template": name}).Error("failed to submit workflow from template")
		}
	}
}

func (o *Operation) submitWorkflowsFromWorkflowTemplate(namespace, name string) error {
	client := auth.GetWfClient(o.ctx)
	tmpl, err := client.ArgoprojV1alpha1().WorkflowTemplates(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get workflow template: %w", err)
	}
	env, err := expressionEnvironment(o.ctx, o.discriminator, o.payload)
	if err != nil {
		return fmt.Errorf("failed to create workflow template expression environment (should by impossible): %w", err)
	}
	for _, event := range tmpl.Spec.Events {
		result, err := expr.Eval(event.Expression, env)
		if err != nil {
			return fmt.Errorf("failed to evaluate workflow template expression: %w", err)
		}
		matched, boolExpr := result.(bool)
		log.WithFields(log.Fields{
			"namespace":  namespace,
			"name":       name,
			"expression": event.Expression,
			"matched":    matched,
			"boolExpr":   boolExpr,
		}).Debug("Expression evaluation")

		data, _ := json.MarshalIndent(env, "", "  ")
		log.Debugln(string(data))

		if !boolExpr {
			return errors.New("malformed workflow template expression: did not evaluate to boolean")
		} else if matched {
			wf := common.NewWorkflowFromWorkflowTemplate(tmpl.Name, tmpl.Spec.WorkflowMetadata, false)
			o.instanceIDService.Label(wf)
			creator.Label(o.ctx, wf)
			for _, p := range event.Parameters {
				if p.ValueFrom == nil {
					return fmt.Errorf("malformed workflow template parameter \"%s\": validFrom is nil", p.Name)
				}
				result, err := expr.Eval(p.ValueFrom.Expression, env)
				if err != nil {
					return fmt.Errorf("failed to evaluate workflow template parameter \"%s\" expression: %w", p.Name, err)
				}
				intOrString := intstr.Parse(fmt.Sprintf("%v", result))
				wf.Spec.Arguments.Parameters = append(wf.Spec.Arguments.Parameters, wfv1.Parameter{Name: p.Name, Value: &intOrString})
			}
			_, err = client.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
			if err != nil {
				return fmt.Errorf("failed to create workflow: %w", err)
			}
		}
	}
	return nil
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
