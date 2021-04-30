package util

import (
	"fmt"
	"os"

	esv1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	selfLinkEnabled      = os.Getenv("SELF_LINK_ENABLED") == "true"
	managedFieldsEnabled = os.Getenv("MANAGED_FIELDS_ENABLED") == "true"
)

// CleanMetadata removes self links and managed fields.
// This is fast and idempotent, so can called safely multiple time, and you may liberally use it
func CleanMetadata(v interface{}) {
	RemoveSelfLink(v)
	RemoveManagedFields(v)
}

func RemoveManagedFields(v interface{}) {
	if managedFieldsEnabled {
		// noop
	} else if v == nil {
		// noop
	} else if x, ok := v.(metav1.Object); ok {
		x.SetManagedFields(nil)
	} else {
		switch x := v.(type) {
		case []esv1.EventSource:
			for i := range x {
				CleanMetadata(&x[i])
			}
		case []sv1.Sensor:
			for i := range x {
				CleanMetadata(&x[i])
			}
		case []wfv1.CronWorkflow:
			for i := range x {
				CleanMetadata(&x[i])
			}
		case wfv1.ClusterWorkflowTemplates:
			for i := range x {
				CleanMetadata(&x[i])
			}
		case wfv1.Workflows:
			for i := range x {
				CleanMetadata(&x[i])
			}
		case []wfv1.WorkflowEventBinding:
			for i := range x {
				CleanMetadata(&x[i])
			}
		case wfv1.WorkflowTemplates:
			for i := range x {
				CleanMetadata(&x[i])
			}
		default:
			panic(fmt.Errorf("this should be impossible - unexpected type: %T", v))
		}
	}
}

func RemoveSelfLink(v interface{}) {
	if selfLinkEnabled {
		// noop
	} else if v == nil {
		// noop
	} else if x, ok := v.(metav1.Object); ok {
		x.SetSelfLink("")
	} else if x, ok := v.(metav1.ListInterface); ok {
		x.SetSelfLink("")
	} else {
		panic(fmt.Errorf("this should be impossible - unexpected type: %T", v))
	}
}
