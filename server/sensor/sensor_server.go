package sensor

import (
	"context"
	"encoding/json"

	_struct "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/argoproj/argo/pkg/apiclient/sensor"
	"github.com/argoproj/argo/server/auth"
)

type sensorServer struct {
}

func (s sensorServer) ListSensors(ctx context.Context, req *sensor.ListSensorsRequest) (*sensor.ListSensorsResponse, error) {
	if req.ListOptions == nil {
		req.ListOptions = &metav1.ListOptions{}
	}
	config, err := dynamic.NewForConfig(auth.GetRESTConfig(ctx))
	if err != nil {
		return nil, err
	}
	list, err := config.Resource(schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "sensors"}).List(*req.ListOptions)
	if err != nil {
		return nil, err
	}
	var items = make([]*_struct.Struct, len(list.Items))
	for i, item := range list.Items {
		b, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		s := &structpb.Struct{}
		err = protojson.Unmarshal(b, s)
		if err != nil {
			return nil, err
		}
		items[i] = s
	}
	return &sensor.ListSensorsResponse{
		Metadata: &metav1.ListMeta{},
		Items:    items,
	}, nil
}

func NewSensorServer() sensor.SensorServiceServer {
	return &sensorServer{}
}
