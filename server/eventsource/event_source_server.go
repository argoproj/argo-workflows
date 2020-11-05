package eventsource

import (
	"bufio"
	"context"
	"encoding/json"
	"sync"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/watch"

	esv1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"

	eventsourcepkg "github.com/argoproj/argo/pkg/apiclient/eventsource"
	"github.com/argoproj/argo/server/auth"
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
	labelSelector := "eventsource-name"
	coreV1 := auth.GetKubeClient(svr.Context()).CoreV1()
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	podLogOptions := in.PodLogOptions
	if podLogOptions == nil {
		podLogOptions = &corev1.PodLogOptions{}
	}
	list, err := coreV1.Pods(in.Namespace).List(listOptions)
	if err != nil {
		return err
	}
	streaming := &sync.Map{}
	streamPod := func(namespace, eventSourceName, podName string) error {
		log.WithFields(log.Fields{"namespace": namespace, "podName": podName}).Debug("streaming pod logs")
		_, loaded := streaming.LoadOrStore(podName, true)
		if loaded {
			return nil
		}
		defer streaming.Delete(podName)
		stream, err := coreV1.Pods(namespace).GetLogs(podName, podLogOptions).Stream()
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			bytes := scanner.Bytes()
			e := &eventsourcepkg.LogEntry{Namespace: namespace, EventSourceName: eventSourceName, Msg: string(bytes)}
			_ = json.Unmarshal(bytes, e)
			err = svr.Send(e)
			if err != nil {
				return err
			}
		}
		return nil
	}
	for _, p := range list.Items {
		err := streamPod(p.Namespace, p.Labels[labelSelector], p.Name)
		if err != nil {
			return err
		}
	}
	watcher, err := watch.NewRetryWatcher(list.ResourceVersion, coreV1.Pods(in.Namespace))
	if err != nil {
		return err
	}
	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			return apierr.FromObject(event.Object)
		}
		err := streamPod(pod.Namespace, pod.Labels[labelSelector], pod.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewEventSourceServer() eventsourcepkg.EventSourceServiceServer {
	return &eventSourceServer{}
}
