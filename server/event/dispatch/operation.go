package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/antonmedv/expr"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/hydrator"
	"github.com/argoproj/argo/workflow/util"
)

type Operation struct {
	// system context
	client                    versioned.Interface
	hydrator                  hydrator.Interface
	workflowKeyLister         cache.KeyLister
	workflowTemplateKeyLister cache.KeyLister
	// about the event
	event    *wfv1.Item
	metadata map[string]interface{}
}

func NewOperation(ctx context.Context, hydrator hydrator.Interface, workflowKeyLister cache.KeyLister, workflowTemplateKeyLister cache.KeyLister, event *wfv1.Item) Operation {
	return Operation{
		client:                    auth.GetWfClient(ctx),
		hydrator:                  hydrator,
		workflowKeyLister:         workflowKeyLister,
		workflowTemplateKeyLister: workflowTemplateKeyLister,
		event:                     event,
		metadata:                  metaData(ctx),
	}
}

func (s *Operation) Execute() {
	s.submitWorkflowsFromWorkflowTemplates()
	s.resumeWorkflows()
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

		wf := common.NewWorkflowFromWorkflowTemplate(tmpl.Name, false)
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

func (s *Operation) resumeWorkflows() {
	for _, key := range s.workflowKeyLister.ListKeys() {
		namespace, name, _ := cache.SplitMetaNamespaceKey(key)
		err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
			err := s.resumeWorkflow(namespace, name)
			return err == nil, err
		})
		if err != nil {
			log.WithFields(log.Fields{"namespace": namespace, "workflow": name}).WithError(err).Error("failed to resume workflow")
		}
	}
}

func (s *Operation) resumeWorkflow(namespace, name string) error {
	wf, err := s.client.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}
	err = s.hydrator.Hydrate(wf)
	if err != nil {
		return fmt.Errorf("failed to hydrate workflow: %w", err)
	}
	updated := false
	for _, node := range wf.Status.Nodes {
		if node.Phase == wfv1.NodeRunning && node.Type == wfv1.NodeTypeSuspend {
			t := wf.GetTemplateByName(node.TemplateName)
			if t == nil || t.Suspend == nil || t.Suspend.Event == nil {
				return errors.New("template, suspend, or event is nil (should be impossible)")
			}
			env, err := expressionEnvironment(map[string]interface{}{"event": s.event, "inputs": node.Inputs, "metadata": s.metadata})
			if err != nil {
				return fmt.Errorf("failed to create workflow expression environment (should be impossible): %w", err)
			}
			result, err := expr.Eval(t.Suspend.Event.Expression, env)
			if err != nil {
				// this is a condition, because it is possible for events to fail expression, but it not really be a problem with the expression
				wf.Status.Conditions.UpsertCondition(wfv1.Condition{Status: metav1.ConditionTrue, Type: wfv1.ConditionTypeEventExpressionError, Message: err.Error()})
			} else {
				matches, ok := result.(bool)
				if !ok {
					node = markNodeStatus(wf, node, wfv1.NodeError, "malformed workflow expression: did not evaluate to a boolean")
				} else if matches {
					node.Outputs = &wfv1.Outputs{Parameters: make([]wfv1.Parameter, len(t.Outputs.Parameters))}
					for i, p := range t.Outputs.Parameters {
						if p.ValueFrom == nil {
							node = markNodeStatus(wf, node, wfv1.NodeError, "malformed output parameter \""+p.Name+"\": valueFrom is nil")
							break
						}
						value, err := expr.Eval(p.ValueFrom.Expression, env)
						if err != nil {
							node = markNodeStatus(wf, node, wfv1.NodeError, "output parameter \""+p.Name+"\" expression evaluation error: "+err.Error())
							break
						}
						intOrString := intstr.FromString(fmt.Sprintf("%v", value))
						node.Outputs.Parameters[i] = wfv1.Parameter{Name: p.Name, Value: &intOrString}
					}
					if !node.Phase.Fulfilled() {
						node = markNodeStatus(wf, node, wfv1.NodeSucceeded, "expression evaluated to true")
					}
				} else {
					continue
				}
				log.WithFields(log.Fields{"namespace": wf.Namespace, "workflow": wf.Name, "nodeId": node.ID, "phase": node.Phase, "message": node.Message}).Info("Matched event")
			}
			updated = true
		}
	}
	if updated {
		err := s.hydrator.Dehydrate(wf)
		if err != nil {
			return fmt.Errorf("failed to de-hydrate workflow: %w", err)
		}
		_, err = s.client.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(wf)
		if err != nil {
			return fmt.Errorf("failed to update workflow: %w", err)
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

func markNodeStatus(wf *wfv1.Workflow, node wfv1.NodeStatus, phase wfv1.NodePhase, message string) wfv1.NodeStatus {
	node.Phase = phase
	node.Message = message
	node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
	wf.Status.Nodes[node.ID] = node
	return node
}

func metaData(ctx context.Context) map[string]interface{} {
	meta := map[string]interface{}{
		"user": map[string]string{
			"subject": auth.GetClaims(ctx).Subject,
		},
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
