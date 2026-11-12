package util

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestParseUserOverrideAllowlist(t *testing.T) {
	for _, env := range []string{"", "  "} {
		got, err := parseUserOverrideAllowlist(env)
		require.NoError(t, err)
		assert.Nil(t, got)
	}

	// Input is the YAML/JSON name; output is the Go field name that keys allowedUserOverrideFields.
	got, err := parseUserOverrideAllowlist(" podSpecPatch , volumes ")
	require.NoError(t, err)
	assert.Equal(t, []string{"PodSpecPatch", "Volumes"}, got)

	// Go identifiers are not accepted — operators write YAML names.
	_, err = parseUserOverrideAllowlist("PodSpecPatch")
	require.EqualError(t, err, `WORKFLOW_USER_OVERRIDE_ALLOWLIST: "PodSpecPatch" is not a WorkflowSpec field name`)

	_, err = parseUserOverrideAllowlist("notAField")
	require.EqualError(t, err, `WORKFLOW_USER_OVERRIDE_ALLOWLIST: "notAField" is not a WorkflowSpec field name`)
}

func TestMergeWorkflows(t *testing.T) {
	patchWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/orig-wf.yaml")
	targetWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/patch.yaml")

	err := MergeTo(patchWf, targetWf)
	require.NoError(t, err)
	assert.Equal(t, "start", targetWf.Spec.Entrypoint)
	assert.Equal(t, "argo1", targetWf.Spec.ServiceAccountName)
	assert.Equal(t, "message", targetWf.Spec.Arguments.Parameters[0].Name)
	assert.Equal(t, "patch", targetWf.Spec.Arguments.Parameters[0].Value.String())
}

func TestMergeMetaDataTo(t *testing.T) {
	assert := assert.New(t)
	meta1 := &metav1.ObjectMeta{
		Labels: map[string]string{
			"test": "test", "welcome": "welcome",
		},
		Annotations: map[string]string{
			"test": "test", "welcome": "welcome",
		},
	}
	meta2 := &metav1.ObjectMeta{
		Labels: map[string]string{
			"test1": "test", "welcome1": "welcome",
		},
		Annotations: map[string]string{
			"test1": "test", "welcome1": "welcome",
		},
	}
	mergeMetaDataTo(meta2, meta1)
	assert.Contains(meta1.Labels, "test1")
	assert.Contains(meta1.Annotations, "test1")
	assert.NotContains(meta2.Labels, "test")
}

func TestJoinWfSpecs(t *testing.T) {
	assert := assert.New(t)
	wfDefault := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf-default.yaml")
	wf1 := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf.yaml")
	// wf1 := wfv1.MustUnmarshalWorkflow(wf1)
	wft := wfv1.MustUnmarshalWorkflowTemplate("@testdata/merge/wft.yaml")
	result := wfv1.MustUnmarshalWorkflow("@testdata/merge/result-spec.yaml")

	targetWf, err := JoinWorkflowSpec(&wf1.Spec, wft.GetWorkflowSpec(), &wfDefault.Spec)
	require.NoError(t, err)
	assert.Equal(result.Spec, targetWf.Spec)
	assert.Len(targetWf.Spec.Templates, 3)
	assert.Equal("whalesay", targetWf.Spec.Entrypoint)
}

func TestJoinWfSpecArguments(t *testing.T) {
	assert := assert.New(t)
	wf := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf-arguments.yaml")
	wft := wfv1.MustUnmarshalWorkflowTemplate("@testdata/merge/wf-arguments-template.yaml")
	result := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf-arguments-result.yaml")

	targetWf, err := JoinWorkflowSpec(&wf.Spec, wft.GetWorkflowSpec(), nil)
	require.NoError(t, err)
	assert.Equal(result.Spec.Arguments, targetWf.Spec.Arguments)
}

func TestJoinWfSpecArgumentsWithNil(t *testing.T) {
	assert := assert.New(t)
	wf := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf-arguments.yaml")
	result := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf-arguments.yaml")
	targetWf, err := JoinWorkflowSpec(&wf.Spec, nil, nil)
	require.NoError(t, err)
	assert.Equal(result.Spec.Arguments, targetWf.Spec.Arguments)
}

