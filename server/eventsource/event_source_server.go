package eventsource

import (
	"context"
	"encoding/json"
	"io"

	eventsv1a1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventsourcepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/eventsource"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	"github.com/argoproj/argo-workflows/v4/util/logs"

	sutils "github.com/argoproj/argo-workflows/v4/server/utils"
)

type eventSourceServer struct{}

func (e *eventSourceServer) CreateEventSource(ctx context.Context, in *eventsourcepkg.CreateEventSourceRequest) (*eventsv1a1.EventSource, error) {
	client := auth.GetEventsClient(ctx)

	es, err := client.ArgoprojV1alpha1().EventSources(in.Namespace).Create(ctx, in.EventSource, metav1.CreateOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return es, nil
}

func (e *eventSourceServer) GetEventSource(ctx context.Context, in *eventsourcepkg.GetEventSourceRequest) (*eventsv1a1.EventSource, error) {
	client := auth.GetEventsClient(ctx)

	es, err := client.ArgoprojV1alpha1().EventSources(in.Namespace).Get(ctx, in.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return es, nil
}

func (e *eventSourceServer) DeleteEventSource(ctx context.Context, in *eventsourcepkg.DeleteEventSourceRequest) (*eventsourcepkg.EventSourceDeletedResponse, error) {
	client := auth.GetEventsClient(ctx)
	err := client.ArgoprojV1alpha1().EventSources(in.Namespace).Delete(ctx, in.Name, metav1.DeleteOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return &eventsourcepkg.EventSourceDeletedResponse{}, nil
}

func (e *eventSourceServer) UpdateEventSource(ctx context.Context, in *eventsourcepkg.UpdateEventSourceRequest) (*eventsv1a1.EventSource, error) {
	client := auth.GetEventsClient(ctx)
	es, err := client.ArgoprojV1alpha1().EventSources(in.Namespace).Update(ctx, in.EventSource, metav1.UpdateOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return es, nil
}

func (e *eventSourceServer) ListEventSources(ctx context.Context, in *eventsourcepkg.ListEventSourcesRequest) (*eventsv1a1.EventSourceList, error) {
	client := auth.GetEventsClient(ctx)
	list, err := client.ArgoprojV1alpha1().EventSources(in.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return list, nil
}

func (e *eventSourceServer) EventSourcesLogs(in *eventsourcepkg.EventSourcesLogsRequest, svr eventsourcepkg.EventSourceService_EventSourcesLogsServer) error {
	labelSelector := "eventsource-name"
	if in.Name != "" {
		labelSelector += "=" + in.Name
	}
	ctx := svr.Context()
	err := logs.LogPods(
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
			return sutils.ToStatusError(svr.Send(e), codes.Internal)
		},
	)
	return sutils.ToStatusError(err, codes.Internal)
}

func (e *eventSourceServer) WatchEventSources(in *eventsourcepkg.ListEventSourcesRequest, srv eventsourcepkg.EventSourceService_WatchEventSourcesServer) error {
	ctx := srv.Context()
	listOptions := metav1.ListOptions{}
	if in.ListOptions != nil {
		listOptions = *in.ListOptions
	}
	eventSourceInterface := auth.GetEventsClient(ctx).ArgoprojV1alpha1().EventSources(in.Namespace)
	watcher, err := eventSourceInterface.Watch(ctx, listOptions)
	if err != nil {
		return sutils.ToStatusError(err, codes.Internal)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, open := <-watcher.ResultChan():
			if !open {
				return sutils.ToStatusError(io.EOF, codes.ResourceExhausted)
			}
			es, ok := event.Object.(*eventsv1a1.EventSource)
			if !ok {
				return sutils.ToStatusError(apierr.FromObject(event.Object), codes.Internal)
			}
			err := srv.Send(&eventsourcepkg.EventSourceWatchEvent{Type: string(event.Type), Object: es})
			if err != nil {
				return sutils.ToStatusError(err, codes.Internal)
			}
		}
	}
}

func NewEventSourceServer() eventsourcepkg.EventSourceServiceServer {
	return &eventSourceServer{}
}
