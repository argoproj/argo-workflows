package sensor

import (
	"context"
	"encoding/json"

	"github.com/gogo/protobuf/types"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/argoproj/argo/pkg/apiclient/sensor"
	"github.com/argoproj/argo/server/auth"
)

type sensorServer struct {
}

func (s sensorServer) ListSensors(ctx context.Context, req *sensor.ListSensorsRequest) (*sensor.ListSensorsResponse, error) {
	resourceIf, err := resourceInterface(ctx, req.Namespace)
	if err != nil {
		return nil, err
	}
	list, err := resourceIf.List(listOptions(req))
	if err != nil {
		return nil, err
	}
	items, err := unToStruct(list)
	if err != nil {
		return nil, err
	}
	return &sensor.ListSensorsResponse{Metadata: &metav1.ListMeta{}, Items: items}, nil
}

func resourceInterface(ctx context.Context, namespace string) (dynamic.ResourceInterface, error) {
	config, err := dynamic.NewForConfig(auth.GetRESTConfig(ctx))
	if err != nil {
		return nil, err
	}
	versionResource := schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "sensors"}
	return config.Resource(versionResource).Namespace(namespace), nil
}

func listOptions(req *sensor.ListSensorsRequest) metav1.ListOptions {
	listOptions := metav1.ListOptions{}
	if req.ListOptions != nil {
		listOptions = *req.ListOptions
	}
	return listOptions
}

func unToStruct(list *unstructured.UnstructuredList) ([]*types.Struct, error) {
	var items = make([]*types.Struct, len(list.Items))
	log.WithField("items", list.Items).Debug()
	for i, item := range list.Items {
		b, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		s := &types.Struct{}
		err = json.Unmarshal(b, s)
		if err != nil {
			return nil, err
		}
		items[i] = s
	}
	log.WithField("items", items).Debug()
	return items, nil
}

func NewSensorServer() sensor.SensorServiceServer {
	return &sensorServer{}
}