func TestJoinWorkflowMetaData(t *testing.T) {
	assert := assert.New(t)
	t.Run("WfDefaultMetaData", func(t *testing.T) {
		wfDefault := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf-default.yaml")
		wf1 := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf.yaml")
		JoinWorkflowMetaData(&wf1.ObjectMeta, &wfDefault.ObjectMeta)
		assert.Contains(wf1.Labels, "testLabel")
		assert.Equal("default", wf1.Labels["testLabel"])
		assert.Contains(wf1.Annotations, "testAnnotation")
		assert.Equal("default", wf1.Annotations["testAnnotation"])
	})
	t.Run("WFTMetadata", func(t *testing.T) {
		wfDefault := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf-default.yaml")
		wf2 := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf.yaml")
		JoinWorkflowMetaData(&wf2.ObjectMeta, &wfDefault.ObjectMeta)
		assert.Contains(wf2.Labels, "testLabel")
		assert.Equal("default", wf2.Labels["testLabel"])
		assert.Contains(wf2.Annotations, "testAnnotation")
		assert.Equal("default", wf2.Annotations["testAnnotation"])
	})
	t.Run("WfMetadata", func(t *testing.T) {
		wfDefault := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf-default.yaml")
		wf2 := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf.yaml")
		wf2.Labels = map[string]string{"testLabel": "wf"}
		wf2.Annotations = map[string]string{"testAnnotation": "wf"}
		JoinWorkflowMetaData(&wf2.ObjectMeta, &wfDefault.ObjectMeta)
		assert.Contains(wf2.Labels, "testLabel")
		assert.Equal("wf", wf2.Labels["testLabel"])
		assert.Contains(wf2.Annotations, "testAnnotation")
		assert.Equal("wf", wf2.Annotations["testAnnotation"])
	})
}

func TestMergeHooks(t *testing.T) {
	t.Run("NilBaseAndNilPatch", func(t *testing.T) {
		patchHookWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/patch-nil-hook.yaml")
		targetHookWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/base-nil.yaml")

		err := MergeTo(patchHookWf, targetHookWf)
		require.NoError(t, err)
		assert.Nil(t, targetHookWf.Spec.Hooks)
	})

	t.Run("NilBaseAndNotNilPatch", func(t *testing.T) {
		patchHookWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/patch-hook.yaml")
		targetHookWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/base-nil.yaml")

		err := MergeTo(patchHookWf, targetHookWf)
		require.NoError(t, err)
		assert.Len(t, targetHookWf.Spec.Hooks, 2)
		assert.Equal(t, "c", targetHookWf.Spec.Hooks[`foo`].Template)
		assert.Equal(t, "b", targetHookWf.Spec.Hooks[`bar`].Template)
	})

	// Ensure hook bar ends up in result, but foo is unchanged
	t.Run("NotNilBaseAndPatch", func(t *testing.T) {
		patchHookWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/patch-hook.yaml")
		targetHookWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/base-hook.yaml")

		err := MergeTo(patchHookWf, targetHookWf)
		require.NoError(t, err)
		assert.Len(t, targetHookWf.Spec.Hooks, 2)
		assert.Equal(t, "a", targetHookWf.Spec.Hooks[`foo`].Template)
		assert.Equal(t, "b", targetHookWf.Spec.Hooks[`bar`].Template)
	})
}

func TestMergeLabelsFrom(t *testing.T) {
	t.Run("NilBaseAndNotNilPatch", func(t *testing.T) {
		patchWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/patch-labels-from.yaml")
		targetWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/base-nil.yaml")

		err := MergeTo(patchWf, targetWf)
		require.NoError(t, err)
		assert.Len(t, targetWf.Spec.WorkflowMetadata.LabelsFrom, 2)
		assert.Equal(t, "PATCH", targetWf.Spec.WorkflowMetadata.LabelsFrom[`foo`].Expression)
		assert.Equal(t, "PATCH", targetWf.Spec.WorkflowMetadata.LabelsFrom[`bar`].Expression)
	})

	t.Run("NotNilBaseAndPatch", func(t *testing.T) {
		patchWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/patch-labels-from.yaml")
		targetWf := wfv1.MustUnmarshalWorkflow("@testdata/merge/base-labels-from.yaml")

		err := MergeTo(patchWf, targetWf)
		require.NoError(t, err)
		assert.Len(t, targetWf.Spec.WorkflowMetadata.LabelsFrom, 3)
		assert.Equal(t, "BASE", targetWf.Spec.WorkflowMetadata.LabelsFrom[`foo`].Expression)
		assert.Equal(t, "PATCH", targetWf.Spec.WorkflowMetadata.LabelsFrom[`bar`].Expression)
		assert.Equal(t, "BASE", targetWf.Spec.WorkflowMetadata.LabelsFrom[`baz`].Expression)
	})
}

