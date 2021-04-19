package v1alpha1

import v1 "k8s.io/api/core/v1"

type HTTP struct {
	Headers []HTTPHeader `json:"headers" protobuf:"bytes,1,opt,name=headers"`
	Method  string       `json:"method" protobuf:"bytes,2,opt,name=method"`
	URL     string       `json:"url" protobuf:"bytes,3,opt,name=url"`
	Data    string       `json:"data" protobuf:"bytes,4,opt,name=data"`
}

type HTTPHeader struct {
	Name          string                  `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value         string                  `json:"value" protobuf:"bytes,2,opt,name=value"`
	FromSecrete   v1.SecretKeySelector    `json:"fromSecrete" protobuf:"bytes,3,opt,name=fromSecrete"`
	FromConfigMap v1.ConfigMapKeySelector `json:"fromConfigMap" protobuf:"bytes,4,opt,name=fromConfigMap"`
}
