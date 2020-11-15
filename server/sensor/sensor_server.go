package sensor

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	esv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/watch"

	sensorpkg "github.com/argoproj/argo/pkg/apiclient/sensor"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/logs"
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
	listOptions := metav1.ListOptions{LabelSelector: "sensor-name"}
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
			e := &sensorpkg.LogEntry{
				Namespace:  pod.Namespace,
				SensorName: pod.Labels["sensor-name"],
				Level:      "info",
				Time:       &now,
				Msg:        string(data),
			}
			_ = json.Unmarshal(data, e)
			if in.TriggerName != "" && in.TriggerName != e.TriggerName {
				return nil
			}

			if !grep.MatchString(e.Msg) {
				return nil
			}
			return svr.Send(e)
		},
	)
}

func (e *sensorServer) WatchSensors(in *sensorpkg.ListSensorsRequest, srv sensorpkg.SensorService_WatchSensorsServer) error {
	ctx := srv.Context()
	listOptions := metav1.ListOptions{}
	if in.ListOptions != nil {
		listOptions = *in.ListOptions
	}
	eventSourceInterface := auth.GetSensorClient(ctx).ArgoprojV1alpha1().Sensors(in.Namespace)
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
			es, ok := event.Object.(*esv1.Sensor)
			if !ok {
				return apierr.FromObject(event.Object)
			}
			err := srv.Send(&sensorpkg.SensorWatchEvent{Type: string(event.Type), Object: es})
			if err != nil {
				return err
			}
		}
	}
}
func NewSensorServer() sensorpkg.SensorServiceServer {
	return &sensorServer{}
}