// blockedUserOverrideFields is used by TestAllWorkflowSpecFieldsAccountedFor
// to verify that every WorkflowSpec field is consciously classified as either
// allowed or blocked.
var blockedUserOverrideFields = map[string]bool{
	"Templates":                    true,
	"TemplateDefaults":             true,
	"ServiceAccountName":           true,
	"AutomountServiceAccountToken": true,
	"Executor":                     true,
	"Volumes":                      true,
	"VolumeClaimTemplates":         true,
	"Parallelism":                  true,
	"NodeSelector":                 true,
	"Affinity":                     true,
	"Tolerations":                  true,
	"ImagePullSecrets":             true,
	"HostNetwork":                  true,
	"DNSPolicy":                    true,
	"DNSConfig":                    true,
	"OnExit":                       true,
	"SchedulerName":                true,
	"PodPriorityClassName":         true,
	"HostAliases":                  true,
	"SecurityContext":              true,
	"PodSpecPatch":                 true,
	"PodDisruptionBudget":          true,
	"ArtifactRepositoryRef":        true,
	"Synchronization":              true,
	"RetryStrategy":                true,
	"PodMetadata":                  true,
	"Hooks":                        true,
	"PodResources":                 true,
	"ExecutorPlugins":              true,
	"ResourceClaims":               true,
}

func TestValidateUserOverrides_AllowedFields(t *testing.T) {
	spec := &wfv1.WorkflowSpec{
		Entrypoint: "main",
		Arguments: wfv1.Arguments{
			Parameters: []wfv1.Parameter{{Name: "msg", Value: wfv1.AnyStringPtr("hello")}},
		},
		Shutdown:            wfv1.ShutdownStrategyTerminate,
		Priority:            ptr.To[int32](10),
		WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "my-template"},
	}
	err := ValidateUserOverrides(spec)
	assert.NoError(t, err)
}

func TestValidateUserOverrides_BlockedFields(t *testing.T) {
	tests := []struct {
		name  string
		spec  wfv1.WorkflowSpec
		field string
	}{
		{
			name:  "ServiceAccountName",
			spec:  wfv1.WorkflowSpec{ServiceAccountName: "admin"},
			field: "ServiceAccountName",
		},
		{
			name:  "SecurityContext",
			spec:  wfv1.WorkflowSpec{SecurityContext: &apiv1.PodSecurityContext{}},
			field: "SecurityContext",
		},
		{
			name:  "Templates",
			spec:  wfv1.WorkflowSpec{Templates: []wfv1.Template{{Name: "evil"}}},
			field: "Templates",
		},
		{
			name:  "Volumes",
			spec:  wfv1.WorkflowSpec{Volumes: []apiv1.Volume{{Name: "secret-vol"}}},
			field: "Volumes",
		},
		{
			name:  "HostNetwork",
			spec:  wfv1.WorkflowSpec{HostNetwork: new(true)},
			field: "HostNetwork",
		},
		{
			name:  "PodSpecPatch",
			spec:  wfv1.WorkflowSpec{PodSpecPatch: `{"containers":[]}`},
			field: "PodSpecPatch",
		},
		{
			name:  "OnExit",
			spec:  wfv1.WorkflowSpec{OnExit: "backdoor"},
			field: "OnExit",
		},
		{
			name:  "Hooks",
			spec:  wfv1.WorkflowSpec{Hooks: wfv1.LifecycleHooks{"exit": wfv1.LifecycleHook{Template: "evil"}}},
			field: "Hooks",
		},
		{
			name: "ExecutorPlugins",
			spec: wfv1.WorkflowSpec{
				ExecutorPlugins: []wfv1.ExecutorPlugin{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "example-plugin"},
						Spec: wfv1.ExecutorPluginSpec{
							Sidecar: wfv1.ExecutorPluginSidecar{
								Container: apiv1.Container{Name: "example-plugin", Image: "example/plugin:latest"},
							},
						},
					},
				},
			},
			field: "ExecutorPlugins",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserOverrides(&tt.spec)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.field)
			assert.Contains(t, err.Error(), "not permitted")
		})
	}
}

