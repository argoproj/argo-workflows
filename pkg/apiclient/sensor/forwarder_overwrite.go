package sensor

import (
	"github.com/argoproj/argo-workflows/v4/util/grpc/gateway"
)

func init() {
	forward_SensorService_SensorsLogs_0 = gateway.StreamForwarder
	forward_SensorService_WatchSensors_0 = gateway.StreamForwarder
}
