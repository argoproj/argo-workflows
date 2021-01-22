package workflow

import (
	"os"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// report if the change is caused by a user (rather than just a status update by the controller)
func UserModified(a, b *unstructured.Unstructured) bool {
	return os.Getenv("ALWAYS_USER_MODIFIED") == "true" ||
		!reflect.DeepEqual(a.Object["spec"], b.Object["spec"]) ||
		statusUserModifiable(b)
}

func statusUserModifiable(un *unstructured.Unstructured) bool {
	templates, _, _ := unstructured.NestedSlice(un.Object, "spec", "templates")
	for _, t := range templates {
		_, ok := t.(map[string]interface{})["suspend"]
		if ok {
			return ok
		}
	}
	return false
}
