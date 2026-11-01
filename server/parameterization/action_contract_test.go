package parameterization

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
)

const contractTestImage = "example.invalid/demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func reviewedActionFixture(t *testing.T) (*wfv1.Template, ActionDefinition) {
	t.Helper()
	def := ActionDefinition{ExecutionContract: &parameterapi.ActionExecutionContract{Effect: "Pure", TargetScope: "local-calculation-v1", PermissionScope: "none-v1", OutputMeaning: "result-v1", Capabilities: []parameterapi.ActionCapability{{Name: "Idempotency", Condition: "same inputs"}}}, InputSchema: map[string]any{"type": "object"}, OutputSchema: map[string]any{"type": "object"}, Template: &wfv1.Template{Name: "replacement", Container: &corev1.Container{Image: contractTestImage}}}
	baseline := def.Template.DeepCopy()
	baseline.Name = "original"
	policy := ActionReplacementPolicy{Contract: *def.ExecutionContract, ImplementationDigests: []string{actionImplementationDigest(def)}}
	raw, err := json.Marshal(policy)
	require.NoError(t, err)
	baseline.Metadata.Annotations = map[string]string{actionReplacementAnnotation: string(raw)}
	return baseline, def
}

func TestActionReplacementRequiresAuthorReviewedSemanticsAndImplementation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*wfv1.Template, *ActionDefinition)
		want   string
	}{
		{"matching", func(_ *wfv1.Template, _ *ActionDefinition) {}, ""},
		{"target", func(_ *wfv1.Template, d *ActionDefinition) { d.ExecutionContract.TargetScope = "another-target" }, "changes effect"},
		{"permissions", func(_ *wfv1.Template, d *ActionDefinition) { d.ExecutionContract.PermissionScope = "elevated" }, "permission scope"},
		{"output meaning", func(_ *wfv1.Template, d *ActionDefinition) { d.ExecutionContract.OutputMeaning = "another-unit" }, "output meaning"},
		{"effect", func(_ *wfv1.Template, d *ActionDefinition) { d.ExecutionContract.Effect = "Mutating" }, "changes effect"},
		{"condition", func(_ *wfv1.Template, d *ActionDefinition) {
			d.ExecutionContract.Capabilities[0].Condition = "only after approval"
		}, "capability conditions"},
		{"missing capability", func(_ *wfv1.Template, d *ActionDefinition) { d.ExecutionContract.Capabilities = nil }, "capability conditions"},
		{"changed executable", func(_ *wfv1.Template, d *ActionDefinition) { d.Template.Container.Command = []string{"other-program"} }, "not been reviewed"},
		{"changed output schema", func(_ *wfv1.Template, d *ActionDefinition) { d.OutputSchema["description"] = "a different result" }, "not been reviewed"},
		{"changed security", func(_ *wfv1.Template, d *ActionDefinition) { d.Template.ServiceAccountName = "another-account" }, "not been reviewed"},
		{"missing contract", func(_ *wfv1.Template, d *ActionDefinition) { d.ExecutionContract = nil }, "both an execution contract"},
		{"missing policy", func(b *wfv1.Template, _ *ActionDefinition) { b.Metadata.Annotations = nil }, "author-approved baseline"},
		{"empty policy", func(b *wfv1.Template, _ *ActionDefinition) { b.Metadata.Annotations[actionReplacementAnnotation] = "" }, "invalid action replacement policy"},
		{"unknown policy field", func(b *wfv1.Template, _ *ActionDefinition) {
			b.Metadata.Annotations[actionReplacementAnnotation] = `{"unexpected":true}`
		}, "invalid action replacement policy"},
		{"trailing policy", func(b *wfv1.Template, _ *ActionDefinition) {
			b.Metadata.Annotations[actionReplacementAnnotation] += " {}"
		}, "invalid action replacement policy"},
		{"mutable image", func(_ *wfv1.Template, d *ActionDefinition) { d.Template.Container.Image = "demo:latest" }, "immutable sha256"},
		{"mutable sidecar", func(_ *wfv1.Template, d *ActionDefinition) {
			d.Template.Sidecars = []wfv1.UserContainer{{Container: corev1.Container{Image: "sidecar:latest"}}}
		}, "immutable sha256"},
		{"dynamic scope", func(_ *wfv1.Template, d *ActionDefinition) {
			d.ExecutionContract.TargetScope = "{{workflow.parameters.scope}}"
		}, "static"},
		{"duplicate capability", func(_ *wfv1.Template, d *ActionDefinition) {
			d.ExecutionContract.Capabilities = append(d.ExecutionContract.Capabilities, d.ExecutionContract.Capabilities[0])
		}, "unique recognized"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			baseline, def := reviewedActionFixture(t)
			tc.mutate(baseline, &def)
			err := checkActionReplacement(baseline, def, ActionCall{})
			if tc.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.want)
			}
		})
	}
	baseline, def := reviewedActionFixture(t)
	require.ErrorContains(t, checkActionReplacement(baseline, def, ActionCall{OutputMappings: []parameterapi.ActionParameterMapping{{From: "a", To: "b"}}}), "interface mappings")
	digest := actionImplementationDigest(def)
	def.Template.Name = "compiler-assigned-name"
	require.Equal(t, digest, actionImplementationDigest(def))
	baseline.Metadata.Annotations = nil
	def.ExecutionContract = nil
	require.NoError(t, checkActionReplacement(baseline, def, ActionCall{}), "legacy bindings remain explicitly unverified")
}

