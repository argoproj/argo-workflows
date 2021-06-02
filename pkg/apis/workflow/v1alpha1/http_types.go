package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
)

type HTTPHeaderSource struct {
	SecretKeyRef *v1.SecretKeySelector `json:"secretKeyRef,omitempty" protobuf:"bytes,1,opt,name=secretKeyRef"`
}

type HTTPHeader struct {
	Name      string            `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value     string            `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
	ValueFrom *HTTPHeaderSource `json:"valueFrom,omitempty" protobuf:"bytes,3,opt,name=valueFrom"`
}

type HTTP struct {
	Method         string       `json:"method,omitempty" protobuf:"bytes,1,opt,name=method"`
	URL            string       `json:"url" protobuf:"bytes,2,opt,name=url"`
	Headers        []HTTPHeader `json:"headers,omitempty" protobuf:"bytes,3,rep,name=headers"`
	TimeoutSeconds *int64       `json:"timeoutSeconds,omitempty" protobuf:"bytes,4,opt,name=timeoutSeconds"`
	Body           []byte       `json:"body,omitempty" protobuf:"bytes,5,opt,name=body"`
}
