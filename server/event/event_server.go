package event

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
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
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/hydrator"
	"github.com/argoproj/argo/workflow/suspend"
)

type eventServer struct {
	hydrator hydrator.Interface
}

func (s *eventServer) ReceiveEvent(ctx context.Context, req *eventpkg.EventRequest) (*eventpkg.EventResponse, error) {
	wfClient := auth.GetWfClient(ctx)
	workflowList, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).List(listOptions())
	if err != nil {
		return nil, err
	}
	templateList, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).List(metav1.ListOptions{LabelSelector: common.LabelKeyEvent})
	if err != nil {
		return nil, err
	}
	s.resumeSuspendedWorkflows(ctx, workflowList.Items, req.Event)
	s.createWorkflowsFromLabelledTemplates(ctx, templateList.Items, req.Event)
	return &eventpkg.EventResponse{}, nil
}

func (s *eventServer) resumeSuspendedWorkflows(ctx context.Context, workflows []wfv1.Workflow, event *wfv1.Item) {
	wfClient := auth.GetWfClient(ctx)
	for _, wf := range workflows {
		logCtx := log.WithField("namespace", wf.Namespace).WithField("workflow", wf.Name)
		updated := false
		err := s.hydrator.Hydrate(&wf)
		if err != nil {
			logCtx.WithError(err).Error("failed to hydrate workflow")
			continue
		}
		for _, node := range wf.Status.Nodes {
			if !node.Phase.Fulfilled() && node.Type == wfv1.NodeTypeSuspend {
				env, err := expressionEnvironment(map[string]interface{}{"event": event, "workflow": wf, "inputs": node.Inputs, "metadata": metaData(ctx)})
				if err != nil {
					logCtx.WithError(err).Error("failed to create expression environment - should never happen")
					continue
				}
				t := wf.GetTemplateByName(node.TemplateName)
				if t == nil {
					logCtx.Error("malformed workflow: template is nil - should never happen")
					continue
				}
				if t.Suspend == nil {
					logCtx.Error("malformed workflow: template suspend field is nil - should never happen")
					continue
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
					suspend.DecrementEventWait(&wf)
				}
				updated = true
			}
		}
		if updated {
			// TODO - we need a way to share code that applies updates with retry and backoff
			err := s.hydrator.Dehydrate(&wf)
			if err != nil {
				logCtx.WithError(err).Error("failed to de-hydrate workflow")
				continue
			}
			_, err = wfClient.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(&wf)
			if err != nil {
				logCtx.WithError(err).Error("failed to update workflow")
				continue
			}
		}
	}
}

func (s *eventServer) createWorkflowsFromLabelledTemplates(ctx context.Context, templates []wfv1.WorkflowTemplate, event *wfv1.Item) {
	wfClient := auth.GetWfClient(ctx)
	for _, tmpl := range templates {
		logCtx := log.WithField("namespace", tmpl.Namespace).WithField("template", tmpl)
		if tmpl.Spec.Event == nil {
			logCtx.Error("malformed template: event spec is missing")
			continue
		}
		env, err := expressionEnvironment(map[string]interface{}{"event": event, "template": tmpl, "metadata": metaData(ctx)})
		if err != nil {
			logCtx.WithError(err).Error("failed to create template expression environment - should never happen")
			continue
		}
		result, err := expr.Eval(tmpl.Spec.Event.Expression, env)
		if err != nil {
			logCtx.WithError(err).Warn("failed to evaluate template expression")
			continue
		}
		matched, ok := result.(bool)
		if !ok {
			logCtx.WithField("type", reflect.TypeOf(result).Name()).Error("malformed template expression: did not evaluate to boolean")
		} else if matched {
			wf := common.NewWorkflowFromWorkflowTemplate(tmpl.Name, false)
			_, err := wfClient.ArgoprojV1alpha1().Workflows(tmpl.Namespace).Create(wf)
			if err != nil {
				logCtx.WithError(err).Error("failed to create workflow from template")
				continue
			}
		}
	}
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
	log.Debug(md)
	meta := make(map[string][]string)
	for k, v := range md {
		switch k {
		case "X-GitHub-Event":
			meta[k] = v
		}
	}
	return meta
}

func listOptions() metav1.ListOptions {
	req, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.NotEquals, []string{"true"})
	selector, _ := labels.Parse(common.LabelKeyEventWait)
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
	return &eventServer{hydrator}
}
