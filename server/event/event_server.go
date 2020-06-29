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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/utils/pointer"

	eventpkg "github.com/argoproj/argo/pkg/apiclient/event"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/hydrator"
	"github.com/argoproj/argo/workflow/suspend"
)

type eventServer struct {
	hydrator hydrator.Interface
	messages chan message
}

type message struct {
	wfClient  versioned.Interface
	hydrator  hydrator.Interface
	namespace string
	event     *wfv1.Item
	metadata  map[string][]string
}

func (s message) Execute() {
	s.resumeWorkflows()
	s.createWorkflowsFromLabelledTemplates()
}

func (s *eventServer) Run(stopCh <-chan struct{}) {
	for {
		select {
		case message := <-s.messages:
			message.Execute()
		case <-stopCh:
			return
		}
	}
}

func (s *eventServer) ReceiveEvent(ctx context.Context, req *eventpkg.EventRequest) (*eventpkg.EventResponse, error) {
	s.messages <- message{auth.GetWfClient(ctx), s.hydrator, req.Namespace, req.Event, metaData(ctx)}
	return &eventpkg.EventResponse{}, nil
}

func (s *message) resumeWorkflows() {
	workflowList, err := s.wfClient.ArgoprojV1alpha1().Workflows(s.namespace).List(listOptions())
	if err != nil {
		log.WithError(err).Error("failed to list workflows")
		return
	}
	for _, wf := range workflowList.Items {
		err := s.resumeWorkflow(wf)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{"namespace": wf.Namespace, "workflow": wf.Name}).Error("failed to resume workflow")
		}
	}
}

func (s *message) resumeWorkflow(wf wfv1.Workflow) error {
	err := s.hydrator.Hydrate(&wf)
	if err != nil {
		return fmt.Errorf("failed to hydrate workflow: %v", err)
	}
	updated := false
	for _, node := range wf.Status.Nodes {
		if !node.Phase.Fulfilled() && node.Type == wfv1.NodeTypeSuspend {
			env, err := expressionEnvironment(map[string]interface{}{"event": s.event, "inputs": node.Inputs, "metadata": s.metadata})
			if err != nil {
				return errors.New("failed to create expression environment - should never happen")
			}
			t := wf.GetTemplateByName(node.TemplateName)
			if t == nil {
				return errors.New("malformed workflow: template is nil - should never happen")
			}
			if t.Suspend == nil {
				return errors.New("malformed workflow: template suspend field is nil - should never happen")
			}
			if t.Suspend.Event == nil {
				continue
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
						if p.Value == nil {
							node = markNodeStatus(wf, node, wfv1.NodeError, " malformed output parameter \""+p.Name+"\": value nil")
							break
						}
						value, err := expr.Eval(*p.Value, env)
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
				suspend.DecrementEventWaitCount(&wf)
			}
			updated = true
		}
	}
	if updated {
		// TODO - we need a way to share code that applies updates with retry and backoff
		err := s.hydrator.Dehydrate(&wf)
		if err != nil {
			return fmt.Errorf("failed to de-hydrate workflow: %v", err)
		}
		_, err = s.wfClient.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(&wf)
		if err != nil {
			return fmt.Errorf("failed to update workflow: %v", err)
		}
	}
	return nil
}

func (s *message) createWorkflowsFromLabelledTemplates() {
	templateList, err := s.wfClient.ArgoprojV1alpha1().WorkflowTemplates(s.namespace).List(metav1.ListOptions{LabelSelector: common.LabelKeyEvent})
	if err != nil {
		log.WithError(err).Error("failed to list workflows")
		return
	}
	for _, tmpl := range templateList.Items {
		err := s.createWorkflowFromTemplate(tmpl)
		log.WithError(err).WithFields(log.Fields{"namespace": tmpl.Namespace, "template": tmpl.Name}).Error("failed to create template")
	}
}

func (s *message) createWorkflowFromTemplate(tmpl wfv1.WorkflowTemplate) error {
	if tmpl.Spec.Event == nil {
		return errors.New("malformed template: event spec is missing")
	}
	env, err := expressionEnvironment(map[string]interface{}{"event": s.event, "template": tmpl, "metadata": s.metadata})
	if err != nil {
		return fmt.Errorf("failed to create template expression environment - should never happen: %v", err)
	}
	result, err := expr.Eval(tmpl.Spec.Event.Expression, env)
	if err != nil {
		return errors.New("failed to evaluate template expression")
	}
	matched, ok := result.(bool)
	if !ok {
		return errors.New("malformed template expression: did not evaluate to boolean")
	} else if matched {
		wf := common.NewWorkflowFromWorkflowTemplate(tmpl.Name, false)
		_, err := s.wfClient.ArgoprojV1alpha1().Workflows(tmpl.Namespace).Create(wf)
		if err != nil {
			return fmt.Errorf("failed to create workflow from template: %v", err)
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

func listOptions() metav1.ListOptions {
	req, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.NotEquals, []string{"true"})
	selector, _ := labels.Parse(common.LabelKeyEventWaitCount)
	selector.Add(*req)
	return metav1.ListOptions{LabelSelector: selector.String()}
}

func markNodeStatus(wf wfv1.Workflow, node wfv1.NodeStatus, phase wfv1.NodePhase, message string) wfv1.NodeStatus {
	node.Phase = phase
	node.Message = message
	node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
	wf.Status.Nodes[node.ID] = node
	return node
}

var _ eventpkg.EventServiceServer = &eventServer{}

func NewEventServer(hydrator hydrator.Interface) eventpkg.EventServiceServer {
	return &eventServer{hydrator, make(chan message, 64)}
}