func TestValidateUserOverrides_MultipleViolations(t *testing.T) {
	spec := &wfv1.WorkflowSpec{
		ServiceAccountName: "admin",
		HostNetwork:        new(true),
		PodSpecPatch:       `{}`,
		Templates:          []wfv1.Template{{Name: "evil"}},
	}
	err := ValidateUserOverrides(spec)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "ServiceAccountName")
	assert.Contains(t, msg, "HostNetwork")
	assert.Contains(t, msg, "PodSpecPatch")
	assert.Contains(t, msg, "Templates")
}

func TestValidateUserOverrides_NilSpec(t *testing.T) {
	assert.NoError(t, ValidateUserOverrides(nil))
}

func TestSanitizeUserWorkflowSpec(t *testing.T) {
	spec := &wfv1.WorkflowSpec{
		Entrypoint:         "main",
		ServiceAccountName: "admin",
		HostNetwork:        new(true),
		Arguments: wfv1.Arguments{
			Parameters: []wfv1.Parameter{{Name: "msg", Value: wfv1.AnyStringPtr("hello")}},
		},
		Volumes:             []apiv1.Volume{{Name: "secret-vol"}},
		Templates:           []wfv1.Template{{Name: "evil"}},
		Shutdown:            wfv1.ShutdownStrategyTerminate,
		WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "my-template"},
		ExecutorPlugins: []wfv1.ExecutorPlugin{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "example-plugin"},
				Spec: wfv1.ExecutorPluginSpec{
					Sidecar: wfv1.ExecutorPluginSidecar{
						Container: apiv1.Container{Name: "example-plugin", Image: "example/plugin:latest"},
					},
				},
			},
		},
	}

	sanitized := SanitizeUserWorkflowSpec(spec)

	// Allowed fields are preserved
	assert.Equal(t, "main", sanitized.Entrypoint)
	assert.Equal(t, wfv1.ShutdownStrategyTerminate, sanitized.Shutdown)
	assert.Len(t, sanitized.Arguments.Parameters, 1)
	assert.Equal(t, "my-template", sanitized.WorkflowTemplateRef.Name)

	// Blocked fields are zeroed
	assert.Empty(t, sanitized.ServiceAccountName)
	assert.Nil(t, sanitized.HostNetwork)
	assert.Nil(t, sanitized.Volumes)
	assert.Nil(t, sanitized.Templates)
	assert.Nil(t, sanitized.ExecutorPlugins)
}

func TestSanitizeUserWorkflowSpec_Nil(t *testing.T) {
	assert.Nil(t, SanitizeUserWorkflowSpec(nil))
}

