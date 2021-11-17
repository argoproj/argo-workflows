package controller

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type Workflow struct {
	// Required: true
	ObjectMeta metav1.ObjectMeta `json:"metadata"`
}
