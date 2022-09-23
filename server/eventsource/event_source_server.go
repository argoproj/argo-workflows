package eventsource

import (
	"context"
	"encoding/json"
	"io"

	esv1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventsourcepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/eventsource"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/util/logs"
)

type eventSourceServer struct{}

func (e *eventSourceServer) CreateEventSource(ctx context.Context, in *eventsourcepkg.CreateEventSourceRequest) (*esv1.EventSource, error) {
	client := auth.GetEventSourceClient(ctx)

	es, err := client.ArgoprojV1alpha1().EventSources(in.Namespace).Create(ctx, in.EventSource, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return es, nil
}

func (e *eventSourceServer) GetEventSource(ctx context.Context, in *eventsourcepkg.GetEventSourceRequest) (*esv1.EventSource, error) {
	client := auth.GetEventSourceClient(ctx)

	es, err := client.ArgoprojV1alpha1().EventSources(in.Namespace).Get(ctx, in.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return es, nil
}

func (e *eventSourceServer) DeleteEventSource(ctx context.Context, in *eventsourcepkg.DeleteEventSourceRequest) (*eventsourcepkg.EventSourceDeletedResponse, error) {
	client := auth.GetEventSourceClient(ctx)
	err := client.ArgoprojV1alpha1().EventSources(in.Namespace).Delete(ctx, in.Name, metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}
	return &eventsourcepkg.EventSourceDeletedResponse{}, nil
}

func (e *eventSourceServer) UpdateEventSource(ctx context.Context, in *eventsourcepkg.UpdateEventSourceRequest) (*esv1.EventSource, error) {
	client := auth.GetEventSourceClient(ctx)
	es, err := client.ArgoprojV1alpha1().EventSources(in.Namespace).Update(ctx, in.EventSource, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return es, nil
}

func (e *eventSourceServer) ListEventSources(ctx context.Context, in *eventsourcepkg.ListEventSourcesRequest) (*esv1.EventSourceList, error) {
	client := auth.GetEventSourceClient(ctx)
	list, err := client.ArgoprojV1alpha1().EventSources(in.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (e *eventSourceServer) EventSourcesLogs(in *eventsourcepkg.EventSourcesLogsRequest, svr eventsourcepkg.EventSourceService_EventSourcesLogsServer) error {
	labelSelector := "eventsource-name"
	if in.Name != "" {
		labelSelector += "=" + in.Name
	}
	ctx := svr.Context()
	return logs.LogPods(
		ctx,
		auth.GetKubeClient(ctx),
		in.Namespace,
		labelSelector,
		in.Grep,
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

func (e *eventSourceServer) WatchEventSources(in *eventsourcepkg.ListEventSourcesRequest, srv eventsourcepkg.EventSourceService_WatchEventSourcesServer) error {
	ctx := srv.Context()
	listOptions := metav1.ListOptions{}
	if in.ListOptions != nil {
		listOptions = *in.ListOptions
	}
	eventSourceInterface := auth.GetEventSourceClient(ctx).ArgoprojV1alpha1().EventSources(in.Namespace)
	watcher, err := eventSourceInterface.Watch(ctx, listOptions)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, open := <-watcher.ResultChan():
			if !open {
				return io.EOF
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
