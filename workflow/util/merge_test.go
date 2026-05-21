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

var origWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  arguments:
    parameters:
    - name: message
      value: original
  entrypoint: start
  onExit: end
  serviceAccountName: default
  workflowTemplateRef:
    name: workflow-template-submittable
`

var patchWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  arguments:
    parameters:
    - name: message
      value: patch
  serviceAccountName: argo1
  podGC:
    strategy: OnPodSuccess
`

func TestMergeWorkflows(t *testing.T) {
	patchWf := wfv1.MustUnmarshalWorkflow(origWF)
	targetWf := wfv1.MustUnmarshalWorkflow(patchWF)

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

var wfDefault = `
metadata:
  annotations:
    testAnnotation: default
  labels:
    testLabel: default
spec:
  entrypoint: whalesay
  activeDeadlineSeconds: 7200
  arguments:
    artifacts:
      -
        name: message
        path: /tmp/message
    parameters:
      -
        name: message
        value: "hello world"
  onExit: whalesay-exit
  serviceAccountName: default
  templates:
    -
      container:
        args:
          - "hello from the default exit handler"
        command:
          - cowsay
        image: docker/whalesay
      name: whalesay-exit
  ttlStrategy:
    secondsAfterCompletion: 60
  volumes:
    -
      name: test
      secret:
        secretName: test
`

var wft = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-submittable
  namespace: default
spec:
  workflowMetaData:
    annotations:
      testAnnotation: wft
    labels:
      testLabel: wft
  entrypoint: whalesay-template
  arguments:
    parameters:
      - name: message
        value: hello world
  templates:
    - name: whalesay-template
      inputs:
        parameters:
          - name: message
      container:
        image: docker/whalesay
        command: [cowsay]
        args: ["{{inputs.parameters.message}}"]
`

var wf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
    -
      container:
        args:
          - "hello world"
        command:
          - cowsay
        image: "docker/whalesay:latest"
      name: whalesay
`

var resultSpec = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  activeDeadlineSeconds: 7200
  workflowMetadata:
    annotations:
      testAnnotation: wft
    labels:
      testLabel: wft
  arguments:
    artifacts:
      -
        name: message
        path: /tmp/message
    parameters:
      -
        name: message
        value: "hello world"
  entrypoint: whalesay
  onExit: whalesay-exit
  serviceAccountName: default
  templates:
    -
      container:
        args:
          - "hello world"
        command:
          - cowsay
        image: "docker/whalesay:latest"
      name: whalesay
    -
      container:
        args:
          - "{{inputs.parameters.message}}"
        command:
          - cowsay
        image: docker/whalesay
      inputs:
        parameters:
          -
            name: message
      name: whalesay-template
    -
      container:
        args:
          - "hello from the default exit handler"
        command:
          - cowsay
        image: docker/whalesay
      name: whalesay-exit
  ttlStrategy:
    secondsAfterCompletion: 60
  volumes:
    -
      name: test
      secret:
        secretName: test

`

var wfArguments = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: test-workflow
spec:
  workflowTemplateRef:
    name: test-workflow-template
  arguments:
    parameters:
      - name: PARAM1
        valueFrom:
          configMapKeyRef:
            name: test-config-map
            key: PARAM1
      - name: PARAM2
        valueFrom:
          configMapKeyRef:
            name: test-config-map
            key: PARAM2
      - name: PARAM4
        valueFrom:
          configMapKeyRef:
            name: test-config-map
            key: PARAM4
      - name: PARAM5
        value: "Workflow value 5"`

var wfArgumentsTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-workflow-template
spec:
  entrypoint: main
  ttlStrategy:
    secondsAfterCompletion: 600
    secondsAfterSuccess: 600
    secondsAfterFailure: 600
  arguments:
    parameters:
      - name: PARAM1
        value: WorkflowTemplate value 1 ignored
      - name: PARAM2
      - name: PARAM3
        value: WorkflowTemplate value 3
      - name: PARAM4
      - name: PARAM5
  templates:
    - name: main
      inputs:
        parameters:
          - name: PARAM1
            value: "{{workflow.parameters.PARAM1}}"
          - name: PARAM2
            value: "{{workflow.parameters.PARAM2}}"
          - name: PARAM3
            value: "{{workflow.parameters.PARAM3}}"
          - name: PARAM4
            value: "{{workflow.parameters.PARAM4}}"
          - name: PARAM5
            value: "{{workflow.parameters.PARAM5}}"
      script:
        image: busybox:latest
        command:
          - sh
        source: |
          echo -e "
            PARAM1={{inputs.parameters.PARAM1}}
            PARAM2={{inputs.parameters.PARAM2}}
            PARAM3={{inputs.parameters.PARAM3}}
            PARAM4={{inputs.parameters.PARAM4}}
            PARAM5={{inputs.parameters.PARAM5}}
          "
`

var wfArgumentsResult = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-workflow
spec:
  entrypoint: main
  ttlStrategy:
    secondsAfterCompletion: 600
    secondsAfterSuccess: 600
    secondsAfterFailure: 600
  arguments:
    parameters:
      - name: PARAM1
        valueFrom:
          configMapKeyRef:
            name: test-config-map
            key: PARAM1
      - name: PARAM2
        valueFrom:
          configMapKeyRef:
            name: test-config-map
            key: PARAM2
      - name: PARAM3
        value: WorkflowTemplate value 3
      - name: PARAM4
        valueFrom:
          configMapKeyRef:
            name: test-config-map
            key: PARAM4
      - name: PARAM5
        value: "Workflow value 5"
  templates:
    - name: main
      inputs:
        parameters:
          - name: PARAM1
            value: "{{workflow.parameters.PARAM1}}"
          - name: PARAM2
            value: "{{workflow.parameters.PARAM2}}"
          - name: PARAM3
            value: "{{workflow.parameters.PARAM3}}"
          - name: PARAM4
            value: "{{workflow.parameters.PARAM4}}"
          - name: PARAM5
            value: "{{workflow.parameters.PARAM5}}"
      script:
        image: busybox:latest
        command:
          - sh
        source: |
          echo -e "
            PARAM1={{inputs.parameters.PARAM1}}
            PARAM2={{inputs.parameters.PARAM2}}
            PARAM3={{inputs.parameters.PARAM3}}
            PARAM4={{inputs.parameters.PARAM4}}
            PARAM5={{inputs.parameters.PARAM5}}
          "
`

func TestJoinWfSpecs(t *testing.T) {
	assert := assert.New(t)
	wfDefault := wfv1.MustUnmarshalWorkflow(wfDefault)
	wf1 := wfv1.MustUnmarshalWorkflow(wf)
	// wf1 := wfv1.MustUnmarshalWorkflow(wf1)
	wft := wfv1.MustUnmarshalWorkflowTemplate(wft)
	result := wfv1.MustUnmarshalWorkflow(resultSpec)

	targetWf, err := JoinWorkflowSpec(&wf1.Spec, wft.GetWorkflowSpec(), &wfDefault.Spec)
	require.NoError(t, err)
	assert.Equal(result.Spec, targetWf.Spec)
	assert.Len(targetWf.Spec.Templates, 3)
	assert.Equal("whalesay", targetWf.Spec.Entrypoint)
}

func TestJoinWfSpecArguments(t *testing.T) {
	assert := assert.New(t)
	wf := wfv1.MustUnmarshalWorkflow(wfArguments)
	wft := wfv1.MustUnmarshalWorkflowTemplate(wfArgumentsTemplate)
	result := wfv1.MustUnmarshalWorkflow(wfArgumentsResult)

	targetWf, err := JoinWorkflowSpec(&wf.Spec, wft.GetWorkflowSpec(), nil)
	require.NoError(t, err)
	assert.Equal(result.Spec.Arguments, targetWf.Spec.Arguments)
}

func TestJoinWfSpecArgumentsWithNil(t *testing.T) {
	assert := assert.New(t)
	wf := wfv1.MustUnmarshalWorkflow(wfArguments)
	result := wfv1.MustUnmarshalWorkflow(wfArguments)
	targetWf, err := JoinWorkflowSpec(&wf.Spec, nil, nil)
	require.NoError(t, err)
	assert.Equal(result.Spec.Arguments, targetWf.Spec.Arguments)
}

func TestJoinWorkflowMetaData(t *testing.T) {
	assert := assert.New(t)
	t.Run("WfDefaultMetaData", func(t *testing.T) {
		wfDefault := wfv1.MustUnmarshalWorkflow(wfDefault)
		wf1 := wfv1.MustUnmarshalWorkflow(wf)
		JoinWorkflowMetaData(&wf1.ObjectMeta, &wfDefault.ObjectMeta)
		assert.Contains(wf1.Labels, "testLabel")
		assert.Equal("default", wf1.Labels["testLabel"])
		assert.Contains(wf1.Annotations, "testAnnotation")
		assert.Equal("default", wf1.Annotations["testAnnotation"])
	})
	t.Run("WFTMetadata", func(t *testing.T) {
		wfDefault := wfv1.MustUnmarshalWorkflow(wfDefault)
		wf2 := wfv1.MustUnmarshalWorkflow(wf)
		JoinWorkflowMetaData(&wf2.ObjectMeta, &wfDefault.ObjectMeta)
		assert.Contains(wf2.Labels, "testLabel")
		assert.Equal("default", wf2.Labels["testLabel"])
		assert.Contains(wf2.Annotations, "testAnnotation")
		assert.Equal("default", wf2.Annotations["testAnnotation"])
	})
	t.Run("WfMetadata", func(t *testing.T) {
		wfDefault := wfv1.MustUnmarshalWorkflow(wfDefault)
		wf2 := wfv1.MustUnmarshalWorkflow(wf)
		wf2.Labels = map[string]string{"testLabel": "wf"}
		wf2.Annotations = map[string]string{"testAnnotation": "wf"}
		JoinWorkflowMetaData(&wf2.ObjectMeta, &wfDefault.ObjectMeta)
		assert.Contains(wf2.Labels, "testLabel")
		assert.Equal("wf", wf2.Labels["testLabel"])
		assert.Contains(wf2.Annotations, "testAnnotation")
		assert.Equal("wf", wf2.Annotations["testAnnotation"])
	})
}

var baseNilWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
`

var baseHookWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  hooks:
    foo:
      template: a
      expression: workflow.status == "Pending"
`

var patchNilHookWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec:
`

var patchHookWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec:
  hooks:
    foo:
      template: c
      expression: workflow.status == "Pending"
    bar:
      template: b
      expression: workflow.status == "Pending"
`

func TestMergeHooks(t *testing.T) {
	t.Run("NilBaseAndNilPatch", func(t *testing.T) {
		patchHookWf := wfv1.MustUnmarshalWorkflow(patchNilHookWF)
		targetHookWf := wfv1.MustUnmarshalWorkflow(baseNilWF)

		err := MergeTo(patchHookWf, targetHookWf)
		require.NoError(t, err)
		assert.Nil(t, targetHookWf.Spec.Hooks)
	})

	t.Run("NilBaseAndNotNilPatch", func(t *testing.T) {
		patchHookWf := wfv1.MustUnmarshalWorkflow(patchHookWF)
		targetHookWf := wfv1.MustUnmarshalWorkflow(baseNilWF)

		err := MergeTo(patchHookWf, targetHookWf)
		require.NoError(t, err)
		assert.Len(t, targetHookWf.Spec.Hooks, 2)
		assert.Equal(t, "c", targetHookWf.Spec.Hooks[`foo`].Template)
		assert.Equal(t, "b", targetHookWf.Spec.Hooks[`bar`].Template)
	})

	// Ensure hook bar ends up in result, but foo is unchanged
	t.Run("NotNilBaseAndPatch", func(t *testing.T) {
		patchHookWf := wfv1.MustUnmarshalWorkflow(patchHookWF)
		targetHookWf := wfv1.MustUnmarshalWorkflow(baseHookWF)

		err := MergeTo(patchHookWf, targetHookWf)
		require.NoError(t, err)
		assert.Len(t, targetHookWf.Spec.Hooks, 2)
		assert.Equal(t, "a", targetHookWf.Spec.Hooks[`foo`].Template)
		assert.Equal(t, "b", targetHookWf.Spec.Hooks[`bar`].Template)
	})
}

var patchLabelsFromWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec:
  workflowMetadata:
    labelsFrom:
      foo:
        expression: PATCH
      bar:
        expression: PATCH
`
var baseLabelsFromWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec:
  workflowMetadata:
    labelsFrom:
      foo:
        expression: BASE
      baz:
        expression: BASE
`

func TestMergeLabelsFrom(t *testing.T) {
	t.Run("NilBaseAndNotNilPatch", func(t *testing.T) {
		patchWf := wfv1.MustUnmarshalWorkflow(patchLabelsFromWF)
		targetWf := wfv1.MustUnmarshalWorkflow(baseNilWF)

		err := MergeTo(patchWf, targetWf)
		require.NoError(t, err)
		assert.Len(t, targetWf.Spec.WorkflowMetadata.LabelsFrom, 2)
		assert.Equal(t, "PATCH", targetWf.Spec.WorkflowMetadata.LabelsFrom[`foo`].Expression)
		assert.Equal(t, "PATCH", targetWf.Spec.WorkflowMetadata.LabelsFrom[`bar`].Expression)
	})

	t.Run("NotNilBaseAndPatch", func(t *testing.T) {
		patchWf := wfv1.MustUnmarshalWorkflow(patchLabelsFromWF)
		targetWf := wfv1.MustUnmarshalWorkflow(baseLabelsFromWF)

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
}

func TestSanitizeUserWorkflowSpec_Nil(t *testing.T) {
	assert.Nil(t, SanitizeUserWorkflowSpec(nil))
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
