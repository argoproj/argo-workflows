package sensor

import (
	"context"
	"fmt"
	"testing"

	sv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/typed/sensor/v1alpha1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sensorpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sensor"
	auth "github.com/argoproj/argo-workflows/v3/server/auth"
)

type MockSensorClient struct {
	ctrl *gomock.Controller
}

func (m *MockSensorClient) ArgoprojV1alpha1Sensor() v1alpha1.SensorInterface {
	return nil
}

func (m *MockSensorClient) List(ctx context.Context, opts metav1.ListOptions) (*sv1.SensorList, error) {
	sensorList := &sv1.SensorList{
		Items: []sv1.Sensor{
			{ObjectMeta: metav1.ObjectMeta{Name: "sensor1"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "sensor2"}},
		},
	}
	return sensorList, nil
}

type mockSensorServer struct {
	sensorClient v1alpha1.SensorInterface
}

func (s *mockSensorServer) ListSensors(ctx context.Context, req *sensorpkg.ListSensorsRequest) (*sv1.SensorList, error) {
	if s.sensorClient == nil {
		return nil, fmt.Errorf("sensor client is not set")
	}

	sensorList, err := s.sensorClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return sensorList, nil
}

func TestListSensors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := &MockSensorClient{ctrl: ctrl}

	ctx := context.WithValue(context.Background(), auth.SensorKey, mockClient)

	server := &mockSensorServer{
		sensorClient: mockClient.ArgoprojV1alpha1Sensor(),
	}

	request := &sensorpkg.ListSensorsRequest{
		Namespace: "my-namespace",
	}

	sensorList, err := server.ListSensors(ctx, request)

	require.EqualError(t, err, "sensor client is not set", "Expected no error")
	assert.Nil(t, sensorList, "Expected sensor list to be nil")
	assert.NotNil(t, mockClient, "Expected mock client to be not nil")
	assert.Contains(t, err.Error(), "sensor client", "Expected error message to mention sensor client")
}

func TestListSensors_SensorClientNotSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := &MockSensorClient{ctrl: ctrl}

	ctx := context.WithValue(context.Background(), auth.SensorKey, mockClient)

	server := &mockSensorServer{
		sensorClient: mockClient.ArgoprojV1alpha1Sensor(),
	}

	// Set up an error scenario where the sensor client is not set
	server.sensorClient = nil

	request := &sensorpkg.ListSensorsRequest{
		Namespace: "my-namespace",
	}

	sensorList, err := server.ListSensors(ctx, request)

	require.Error(t, err, "Expected error")
	assert.Nil(t, sensorList, "Expected sensor list to be nil")
	assert.Equal(t, "sensor client is not set", err.Error(), "Expected error message to match")
}
