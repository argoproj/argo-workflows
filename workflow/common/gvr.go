package common

import "k8s.io/apimachinery/pkg/runtime/schema"

var ConfigMapGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
var ServiceAccountGVR = schema.GroupVersionResource{Version: "v1", Resource: "serviceaccounts"}
var SecretsGVR = schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
var PodGVR = schema.GroupVersionResource{Version: "v1", Resource: "pods"}
