package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/antonmedv/expr"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/hydrator"
	"github.com/argoproj/argo/workflow/util"
)

type operation struct {
	// system context
	client    versioned.Interface
	hydrator  hydrator.Interface
	workflows map[id]bool
	templates map[id]bool
	// about the event
	namespace string
	event     *wfv1.Item
	metadata  map[string][]string
}

func (s *operation) Execute() {
	s.resumeWorkflows()
	s.submitWorkflows()
}

func (s *operation) resumeWorkflows() {
	for wf := range s.workflows {
		err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
			err := s.resumeWorkflow(wf.namespace, wf.name)
			return err == nil, err
		})
		if err != nil {
			log.WithFields(log.Fields{"namespace": wf.namespace, "workflow": wf.name}).WithError(err).Error("failed to resume workflow")
		}
	}
}

func (s *operation) resumeWorkflow(namespace, name string) error {
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
				return errors.New("malformed workflow: template,  suspend, or event is nil - should never happen")
			}
			env, err := expressionEnvironment(map[string]interface{}{"event": s.event, "inputs": node.Inputs, "metadata": s.metadata})
			if err != nil {
				return errors.New("failed to create expression environment - should never happen")
			}
			result, err := expr.Eval(t.Suspend.Event.Expression, env)
			if err != nil {
				// this is a condition, because it is possible for events to fail expression, but it not really be a problem with the expression
				wf.Status.Conditions.UpsertCondition(wfv1.Condition{Status: metav1.ConditionTrue, Type: wfv1.ConditionTypeEventExpressionError, Message: err.Error()})
			} else {
				matches, ok := result.(bool)
				if !ok {
					node = markNodeStatus(wf, node, wfv1.NodeError, "malformed expression: did not evaluate to a boolean: "+reflect.TypeOf(result).Name())
				} else if matches {
					node.Outputs = &wfv1.Outputs{Parameters: make([]wfv1.Parameter, len(t.Outputs.Parameters))}
					for i, p := range t.Outputs.Parameters {
						if p.ValueFrom == nil {
							node = markNodeStatus(wf, node, wfv1.NodeError, " malformed output parameter \""+p.Name+"\": valueFrom nil")
							break
						}
						value, err := expr.Eval(p.ValueFrom.Expression, env)
						if err != nil {
							node = markNodeStatus(wf, node, wfv1.NodeError, "output parameter \""+p.Name+"\" expression evaluation error: "+err.Error())
							break
						}
						node.Outputs.Parameters[i] = wfv1.Parameter{Name: p.Name, Value: pointer.StringPtr(fmt.Sprintf("%v", value))}
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

func (s *operation) submitWorkflows() {
	for tmpl := range s.templates {
		err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
			err := s.submitWorkflowFromTemplate(tmpl.namespace, tmpl.name)
			return err == nil, err
		})
		log.WithError(err).WithFields(log.Fields{"namespace": tmpl.namespace, "template": tmpl.name}).Error("failed to submit workflow from template")
	}
}

func (s *operation) submitWorkflowFromTemplate(namespace, name string) error {
	tmpl, err := s.client.ArgoprojV1alpha1().WorkflowTemplates(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get workflow template: %w", err)
	}
	if tmpl.Spec.Event == nil {
		return errors.New("malformed template: event spec is missing - should never happen")
	}
	env, err := expressionEnvironment(map[string]interface{}{"event": s.event, "template": tmpl, "metadata": s.metadata})
	if err != nil {
		return fmt.Errorf("failed to create template expression environment - should never happen: %w", err)
	}
	result, err := expr.Eval(tmpl.Spec.Event.Expression, env)
	if err != nil {
		return errors.New("failed to evaluate template expression")
	}
	matched, ok := result.(bool)
	if !ok {
		return errors.New("malformed template expression: did not evaluate to boolean")
	} else if matched {
		parameters := make([]string, len(tmpl.Spec.Arguments.Parameters))
		for i, p := range tmpl.Spec.Arguments.Parameters {
			if p.ValueFrom == nil {
				return fmt.Errorf("malformed workflow templates: parameter \"%s\" valueFrom is nil", p.Name)
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
			return fmt.Errorf("failed to apply submit options to workflow template: %w", err)
		}
		_, err = s.client.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
		if err != nil {
			return fmt.Errorf("failed to create workflow from template: %w", err)
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

func metaData(ctx context.Context) map[string][]string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}
	meta := make(map[string][]string)
	for k, v := range md {
		if strings.HasPrefix(k, "x-") {
			meta[k] = v
		}
	}
	return meta
}
