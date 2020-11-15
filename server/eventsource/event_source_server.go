package eventsource

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	esv1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/watch"

	eventsourcepkg "github.com/argoproj/argo/pkg/apiclient/eventsource"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/logs"
)

type eventSourceServer struct{}

func (e *eventSourceServer) ListEventSources(ctx context.Context, in *eventsourcepkg.ListEventSourcesRequest) (*esv1.EventSourceList, error) {
	client := auth.GetEventSourceClient(ctx)
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
	grep, err := regexp.Compile(in.Grep)
	if err != nil {
		return err
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
			if !grep.MatchString(e.Msg) {
				return nil
			}
			return svr.Send(e)
		},
	)
}

func (e *eventSourceServer) WatchEventSources(in *eventsourcepkg.ListEventSourcesRequest, srv eventsourcepkg.EventSourceService_WatchEventSourcesServer) error {
	ctx := srv.Context()
	listOptions := metav1.ListOptions{}
	if in.ListOptions != nil {
		listOptions = *in.ListOptions
	}
	eventSourceInterface := auth.GetEventSourceClient(ctx).ArgoprojV1alpha1().EventSources(in.Namespace)
	watcher, err := watch.NewRetryWatcher(listOptions.ResourceVersion, eventSourceInterface)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("failed to read event")
			}
			es, ok := event.Object.(*esv1.EventSource)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			err := srv.Send(&eventsourcepkg.EventSourceWatchEvent{Type: string(event.Type), Object: es})
			if err != nil {
				return err
			}
		}
	}
}

func NewEventSourceServer() eventsourcepkg.EventSourceServiceServer {
	return &eventSourceServer{}
}
