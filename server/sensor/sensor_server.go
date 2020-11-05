package sensor

import (
	"bufio"
	"context"
	"sync"

	sv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/watch"

	sensorpkg "github.com/argoproj/argo/pkg/apiclient/sensor"
	"github.com/argoproj/argo/server/auth"
)

type sensorServer struct{}

func (s *sensorServer) ListSensors(ctx context.Context, in *sensorpkg.ListSensorsRequest) (*sv1.SensorList, error) {
	client := auth.GetSensorClient(ctx)
	list, err := client.ArgoprojV1alpha1().Sensors(in.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (s *sensorServer) SensorsLogs(in *sensorpkg.SensorsLogsRequest, svr sensorpkg.SensorService_SensorsLogsServer) error {
	labelSelector := "sensor-name"
	podsInterface := auth.GetKubeClient(svr.Context()).CoreV1().Pods(in.Namespace)
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	podLogOptions := in.PodLogOptions
	if podLogOptions == nil {
		podLogOptions = &corev1.PodLogOptions{}
	}
	list, err := podsInterface.List(listOptions)
	if err != nil {
		return err
	}
	streaming := &sync.Map{}
	streamPod := func(sensorName, podName string) error {
		log.With("sensorName", sensorName).With("podName", podName).Debug("streaming pod logs")
		_, loaded := streaming.LoadOrStore(podName, true)
		if loaded {
			return nil
		}
		defer streaming.Delete(podName)
		stream, err := podsInterface.GetLogs(podName, podLogOptions).Stream()
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			text := scanner.Text()
			err := svr.Send(&sensorpkg.LogEntry{SensorName: sensorName, Content: text})
			if err != nil {
				return err
			}
		}
		return nil
	}
	for _, p := range list.Items {
		err := streamPod(p.Labels[labelSelector], p.Name)
		if err != nil {
			return err
		}
	}
	watcher, err := watch.NewRetryWatcher(list.ResourceVersion, podsInterface)
	if err != nil {
		return err
	}
	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			return apierr.FromObject(event.Object)
		}
		err := streamPod(pod.Labels[labelSelector], pod.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewSensorServer() sensorpkg.SensorServiceServer {
	return &sensorServer{}
}
