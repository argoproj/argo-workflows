package sensor

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

	sv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"

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
	streamPod := func(namespace, sensorName, podName string) error {
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
			e := &sensorpkg.LogEntry{Namespace: namespace, SensorName: sensorName, Msg: string(bytes)}
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
		err := streamPod(pod.Labels[labelSelector], pod.Namespace, pod.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewSensorServer() sensorpkg.SensorServiceServer {
	return &sensorServer{}
}
