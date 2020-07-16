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
	// system context
	ctx                       context.Context
	instanceIDService         instanceid.Service
	workflowTemplateKeyLister cache.KeyLister
	// about the event
	event *wfv1.Item
}

func NewOperation(ctx context.Context, instanceIDService instanceid.Service, workflowTemplateKeyLister cache.KeyLister, event *wfv1.Item) Operation {
	return Operation{
		ctx:                       ctx,
		instanceIDService:         instanceIDService,
		workflowTemplateKeyLister: workflowTemplateKeyLister,
		event:                     event,
	}
}

func (o *Operation) Execute() {
	o.submitWorkflowsFromWorkflowTemplates()
}

func (o *Operation) submitWorkflowsFromWorkflowTemplates() {
	for _, key := range o.workflowTemplateKeyLister.ListKeys() {
		namespace, name, _ := cache.SplitMetaNamespaceKey(key)
		err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
			_, err := o.submitWorkflowFromWorkflowTemplate(namespace, name)
			return err == nil, err
		})
		if err != nil {
			log.WithError(err).WithFields(log.Fields{"namespace": namespace, "template": name}).Error("failed to submit workflow from template")
		}
	}
}

func (o *Operation) submitWorkflowFromWorkflowTemplate(namespace, name string) (*wfv1.Workflow, error) {
	client := auth.GetWfClient(o.ctx)
	tmpl, err := client.ArgoprojV1alpha1().WorkflowTemplates(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow template: %w", err)
	}
	if tmpl.Spec.Event == nil {
		// we should have filtered this out
		return nil, errors.New("event spec is missing (should be impossible)")
	}
	env, err := expressionEnvironment(map[string]interface{}{"event": o.event, "metadata": metaData(o.ctx)})
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow template expression environment (should by impossible): %w", err)
	}
	result, err := expr.Eval(tmpl.Spec.Event.Expression, env)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate workflow template expression: %w", err)
	}
	matched, ok := result.(bool)
	if !ok {
		return nil, errors.New("malformed workflow template expression: did not evaluate to boolean")
	} else if matched {
		wf := common.NewWorkflowFromWorkflowTemplate(tmpl.Name, tmpl.Spec.WorkflowMetadata, false)
		o.instanceIDService.Label(wf)
		creator.Label(o.ctx, wf)
		for _, p := range tmpl.Spec.Event.Parameters {
			result, err := expr.Eval(p.Expression, env)
			if err != nil {
				return nil, fmt.Errorf("failed to evalute workflow template parameter \"%s\" expression: %w", p.Name, err)
			}
			intOrString := intstr.Parse(fmt.Sprintf("%v", result))
			wf.Spec.Arguments.Parameters = append(wf.Spec.Arguments.Parameters, wfv1.Parameter{Name: p.Name, Value: &intOrString})
		}
		wf, err = client.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
		if err != nil {
			return nil, fmt.Errorf("failed to create workflow: %w", err)
		}
		return wf, nil
	}
	return nil, nil
}

func expressionEnvironment(src map[string]interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	log.WithField("data", string(data)).Debug("Expression environment")
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
