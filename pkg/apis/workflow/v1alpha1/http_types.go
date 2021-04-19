package v1alpha1

import v1 "k8s.io/api/core/v1"

type HTTP struct {
	// Headers holds the http headers
	Headers []HTTPHeader `json:"headers" protobuf:"bytes,1,opt,name=headers"`
	// Method is http methods (POST, GET, UPDATE)
	Method string `json:"method" protobuf:"bytes,2,opt,name=method"`
	// URL is a invoke URL
	URL string `json:"url" protobuf:"bytes,3,opt,name=url"`
	// Data is a body of http request
	Data string `json:"data" protobuf:"bytes,4,opt,name=data"`
}

type HTTPHeader struct {
	// Name is the name of the hearder
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Value is the value of the header value
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
	// FromSecrete is the secret selector to the header value
	FromSecrete v1.SecretKeySelector `json:"fromSecrete" protobuf:"bytes,3,opt,name=fromSecrete"`
	// FromConfigMap is the configmap selector to the header value
	FromConfigMap v1.ConfigMapKeySelector `json:"fromConfigMap" protobuf:"bytes,4,opt,name=fromConfigMap"`
}
