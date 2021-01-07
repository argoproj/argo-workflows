package sensor

import (
	"context"
	"encoding/json"
	"io"
	"regexp"

	sv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

func (s *sensorServer) GetSensor(ctx context.Context, in *sensorpkg.GetSensorRequest) (*sv1.Sensor, error) {
	client := auth.GetSensorClient(ctx)
	return client.ArgoprojV1alpha1().Sensors(in.Namespace).Get(in.Name, metav1.GetOptions{})
}

func (s *sensorServer) CreateSensor(ctx context.Context, in *sensorpkg.CreateSensorRequest) (*sv1.Sensor, error) {
	client := auth.GetSensorClient(ctx)
	return client.ArgoprojV1alpha1().Sensors(in.Namespace).Create(in.Sensor)
}

func (s *sensorServer) UpdateSensor(ctx context.Context, in *sensorpkg.UpdateSensorRequest) (*sv1.Sensor, error) {
	client := auth.GetSensorClient(ctx)
	return client.ArgoprojV1alpha1().Sensors(in.Namespace).Update(in.Sensor)
}

func (s *sensorServer) DeleteSensor(ctx context.Context, in *sensorpkg.DeleteSensorRequest) (*sensorpkg.DeleteSensorResponse, error) {
	client := auth.GetSensorClient(ctx)
	if err := client.ArgoprojV1alpha1().Sensors(in.Namespace).Delete(in.Name, &metav1.DeleteOptions{}); err != nil {
		return nil, err
	}
	return &sensorpkg.DeleteSensorResponse{}, nil
}

func (s *sensorServer) SensorsLogs(in *sensorpkg.SensorsLogsRequest, svr sensorpkg.SensorService_SensorsLogsServer) error {
	labelSelector := "sensor-name"
	if in.Name != "" {
		labelSelector += "=" + in.Name
	}
	grep, err := regexp.Compile(in.Grep)
	if err != nil {
		return err
	}
	return logs.LogPods(
		svr.Context(),
		in.Namespace,
		labelSelector,
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

func (s *sensorServer) WatchSensors(in *sensorpkg.ListSensorsRequest, srv sensorpkg.SensorService_WatchSensorsServer) error {
	ctx := srv.Context()
	listOptions := metav1.ListOptions{}
	if in.ListOptions != nil {
		listOptions = *in.ListOptions
	}
	eventSourceInterface := auth.GetSensorClient(ctx).ArgoprojV1alpha1().Sensors(in.Namespace)
	watcher, err := eventSourceInterface.Watch(listOptions)
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
			es, ok := event.Object.(*sv1.Sensor)
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

// NewSensorServer returns a new sensorServer instance
func NewSensorServer() sensorpkg.SensorServiceServer {
	return &sensorServer{}
}
