package sensor

import (
	"github.com/argoproj/pkg/grpc/http"
)

func init() {
	forward_SensorService_WatchSensorsLogs_0 = http.StreamForwarder
	forward_SensorService_WatchSensors_0 = http.StreamForwarder
}
