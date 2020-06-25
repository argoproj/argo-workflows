package event

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/antonmedv/expr"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	eventpkg "github.com/argoproj/argo/pkg/apiclient/event"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/workflow/common"
)

type eventServer struct{}

func (e *eventServer) ReceiveEvent(ctx context.Context, event *eventpkg.Event) (*eventpkg.EventReceived, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{"event": string(data)}).Info("Received CloudEvent")
	env := make(map[string]interface{})
	err = json.Unmarshal(data, &env)
	if err != nil {
		return nil, err
	}
	wfClient := auth.GetWfClient(ctx)
	selector, _ := labels.Parse(common.LabelKeyEventWait)
	req, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.NotEquals, []string{"true"})
	selector.Add(*req)
	list, err := wfClient.ArgoprojV1alpha1().Workflows(event.Namespace).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	for _, wf := range list.Items {
		updated := false
		for _, node := range wf.Status.Nodes {
			if node.Phase == wfv1.NodeRunning && node.Type == wfv1.NodeTypeEventConsumer {
				t := wf.GetTemplateByName(node.TemplateName)
				if t == nil {
					continue
				}
				result, err := expr.Eval(t.EventConsumer.Expression, env)
				if err != nil {
					markNodeStatus(wf, node, wfv1.NodeError, "expression evaluation error: "+err.Error())
				} else {
					matches, ok := result.(bool)
					if !ok {
						markNodeStatus(wf, node, wfv1.NodeError, "expression did not evaluate to a boolean")
					} else if matches {
						markNodeStatus(wf, node, wfv1.NodeSucceeded, "")
					} else {
						continue
					}
				}
				log.WithFields(log.Fields{"namespace": wf.Namespace, "workflow": wf.Name, "nodeId": node.ID}).Info("Matched event")
				count, _ := strconv.Atoi(wf.GetLabels()[common.LabelKeyEventWait])
				if count > 1 {
					wf.GetLabels()[common.LabelKeyEventWait] = strconv.Itoa(count - 1)
				} else {
					delete(wf.GetLabels(), common.LabelKeyEventWait)
				}
				updated = true
			}
		}
		if updated {
			_, err = wfClient.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(&wf)
			if err != nil {
				return nil, err
			}
		}
	}
	return &eventpkg.EventReceived{}, nil
}

func markNodeStatus(wf wfv1.Workflow, node wfv1.NodeStatus, phase wfv1.NodePhase, message string) {
	node.Phase = phase
	node.Message = message
	node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
	wf.Status.Nodes[node.ID] = node
}

var _ eventpkg.EventServiceServer = &eventServer{}

func NewEventServer() eventpkg.EventServiceServer {
	return &eventServer{}
}