// TestValidateUserOverrides_ArtifactGCNestedFields guards against the
// GHSA-3775-99mw-8rp4 (CVE-2026-42296) bug class: ArtifactGC is allow-listed,
// but its nested ServiceAccountName/PodSpecPatch/PodMetadata reach the
// artifact-GC Pod and so must be rejected under Strict/Secure mode.
func TestValidateUserOverrides_ArtifactGCNestedFields(t *testing.T) {
	t.Run("benign ArtifactGC fields are allowed", func(t *testing.T) {
		spec := &wfv1.WorkflowSpec{
			WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "my-template"},
			ArtifactGC: &wfv1.WorkflowLevelArtifactGC{
				ArtifactGC:            wfv1.ArtifactGC{Strategy: wfv1.ArtifactGCOnWorkflowCompletion},
				ForceFinalizerRemoval: true,
			},
		}
		assert.NoError(t, ValidateUserOverrides(spec))
	})

	tests := []struct {
		name  string
		agc   *wfv1.WorkflowLevelArtifactGC
		field string
	}{
		{
			name:  "ServiceAccountName",
			agc:   &wfv1.WorkflowLevelArtifactGC{ArtifactGC: wfv1.ArtifactGC{ServiceAccountName: "privileged"}},
			field: "ArtifactGC.ServiceAccountName",
		},
		{
			name:  "PodSpecPatch",
			agc:   &wfv1.WorkflowLevelArtifactGC{PodSpecPatch: `{"containers":[{"name":"main","image":"attacker/x"}]}`},
			field: "ArtifactGC.PodSpecPatch",
		},
		{
			name:  "PodMetadata",
			agc:   &wfv1.WorkflowLevelArtifactGC{ArtifactGC: wfv1.ArtifactGC{PodMetadata: &wfv1.Metadata{Labels: map[string]string{"a": "b"}}}},
			field: "ArtifactGC.PodMetadata",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &wfv1.WorkflowSpec{
				WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "my-template"},
				ArtifactGC:          tt.agc,
			}
			err := ValidateUserOverrides(spec)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.field)
			assert.Contains(t, err.Error(), "not permitted")
		})
	}
}

// TestSanitizeUserWorkflowSpec_ArtifactGC verifies defense-in-depth: the
// security-sensitive nested ArtifactGC fields are stripped while benign ones are
// kept, and the caller's original spec is not mutated.
func TestSanitizeUserWorkflowSpec_ArtifactGC(t *testing.T) {
	spec := &wfv1.WorkflowSpec{
		WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "my-template"},
		ArtifactGC: &wfv1.WorkflowLevelArtifactGC{
			ArtifactGC: wfv1.ArtifactGC{
				Strategy:           wfv1.ArtifactGCOnWorkflowCompletion,
				ServiceAccountName: "privileged",
				PodMetadata:        &wfv1.Metadata{Labels: map[string]string{"a": "b"}},
			},
			ForceFinalizerRemoval: true,
			PodSpecPatch:          `{"containers":[]}`,
		},
	}

	sanitized := SanitizeUserWorkflowSpec(spec)

	// Benign fields are preserved.
	require.NotNil(t, sanitized.ArtifactGC)
	assert.Equal(t, wfv1.ArtifactGCOnWorkflowCompletion, sanitized.ArtifactGC.Strategy)
	assert.True(t, sanitized.ArtifactGC.ForceFinalizerRemoval)

	// Security-sensitive fields are stripped.
	assert.Empty(t, sanitized.ArtifactGC.ServiceAccountName)
	assert.Empty(t, sanitized.ArtifactGC.PodSpecPatch)
	assert.Nil(t, sanitized.ArtifactGC.PodMetadata)

	// The original spec must not be mutated.
	assert.Equal(t, "privileged", spec.ArtifactGC.ServiceAccountName)
	assert.JSONEq(t, `{"containers":[]}`, spec.ArtifactGC.PodSpecPatch)
	assert.NotNil(t, spec.ArtifactGC.PodMetadata)
}

// TestAllWorkflowSpecFieldsAccountedFor is a compile-time safety net.
// It ensures that every field in WorkflowSpec appears in either the
// allowed or blocked list, so new fields force a conscious decision.
func TestAllWorkflowSpecFieldsAccountedFor(t *testing.T) {
	specType := reflect.TypeFor[wfv1.WorkflowSpec]()
	for field := range specType.Fields() {
		fieldName := field.Name
		inAllowed := allowedUserOverrideFields[fieldName]
		inBlocked := blockedUserOverrideFields[fieldName]
		if !inAllowed && !inBlocked {
			t.Errorf("WorkflowSpec field %q is not classified in either allowedUserOverrideFields or blockedUserOverrideFields — add it to one of them", fieldName)
		}
		if inAllowed && inBlocked {
			t.Errorf("WorkflowSpec field %q appears in both allowedUserOverrideFields and blockedUserOverrideFields — it should be in exactly one", fieldName)
		}
	}
}

