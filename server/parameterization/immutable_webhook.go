package parameterization

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// Enforce the same snapshot invariant on clusters without
// ValidatingAdmissionPolicy. This needs no Kubernetes read or write privileges:
// both documents come from the API server's admission request. In particular,
// decode arbitrary values without pruning fields or rounding large integers.
func (h AdmissionHandler) validateSnapshotUpdate(req *admissionv1.AdmissionRequest) error {
	conditional := false
	switch req.Resource.Resource {
	case "parametertemplates", "workflowforms", "workflowparameterbindings", "actionresulttemplates":
		conditional = true
	case "parameterrevisions", "workflowsubmissions", "actiondefinitions", "workflowactionbindings", "parameterretries", "parameternodeinputs", "resourceoperations":
	default:
		return nil
	}
	type document struct {
		Metadata metav1.ObjectMeta `json:"metadata"`
		Spec     map[string]any    `json:"spec"`
	}
	var old, next document
	for _, input := range []struct {
		raw    []byte
		target *document
	}{{req.OldObject.Raw, &old}, {req.Object.Raw, &next}} {
		decoder := json.NewDecoder(bytes.NewReader(input.raw))
		decoder.UseNumber()
		if err := decoder.Decode(input.target); err != nil || input.target.Spec == nil {
			return fmt.Errorf("invalid snapshot admission document")
		}
	}
	if req.Namespace == "" || req.Namespace != h.Namespace || old.Metadata.Namespace != req.Namespace || next.Metadata.Namespace != req.Namespace || old.Metadata.UID != next.Metadata.UID || old.Metadata.Name != next.Metadata.Name {
		return fmt.Errorf("snapshot identity or admission namespace changed")
	}
	_, published := old.Metadata.Annotations[p.Group+"/definition-digest"]
	before, after := any(old.Spec), any(next.Spec)
	if req.Resource.Resource == "resourceoperations" {
		before, after = old.Spec["operation"], next.Spec["operation"]
	}
	if (!conditional || published) && !reflect.DeepEqual(before, after) {
		return fmt.Errorf("snapshot is immutable; create a new revision or operation")
	}
	if published {
		for key, value := range old.Metadata.Annotations {
			if strings.HasPrefix(key, p.Group+"/definition-") {
				actual, exists := next.Metadata.Annotations[key]
				if !exists || actual != value {
					return fmt.Errorf("published definition provenance is immutable")
				}
			}
		}
		family, exists := next.Metadata.Labels[p.Group+"/definition-family"]
		if !exists || old.Metadata.Labels[p.Group+"/definition-family"] != family {
			return fmt.Errorf("published definition family is immutable")
		}
	}
	return nil
}
