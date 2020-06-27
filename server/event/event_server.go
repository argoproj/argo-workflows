package event

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/antonmedv/expr"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	eventpkg "github.com/argoproj/argo/pkg/apiclient/event"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/hydrator"
)

type eventServer struct {
	hydrator hydrator.Interface
}

func (s *eventServer) ReceiveEvent(ctx context.Context, req *eventpkg.EventRequest) (*eventpkg.EventResponse, error) {
	wfClient := auth.GetWfClient(ctx)
	list, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).List(listOptions())
	if err != nil {
		return nil, err
	}
	for _, wf := range list.Items {
		updated := false
		err := s.hydrator.Hydrate(&wf)
		if err != nil {
			return nil, err
		}
		mapEnv := map[string]interface{}{"event": req.Event, "workflow": wf}
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			meta := make(map[string][]string)
			for k, v := range md {
				log.Debug(k)
				switch k {
				case "X-GitHub-Event":
					meta[k] = v
				}
			}
			mapEnv["metadata"] = meta
		}
		data, err := json.Marshal(mapEnv)
		if err != nil {
			return nil, err
		}
		log.WithField("data", string(data)).Debug("Expression environment")
		env := make(map[string]interface{})
		err = json.Unmarshal(data, &env)
		if err != nil {
			return nil, err
		}
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
			err := s.hydrator.Dehydrate(&wf)
			if err != nil {
				return nil, err
			}
			_, err = wfClient.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(&wf)
			if err != nil {
				return nil, err
			}
		}
	}
	return &eventpkg.EventResponse{}, nil
}

func listOptions() metav1.ListOptions {
	req, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.NotEquals, []string{"true"})
	selector, _ := labels.Parse(common.LabelKeyEventWait)
	selector.Add(*req)
	return metav1.ListOptions{LabelSelector: selector.String()}
}

func markNodeStatus(wf wfv1.Workflow, node wfv1.NodeStatus, phase wfv1.NodePhase, message string) {
	node.Phase = phase
	node.Message = message
	node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
	wf.Status.Nodes[node.ID] = node
}

var _ eventpkg.EventServiceServer = &eventServer{}

func NewEventServer(hydrator hydrator.Interface) eventpkg.EventServiceServer {
	return &eventServer{hydrator}
}
