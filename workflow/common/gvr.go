package common

import "k8s.io/apimachinery/pkg/runtime/schema"

var PodGVR = schema.GroupVersionResource{Version: "v1", Resource: "pods"}