func TestJoinWorkflowSpecWithArtifactOverride(t *testing.T) {
	assert := assert.New(t)
	wf := wfv1.MustUnmarshalWorkflow("@testdata/merge/wf-with-artifact-override.yaml")
	wft := wfv1.MustUnmarshalWorkflowTemplate("@testdata/merge/wft-with-artifact.yaml")

	targetWf, err := JoinWorkflowSpec(&wf.Spec, wft.GetWorkflowSpec(), nil)
	require.NoError(t, err)

	// Should have 1 artifact (my-uploaded-file)
	assert.Len(targetWf.Spec.Arguments.Artifacts, 1)
	assert.Equal("my-uploaded-file", targetWf.Spec.Arguments.Artifacts[0].Name)

	// The key should be from the workflow (overridden), not from the template
	key, err := targetWf.Spec.Arguments.Artifacts[0].S3.GetKey()
	require.NoError(t, err)
	assert.Equal("uploads/argo/12345678-1234-1234-1234-123456789012/file.zip", key)
	assert.NotEqual("input-file/placeholder", key)
}

// TestJoinWorkflowSpec_ResourceClaimsAreReplaced verifies that a workflow's
// resourceClaims replace the WorkflowTemplate's list rather than merging by
// claim name. Merging would union the two entries, leaving a claim with both
// resourceClaimName and resourceClaimTemplateName set, which the API server
// rejects when the pod is created, halfway through the workflow. Neither spec
// is invalid on its own, so validation never sees the combination.
func TestJoinWorkflowSpec_ResourceClaimsAreReplaced(t *testing.T) {
	wftSpec := &wfv1.WorkflowSpec{
		ResourceClaims: []apiv1.PodResourceClaim{
			{Name: "gpu", ResourceClaimTemplateName: new("template-gpu")},
			{Name: "nic", ResourceClaimTemplateName: new("template-nic")},
		},
	}
	wfSpec := &wfv1.WorkflowSpec{
		ResourceClaims: []apiv1.PodResourceClaim{
			{Name: "gpu", ResourceClaimName: new("shared-gpu")},
		},
	}

	got, err := JoinWorkflowSpec(wfSpec, wftSpec, nil)
	require.NoError(t, err)

	require.Len(t, got.Spec.ResourceClaims, 1, "the workflow's list replaces the template's, so nic must not be inherited")
	claim := got.Spec.ResourceClaims[0]
	assert.Equal(t, "gpu", claim.Name)
	assert.Equal(t, new("shared-gpu"), claim.ResourceClaimName)
	assert.Nil(t, claim.ResourceClaimTemplateName, "a claim must never end up with both sources set")
}

// TestJoinWorkflowSpec_ArtifactFieldMerge verifies that a key-only artifact
// override at the workflow level does not drop bucket/endpoint inherited
// from the workflow-defaults level, since those fields live on the same S3
// backend and only the key is meant to change.
func TestJoinWorkflowSpec_ArtifactFieldMerge(t *testing.T) {
	wfSpec := &wfv1.WorkflowSpec{
		Arguments: wfv1.Arguments{
			Artifacts: wfv1.Artifacts{
				{
					Name: "my-artifact",
					ArtifactLocation: wfv1.ArtifactLocation{
						S3: &wfv1.S3Artifact{Key: "override-key"},
					},
				},
			},
		},
	}
	wfDefaultSpec := &wfv1.WorkflowSpec{
		Arguments: wfv1.Arguments{
			Artifacts: wfv1.Artifacts{
				{
					Name: "my-artifact",
					ArtifactLocation: wfv1.ArtifactLocation{
						S3: &wfv1.S3Artifact{
							S3Bucket: wfv1.S3Bucket{
								Endpoint: "s3.amazonaws.com",
								Bucket:   "my-bucket",
							},
							Key: "default-key",
						},
					},
				},
			},
		},
	}

	targetWf, err := JoinWorkflowSpec(wfSpec, nil, wfDefaultSpec)
	require.NoError(t, err)

	require.Len(t, targetWf.Spec.Arguments.Artifacts, 1)
	merged := targetWf.Spec.Arguments.Artifacts[0].S3
	require.NotNil(t, merged)
	assert.Equal(t, "s3.amazonaws.com", merged.Endpoint)
	assert.Equal(t, "my-bucket", merged.Bucket)
	assert.Equal(t, "override-key", merged.Key)
}
