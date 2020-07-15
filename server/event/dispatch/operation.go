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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

type Operation struct {
	// system context
	client                    versioned.Interface
	workflowTemplateKeyLister cache.KeyLister
	// about the event
	event    *wfv1.Item
	metadata map[string]interface{}
}

func NewOperation(ctx context.Context, workflowTemplateKeyLister cache.KeyLister, event *wfv1.Item) Operation {
	return Operation{
		client:                    auth.GetWfClient(ctx),
		workflowTemplateKeyLister: workflowTemplateKeyLister,
		event:                     event,
		metadata:                  metaData(ctx),
	}
}

func (s *Operation) Execute() {
	s.submitWorkflowsFromWorkflowTemplates()
}

func (s *Operation) submitWorkflowsFromWorkflowTemplates() {
	for _, key := range s.workflowTemplateKeyLister.ListKeys() {
		namespace, name, _ := cache.SplitMetaNamespaceKey(key)
		err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
			err := s.submitWorkflowFromWorkflowTemplate(namespace, name)
			return err == nil, err
		})
		if err != nil {
			log.WithError(err).WithFields(log.Fields{"namespace": namespace, "template": name}).Error("failed to submit workflow from template")
		}
	}
}

func (s *Operation) submitWorkflowFromWorkflowTemplate(namespace, name string) error {
	tmpl, err := s.client.ArgoprojV1alpha1().WorkflowTemplates(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get workflow template: %w", err)
	}
	if tmpl.Spec.Event == nil {
		// we should have filtered this out
		return errors.New("event spec is missing (should be impossible)")
	}
	env, err := expressionEnvironment(map[string]interface{}{"event": s.event, "metadata": s.metadata})
	if err != nil {
		return fmt.Errorf("failed to create workflow template expression environment (should by impossible): %w", err)
	}
	result, err := expr.Eval(tmpl.Spec.Event.Expression, env)
	if err != nil {
		return fmt.Errorf("failed to evaluate workflow template expression: %w", err)
	}
	matched, ok := result.(bool)
	if !ok {
		return errors.New("malformed workflow template expression: did not evaluate to boolean")
	} else if matched {
		parameters := make([]string, len(tmpl.Spec.Arguments.Parameters))
		for i, p := range tmpl.Spec.Arguments.Parameters {
			if p.ValueFrom == nil {
				return fmt.Errorf("malformed workflow template: parameter \"%s\" valueFrom is nil", p.Name)
			}
			result, err := expr.Eval(p.ValueFrom.Expression, env)
			if err != nil {
				return fmt.Errorf("workflow templates parameter \"%s\" expression failed to evaluate: %w", p.Name, err)
			}
			parameters[i] = fmt.Sprintf("%s=%v", p.Name, result)
		}

		wf := common.NewWorkflowFromWorkflowTemplate(tmpl.Name, tmpl.Spec.WorkflowMetadata, false)
		var labels []string
		for k, v := range tmpl.GetLabels() {
			labels = append(labels, k+"="+v)
		}
		err := util.ApplySubmitOpts(wf, &wfv1.SubmitOpts{Parameters: parameters, Labels: strings.Join(labels, ",")})
		if err != nil {
			return fmt.Errorf("failed to apply submit options to workflow: %w", err)
		}
		_, err = s.client.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
		if err != nil {
			return fmt.Errorf("failed to create workflow: %w", err)
		}
	}
	return nil
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
