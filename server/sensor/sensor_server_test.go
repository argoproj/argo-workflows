package sensor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/pkg/apiclient/sensor"
	"github.com/argoproj/argo/server/auth"
)

func Test_sensorServer_ListSensors(t *testing.T) {
	server := NewSensorServer()

	ctx := context.WithValue(context.Background(), auth.RESTConfigKey, &rest.Config{})

	sensors, err := server.ListSensors(ctx, &sensor.ListSensorsRequest{})

	if assert.NoError(t, err) {
		assert.Len(t, sensors, 1)
	}
}