func TestActionContractRejectsNativeEffectContradictions(t *testing.T) {
	for _, tc := range []struct {
		effect   string
		template wfv1.Template
		valid    bool
	}{
		{"ReadOnly", wfv1.Template{Resource: &wfv1.ResourceTemplate{Action: "get"}}, true},
		{"ReadOnly", wfv1.Template{Resource: &wfv1.ResourceTemplate{Action: "apply"}}, false},
		{"Pure", wfv1.Template{Resource: &wfv1.ResourceTemplate{Action: "get"}}, false},
		{"Mutating", wfv1.Template{Resource: &wfv1.ResourceTemplate{Action: "create"}}, true},
		{"ReadOnly", wfv1.Template{HTTP: &wfv1.HTTP{Method: "GET"}}, true},
		{"ReadOnly", wfv1.Template{HTTP: &wfv1.HTTP{Method: "POST"}}, false},
		{"Pure", wfv1.Template{HTTP: &wfv1.HTTP{Method: "GET"}}, false},
	} {
		_, def := reviewedActionFixture(t)
		def.ExecutionContract.Effect = tc.effect
		err := validateActionExecutionContract(def.ExecutionContract, &tc.template)
		if tc.valid {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, "contradicts")
		}
	}
}

func TestActionContractPreviewAndDefinitionValidationSharePolicy(t *testing.T) {
	ctx, s, _, revision := previewFixture(t)
	baseline, def := reviewedActionFixture(t)
	def.Template.Inputs.Parameters = []wfv1.Parameter{{Name: "count"}}
	def.InputSchema = map[string]any{"type": "object", "properties": map[string]any{"count": map[string]any{"type": "integer"}}}
	raw, err := json.Marshal(ActionReplacementPolicy{Contract: *def.ExecutionContract, ImplementationDigests: []string{actionImplementationDigest(def)}})
	require.NoError(t, err)
	native, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowTemplates("lab").Get(ctx, "native", metav1.GetOptions{})
	require.NoError(t, err)
	native.Spec.Templates[0] = *baseline
	native.Spec.Templates[0].Name = "main"
	native.Spec.Templates[0].Metadata.Annotations[actionReplacementAnnotation] = string(raw)
	_, err = auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowTemplates("lab").Update(ctx, native, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = s.resource("actiondefinitions").Create(ctx, object("ActionDefinition", "reviewed", "lab", def), metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = s.resource("workflowactionbindings").Create(ctx, object("WorkflowActionBinding", "actions", "lab", ActionBinding{WorkflowTemplate: "native", Calls: []ActionCall{{Template: "main", ActionDefinition: "reviewed"}}}), metav1.CreateOptions{})
	require.NoError(t, err)
	b, err := s.get(ctx, "workflowparameterbindings", "standard")
	require.NoError(t, err)
	b.Object["spec"].(map[string]any)["actionBinding"] = "actions"
	_, err = s.resource("workflowparameterbindings").Update(ctx, b, metav1.UpdateOptions{})
	require.NoError(t, err)
	review, err := s.actionReview(ctx, "reviewed")
	require.NoError(t, err)
	require.Equal(t, actionImplementationDigest(def), review.ImplementationDigest)
	preview, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.Empty(t, preview.Errors)
	require.Equal(t, def.ExecutionContract, preview.Definitions.Actions["reviewed"].ExecutionContract)
	native.Spec.Templates[0].Metadata.Annotations = nil
	_, err = auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowTemplates("lab").Update(ctx, native, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = s.Preview(ctx, "standard", revision)
	require.ErrorContains(t, err, "author-approved baseline")
}
