package parameterization

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestSnapshotWebhookProtectsSchemalessValuesWithoutPolicyAPI(t *testing.T) {
	for _, resource := range []string{"parameterrevisions", "workflowsubmissions", "actiondefinitions", "workflowactionbindings", "parameterretries", "parameternodeinputs", "resourceoperations", "parametertemplates", "workflowforms", "workflowparameterbindings", "actionresulttemplates"} {
		t.Run(resource, func(t *testing.T) {
			old := object("Snapshot", "snapshot", "lab", map[string]any{"operation": map[string]any{"value": json.Number("9007199254740992")}})
			old.SetAnnotations(map[string]string{p.Group + "/definition-digest": "digest", p.Group + "/definition-source": "source"})
			old.SetLabels(map[string]string{p.Group + "/definition-family": "family"})
			validate := func(next map[string]any) error {
				a, _ := json.Marshal(old.Object)
				b, _ := json.Marshal(next)
				return (AdmissionHandler{Namespace: "lab"}).validate(t.Context(), &admissionv1.AdmissionRequest{Namespace: "lab", Resource: metav1.GroupVersionResource{Group: p.Group, Version: p.Version, Resource: resource}, Operation: admissionv1.Update, OldObject: runtime.RawExtension{Raw: a}, Object: runtime.RawExtension{Raw: b}})
			}
			require.NoError(t, validate(old.Object))
			next := old.DeepCopy()
			next.Object["status"] = map[string]any{"phase": "Ready"}
			require.NoError(t, validate(next.Object))
			next = old.DeepCopy()
			next.Object["spec"].(map[string]any)["operation"] = map[string]any{"value": json.Number("9007199254740993")}
			require.ErrorContains(t, validate(next.Object), "immutable")
			next = old.DeepCopy()
			next.SetAnnotations(nil)
			require.ErrorContains(t, validate(next.Object), "provenance")
			next = old.DeepCopy()
			next.SetLabels(nil)
			require.ErrorContains(t, validate(next.Object), "family")
			next = old.DeepCopy()
			next.SetNamespace("other")
			require.ErrorContains(t, validate(next.Object), "namespace")
		})
	}
}

func TestSnapshotWebhookAllowsDraftAndOperationControlUpdates(t *testing.T) {
	for _, resource := range []string{"parametertemplates", "workflowforms", "workflowparameterbindings", "actionresulttemplates", "resourceoperations"} {
		old := object("Snapshot", "draft", "lab", map[string]any{"operation": map[string]any{"input": "fixed"}, "control": "old"})
		next := old.DeepCopy()
		next.Object["spec"].(map[string]any)["control"] = "new"
		a, _ := json.Marshal(old.Object)
		b, _ := json.Marshal(next.Object)
		require.NoError(t, (AdmissionHandler{Namespace: "lab"}).validate(t.Context(), &admissionv1.AdmissionRequest{Namespace: "lab", Resource: metav1.GroupVersionResource{Group: p.Group, Version: p.Version, Resource: resource}, Operation: admissionv1.Update, OldObject: runtime.RawExtension{Raw: a}, Object: runtime.RawExtension{Raw: b}}))
	}
}
