package eventsource

import (
	"context"
	"encoding/json"

	esv1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventsourcepkg "github.com/argoproj/argo/pkg/apiclient/eventsource"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/logs"
)

type eventSourceServer struct{}

func (e *eventSourceServer) ListEventSources(ctx context.Context, in *eventsourcepkg.ListEventSourcesRequest) (*esv1.EventSourceList, error) {
	client := auth.GetEvenSourceClient(ctx)
	list, err := client.ArgoprojV1alpha1().EventSources(in.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (e *eventSourceServer) EventSourcesLogs(in *eventsourcepkg.EventSourcesLogsRequest, svr eventsourcepkg.EventSourceService_EventSourcesLogsServer) error {
	listOptions := metav1.ListOptions{LabelSelector: "eventsource-name"}
	if in.Name != "" {
		listOptions.LabelSelector += "=" + in.Name
	}
	return logs.LogPods(
		svr.Context(),
		in.Namespace,
		listOptions,
		in.PodLogOptions,
		func(pod *corev1.Pod, data []byte) error {
			now := metav1.Now()
			e := &eventsourcepkg.LogEntry{
				Namespace:       pod.Namespace,
				EventSourceName: pod.Labels["eventsource-name"],
				Level:           "info",
				Time:            &now,
				Msg:             string(data),
			}
			_ = json.Unmarshal(data, e)
			if in.EventSourceType != "" && in.EventSourceType != e.EventSourceType {
				return nil
			}
			if in.EventName != "" && in.EventName != e.EventName {
				return nil
			}
			return svr.Send(e)
		},
	)
}

func NewEventSourceServer() eventsourcepkg.EventSourceServiceServer {
	return &eventSourceServer{}
}
