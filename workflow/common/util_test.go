package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/template"
)

const (
	validWf = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-custom-enforcer
spec:
  entrypoint: test-custom-enforcer
  templates:
  - name: test-custom-enforcer
    steps:
    - - name: crawl-tables
        template: echo
  - name: echo
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`
	invalidWf = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-custom-enforcer
spec:
  entrypoint: test-custom-enforcer
  templates:
  - name: test-custom-enforcer
    steps:
    - - name: crawl-tables
        doesNotExist: 10
        template: echo
  - name: echo
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]

`
)

// TestFindOverlappingVolume tests logic of TestFindOverlappingVolume
func TestFindOverlappingVolume(t *testing.T) {
	volMnt := corev1.VolumeMount{
		Name:      "workdir",
		MountPath: "/user-mount",
	}
	volMntTrailing := corev1.VolumeMount{
		Name:      "aux",
		MountPath: "/trailing-slash/",
	}
	templateWithVolMount := &wfv1.Template{
		Container: &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{volMnt, volMntTrailing},
		},
	}

	deeperVolMnt := corev1.VolumeMount{
		Name:      "workdir",
		MountPath: "/user-mount/deeper",
	}

	templateWithDeeperVolMount := &wfv1.Template{
		Container: &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{volMnt, deeperVolMnt},
		},
	}

	assert.Equal(t, &volMnt, FindOverlappingVolume(templateWithVolMount, "/user-mount"))
	assert.Equal(t, &volMnt, FindOverlappingVolume(templateWithVolMount, "/user-mount/subdir"))
	assert.Equal(t, &volMnt, FindOverlappingVolume(templateWithVolMount, "/user-mount/"))

	assert.Equal(t, &deeperVolMnt, FindOverlappingVolume(templateWithDeeperVolMount, "/user-mount/deeper"))
	assert.Equal(t, &deeperVolMnt, FindOverlappingVolume(templateWithDeeperVolMount, "/user-mount/deeper/with-subdir"))

	assert.Equal(t, &volMntTrailing, FindOverlappingVolume(templateWithVolMount, "/trailing-slash/"))
	assert.Equal(t, &volMntTrailing, FindOverlappingVolume(templateWithVolMount, "/trailing-slash/with-subpath"))

	assert.Nil(t, FindOverlappingVolume(templateWithVolMount, "/user-mount-coincidental-prefix/"))
}

func TestFindVolumeMountNestedUnderPath(t *testing.T) {
	mnt := corev1.VolumeMount{Name: "shared", MountPath: "/data/shared"}
	mntTrailing := corev1.VolumeMount{Name: "aux", MountPath: "/logs/out/"}
	tmpl := &wfv1.Template{
		Container: &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{mnt, mntTrailing},
		},
	}

	// path is a proper ancestor of a mount → detected.
	assert.Equal(t, &mnt, FindVolumeMountNestedUnderPath(tmpl, "/data"))
	assert.Equal(t, &mnt, FindVolumeMountNestedUnderPath(tmpl, "/data/"))
	assert.Equal(t, &mntTrailing, FindVolumeMountNestedUnderPath(tmpl, "/logs"))

	// Exact match is NOT reported (that's the ordinary overlap case).
	assert.Nil(t, FindVolumeMountNestedUnderPath(tmpl, "/data/shared"))

	// path inside a mount is NOT an ancestor.
	assert.Nil(t, FindVolumeMountNestedUnderPath(tmpl, "/data/shared/sub"))

	// Coincidental prefix is not an ancestor.
	assert.Nil(t, FindVolumeMountNestedUnderPath(tmpl, "/dat"))

	// Unrelated path.
	assert.Nil(t, FindVolumeMountNestedUnderPath(tmpl, "/tmp"))
}

func TestUnknownFieldEnforcerForWorkflowStep(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	_, err := SplitWorkflowYAMLFile(ctx, []byte(validWf), false)
	require.NoError(t, err)

	_, err = SplitWorkflowYAMLFile(ctx, []byte(invalidWf), false)
	require.EqualError(t, err, `json: unknown field "doesNotExist"`)
}

func TestParseObjects(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	assert.Len(t, ParseObjects(ctx, []byte(validWf), false), 1)

	res := ParseObjects(ctx, []byte(invalidWf), false)
	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Object)
	require.EqualError(t, res[0].Err, "json: unknown field \"doesNotExist\"")

	invalidObj := []byte(`<div class="blah" style="display: none; outline: none;" tabindex="0"></div>`)
	res = ParseObjects(ctx, invalidObj, false)
	// the document cannot be parsed into a Kubernetes object, so it is returned
	// with a nil object and the error instead of being logged and dropped (#9550)
	assert.Len(t, res, 1)
	assert.Nil(t, res[0].Object)
	assert.Error(t, res[0].Err)
}

func TestGetTemplateHolderString(t *testing.T) {
	assert.Equal(t, "*v1alpha1.DAGTask invalid (https://argo-workflows.readthedocs.io/en/latest/templates/)", GetTemplateHolderString(&wfv1.DAGTask{}))
	assert.Equal(t, "*v1alpha1.DAGTask inlined", GetTemplateHolderString(&wfv1.DAGTask{Inline: &wfv1.Template{}}))
	assert.Equal(t, "*v1alpha1.DAGTask (foo)", GetTemplateHolderString(&wfv1.DAGTask{Template: "foo"}))
	assert.Equal(t, "*v1alpha1.DAGTask (foo/bar#false)", GetTemplateHolderString(&wfv1.DAGTask{TemplateRef: &wfv1.TemplateRef{
		Name:     "foo",
		Template: "bar",
	}}))
	assert.Equal(t, "*v1alpha1.DAGTask (foo/bar#true)", GetTemplateHolderString(&wfv1.DAGTask{TemplateRef: &wfv1.TemplateRef{
		Name:         "foo",
		Template:     "bar",
		ClusterScope: true,
	}}))
}

func TestIsDone(t *testing.T) {
	assert.False(t, IsDone(&unstructured.Unstructured{}))
	assert.True(t, IsDone(&unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"labels": map[string]any{
				LabelKeyCompleted: "true",
			},
		},
	}}))
	assert.False(t, IsDone(&unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"labels": map[string]any{
				LabelKeyCompleted:               "true",
				LabelKeyWorkflowArchivingStatus: "Pending",
			},
		},
	}}))
}

func TestSubstituteConfigMapKeyRefParam(t *testing.T) {
	globalParams := map[string]any{
		"workflow.parameters.name": "simple-parameters",
		"workflow.parameters.key":  "msg",
	}
	tests := []struct {
		name                 string
		configMapKeyRefParam string
		expected             string
		expectedErr          string
	}{
		{
			name:                 "No string templating",
			configMapKeyRefParam: "simple-parameters",
			expected:             "simple-parameters",
			expectedErr:          "",
		},
		{
			name:                 "Simple template",
			configMapKeyRefParam: "{{ workflow.parameters.name }}",
			expected:             "simple-parameters",
			expectedErr:          "",
		},
		{
			name:                 "Simple template with prefix and suffix",
			configMapKeyRefParam: "prefix-{{ workflow.parameters.name }}-suffix",
			expected:             "prefix-simple-parameters-suffix",
			expectedErr:          "",
		},
		{
			name:                 "Expression template",
			configMapKeyRefParam: "{{=upper(workflow.parameters.key)}}",
			expected:             "MSG",
			expectedErr:          "",
		},
		{
			name:                 "Simple template referencing nonexistent param",
			configMapKeyRefParam: "prefix-{{ workflow.parameters.bad }}",
			expected:             "",
			expectedErr:          "failed to substitute configMapKeyRef: failed to resolve {{ workflow.parameters.bad }}",
		},
		{
			name:                 "Expression template with invalid expression",
			configMapKeyRefParam: "{{=!}}",
			expected:             "",
			expectedErr:          "failed to substitute configMapKeyRef: failed to evaluate expression: unexpected token EOF (1:1)\n | !\n | ^",
		},
		{
			name:                 "Malformed template",
			configMapKeyRefParam: "{{ workflow.parameters.bad",
			expected:             "",
			expectedErr:          "Cannot find end tag=\"}}\" in the template=\"{{ workflow.parameters.bad\" starting from \" workflow.parameters.bad\"",
		},
	}

	ctx := logging.TestContext(t.Context())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := substituteConfigMapKeyRefParam(ctx, tt.configMapKeyRefParam, globalParams)
			assert.Equal(t, tt.expected, result)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOverridableDefaultInputArts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := wfv1.Template{}
	tmpl.Name = "artifact-printing"

	art := wfv1.Artifact{}
	art.Name = "overridable-art"
	rawArt := wfv1.RawArtifact{}
	rawArt.Data = "default contents"
	art.Raw = &rawArt
	tmpl.Inputs.Artifacts = []wfv1.Artifact{art}

	inputs := wfv1.Inputs{}

	inputArt := wfv1.Artifact{}
	inputArt.Name = art.Name
	inputRawArt := wfv1.RawArtifact{}
	inputRawArt.Data = "replacement contents"
	inputArt.Raw = &inputRawArt

	inputs.Artifacts = []wfv1.Artifact{}

	globalParams := make(map[string]string)
	localParams := make(map[string]string)

	newTmpl, err := ProcessArgs(ctx, &tmpl, &inputs, globalParams, localParams, false, "", nil)
	require.NoError(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Artifacts[0].Raw.Data, rawArt.Data)

	inputs.Artifacts = []wfv1.Artifact{inputArt}
	newTmpl, err = ProcessArgs(ctx, &tmpl, &inputs, globalParams, localParams, false, "", nil)
	require.NoError(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Artifacts[0].Raw.Data, inputRawArt.Data)
}

type mockConfigMapStore struct {
	getByKey func(key string) (any, bool, error)
}

func (cs mockConfigMapStore) GetByKey(key string) (any, bool, error) {
	return cs.getByKey(key)
}

func TestOverridableTemplateInputParamsValue(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := wfv1.Template{}
	tmpl.Name = "artifact-printing"

	paramName := "value-from-param"

	overrideConfigMapName := "override-config-map-name"
	overrideConfigMapKey := "override-config-map-key"
	overrideConfigMapValue := "override-config-map-value"

	configMapStore := mockConfigMapStore{}
	configMapStore.getByKey = func(key string) (any, bool, error) {
		return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{LabelKeyConfigMapType: LabelValueTypeConfigMapParameter}},
			Data: map[string]string{overrideConfigMapKey: overrideConfigMapValue},
		}, true, nil
	}

	tmpl.Inputs.Parameters = []wfv1.Parameter{{Name: paramName, Value: wfv1.AnyStringPtr("abc")}}

	valueArgs := wfv1.Inputs{Parameters: []wfv1.Parameter{{Name: paramName, Value: wfv1.AnyStringPtr("override")}}}
	valueFromArgs := wfv1.Inputs{Parameters: []wfv1.Parameter{{Name: paramName, ValueFrom: &wfv1.ValueFrom{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: overrideConfigMapName,
		},
		Key: overrideConfigMapKey,
	}}}}}

	globalParams := make(map[string]string)
	localParams := make(map[string]string)

	newTmpl, err := ProcessArgs(ctx, &tmpl, &valueArgs, globalParams, localParams, false, "", configMapStore)
	require.NoError(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Parameters[0].Value.String(), valueArgs.Parameters[0].Value.String())

	newTmpl, err = ProcessArgs(ctx, &tmpl, &valueFromArgs, globalParams, localParams, false, "", configMapStore)
	require.NoError(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Parameters[0].Value.String(), overrideConfigMapValue)
}

func TestOverridableTemplateInputParamsValueFrom(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	tmpl := wfv1.Template{}
	tmpl.Name = "artifact-printing"

	paramName := "value-from-param"

	configMapName := "config-map-name"
	configMapKey := "config-map-key"
	configMapValue := "config-map-value"

	overrideConfigMapName := "override-config-map-name"
	overrideConfigMapKey := "override-config-map-key"
	overrideConfigMapValue := "override-config-map-value"

	configMapStore := mockConfigMapStore{}
	configMapStore.getByKey = func(key string) (any, bool, error) {
		return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{LabelKeyConfigMapType: LabelValueTypeConfigMapParameter}},
			Data: map[string]string{configMapKey: configMapValue, overrideConfigMapKey: overrideConfigMapValue},
		}, true, nil
	}

	tmpl.Inputs.Parameters = []wfv1.Parameter{{Name: paramName, ValueFrom: &wfv1.ValueFrom{
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: configMapName,
			},
			Key: configMapKey,
		},
	}}}

	valueArgs := wfv1.Inputs{Parameters: []wfv1.Parameter{{Name: paramName, Value: wfv1.AnyStringPtr("override")}}}
	valueFromArgs := wfv1.Inputs{Parameters: []wfv1.Parameter{{
		Name: paramName,
		ValueFrom: &wfv1.ValueFrom{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: overrideConfigMapName,
				},
				Key: overrideConfigMapKey,
			},
		},
	}}}

	globalParams := map[string]string{paramName: "overrideValue"}
	localParams := make(map[string]string)

	newTmpl, err := ProcessArgs(ctx, &tmpl, &valueArgs, globalParams, localParams, false, "", configMapStore)
	require.NoError(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Parameters[0].Value.String(), valueArgs.Parameters[0].Value.String())

	newTmpl, err = ProcessArgs(ctx, &tmpl, &valueFromArgs, globalParams, localParams, false, "", configMapStore)
	require.NoError(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Parameters[0].Value.String(), overrideConfigMapValue)
}

// TestProcessArgsAbsentOptional pins the three branches of the AbsentOptionalArgumentValue handling:
// the sentinel marks an argument that was a pure reference to a skipped/omitted node's output with no
// producer default, and ProcessArgs must treat it as unsupplied. The terminal "neither" branch must
// NOT look like a missing-variable error, otherwise the controller would requeue forever.
func TestProcessArgsAbsentOptional(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	paramName := "in"

	sentinelArgs := wfv1.Inputs{Parameters: []wfv1.Parameter{
		{Name: paramName, Value: wfv1.AnyStringPtr(AbsentOptionalArgumentValue)},
	}}

	globalParams := make(map[string]string)
	localParams := make(map[string]string)

	configMapKey := "config-map-key"
	configMapValue := "config-map-value"
	configMapStore := mockConfigMapStore{}
	configMapStore.getByKey = func(key string) (any, bool, error) {
		return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{LabelKeyConfigMapType: LabelValueTypeConfigMapParameter}},
			Data: map[string]string{configMapKey: configMapValue},
		}, true, nil
	}

	t.Run("input default applies", func(t *testing.T) {
		tmpl := wfv1.Template{}
		tmpl.Inputs.Parameters = []wfv1.Parameter{{Name: paramName, Default: wfv1.AnyStringPtr("fallback")}}
		newTmpl, err := ProcessArgs(ctx, &tmpl, &sentinelArgs, globalParams, localParams, false, "", configMapStore)
		require.NoError(t, err)
		require.NotNil(t, newTmpl)
		assert.Equal(t, "fallback", newTmpl.Inputs.Parameters[0].Value.String())
	})

	t.Run("input valueFrom applies", func(t *testing.T) {
		tmpl := wfv1.Template{}
		tmpl.Inputs.Parameters = []wfv1.Parameter{{Name: paramName, ValueFrom: &wfv1.ValueFrom{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "config-map-name"},
				Key:                  configMapKey,
			},
		}}}
		newTmpl, err := ProcessArgs(ctx, &tmpl, &sentinelArgs, globalParams, localParams, false, "", configMapStore)
		require.NoError(t, err)
		require.NotNil(t, newTmpl)
		assert.Equal(t, configMapValue, newTmpl.Inputs.Parameters[0].Value.String())
	})

	t.Run("no default nor valueFrom fails terminally without requeue", func(t *testing.T) {
		tmpl := wfv1.Template{}
		tmpl.Inputs.Parameters = []wfv1.Parameter{{Name: paramName}}
		_, err := ProcessArgs(ctx, &tmpl, &sentinelArgs, globalParams, localParams, false, "", configMapStore)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "absent optional")
		// Must NOT match IsMissingVariableErr, which would requeue the node forever.
		assert.False(t, template.IsMissingVariableErr(err))
	})
}

// TestParseObjectsDuplicateKeyIsReported verifies that a document of a known Argo kind
// whose strict parsing fails (here: a duplicate `templates` key, which the non-strict
// unmarshal silently accepts) is returned with a typed object and the error, instead of
// being silently dropped. Silently dropping it made `argo lint` report "no linting
// errors found!" and `argo submit` find nothing to submit (#9550).
func TestParseObjectsDuplicateKeyIsReported(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	duplicateKeyWf := []byte(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: duplicate-key-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: busybox
      command: [cowsay]
  templates:
  - name: other
    container:
      image: busybox
      command: [cowsay]
`)
	res := ParseObjects(ctx, duplicateKeyWf, true)
	require.Len(t, res, 1, "the document must not be silently dropped")
	require.NotNil(t, res[0].Object, "a typed object must be returned so callers can name it")
	require.ErrorContains(t, res[0].Err, `key "templates" already set in map`)
	assert.Equal(t, "duplicate-key-", res[0].Object.GetGenerateName())

	// the same body must be accepted when strict mode is off
	res = ParseObjects(ctx, duplicateKeyWf, false)
	require.Len(t, res, 1)
	require.NoError(t, res[0].Err)

	// SplitWorkflowYAMLFile must propagate the error instead of returning nothing
	_, err := SplitWorkflowYAMLFile(ctx, duplicateKeyWf, true)
	require.ErrorContains(t, err, `key "templates" already set in map`)
}

// TestParseObjectsUnparseableDocumentIsReported verifies that a document which cannot
// be parsed into a Kubernetes object at all is returned with a nil object and the
// error, so linters can report which file failed (previously only logged, and
// swallowed entirely by strict lints) (#9550).
func TestParseObjectsUnparseableDocumentIsReported(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	brokenYAML := []byte(`foo: [
`)
	res := ParseObjects(ctx, brokenYAML, true)
	require.Len(t, res, 1)
	assert.Nil(t, res[0].Object)
	require.ErrorContains(t, res[0].Err, "did not find expected node content")

	// the HTML snippet previously used as a non-YAML fixture is still returned with
	// its error (no kind detected) instead of being logged and dropped
	invalidObj := []byte(`<div class="blah" style="display: none; outline: none;" tabindex="0"></div>`)
	res = ParseObjects(ctx, invalidObj, false)
	require.Len(t, res, 1)
	assert.Nil(t, res[0].Object)
	require.Error(t, res[0].Err)
}

// TestSplitHelpersUnparseableDocumentIsSkipped verifies that a document which cannot
// be parsed into a Kubernetes object at all is logged-and-skipped by the Split helpers
// instead of panicking on a nil object (#9550).
func TestSplitHelpersUnparseableDocumentIsSkipped(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	mixed := []byte(`foo: [
---
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: mixed-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: busybox
      command: [cowsay]
`)

	wfs, err := SplitWorkflowYAMLFile(ctx, mixed, false)
	require.NoError(t, err)
	require.Len(t, wfs, 1)

	wfts, err := SplitWorkflowTemplateYAMLFile(ctx, mixed, false)
	require.NoError(t, err)
	assert.Empty(t, wfts)

	cwfs, err := SplitCronWorkflowYAMLFile(ctx, mixed, false)
	require.NoError(t, err)
	assert.Empty(t, cwfs)

	cwfts, err := SplitClusterWorkflowTemplateYAMLFile(ctx, mixed, false)
	require.NoError(t, err)
	assert.Empty(t, cwfts)
}

// TestSplitWorkflowTemplateDuplicateKeyIsReported verifies the strict duplicate-key
// error is propagated by every Split helper for its own kind (#9550).
func TestSplitWorkflowTemplateDuplicateKeyIsReported(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	duplicateKeyWft := []byte(`apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: duplicate-key
spec:
  templates:
  - name: a
    container:
      image: busybox
      command: [cowsay]
  templates:
  - name: b
    container:
      image: busybox
      command: [cowsay]
`)
	_, err := SplitWorkflowTemplateYAMLFile(ctx, duplicateKeyWft, true)
	require.ErrorContains(t, err, `key "templates" already set in map`)

	// non-strict mode accepts the document
	wfts, err := SplitWorkflowTemplateYAMLFile(ctx, duplicateKeyWft, false)
	require.NoError(t, err)
	require.Len(t, wfts, 1)
}

// TestSplitCronWorkflowDuplicateKeyIsReported verifies the strict duplicate-key error
// is propagated for CronWorkflows too (#9550).
func TestSplitCronWorkflowDuplicateKeyIsReported(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	duplicateKeyCwf := []byte(`apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: duplicate-key
spec:
  schedule: "* * * * *"
  concurrencyPolicy: Allow
  workflowMetadata:
    labels:
      example: test
  workflowSpec:
    entrypoint: whalesay
    templates:
    - name: whalesay
      container:
        image: busybox
        command: [cowsay]
    templates:
    - name: other
      container:
        image: busybox
        command: [cowsay]
`)
	_, err := SplitCronWorkflowYAMLFile(ctx, duplicateKeyCwf, true)
	require.ErrorContains(t, err, `key "templates" already set in map`)

	cwfs, err := SplitCronWorkflowYAMLFile(ctx, duplicateKeyCwf, false)
	require.NoError(t, err)
	require.Len(t, cwfs, 1)
}

// TestParseObjectsUnknownKindStrictFailureIsReported verifies that a strict-pass
// failure on a document whose kind is not an Argo kind returns an ObjectMeta shell
// with the error; the Split helpers then skip it as a non-argo object (#9550).
func TestParseObjectsUnknownKindStrictFailureIsReported(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	unknownKind := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: foo
data:
  key: value
  key: duplicated
`)
	res := ParseObjects(ctx, unknownKind, true)
	require.Len(t, res, 1)
	require.ErrorContains(t, res[0].Err, `key "key" already set in map`)
	obj, ok := res[0].Object.(*metav1.ObjectMeta)
	require.True(t, ok, "a non-argo kind must surface as an ObjectMeta shell")
	assert.Equal(t, "foo", obj.Name)

	// Split helpers log-and-skip it as a non-argo object
	wfs, err := SplitWorkflowYAMLFile(ctx, unknownKind, true)
	require.NoError(t, err)
	assert.Empty(t, wfs)
}

// TestParseObjectsKindlessDuplicateKeyIsReported verifies that a document without a
// kind whose strict parse fails (duplicate keys) is returned with a nil object and
// the error (#9550).
func TestParseObjectsKindlessDuplicateKeyIsReported(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	kindlessDup := []byte(`metadata:
  name: foo
data:
  key: value
  key: duplicated
`)
	res := ParseObjects(ctx, kindlessDup, true)
	require.Len(t, res, 1)
	assert.Nil(t, res[0].Object)
	// the strict JSON decode of an ObjectMeta shell fails on the missing kind
	// before the duplicate-key check; the point is that the error surfaces
	require.ErrorContains(t, res[0].Err, "Object 'Kind' is missing")
}

// TestSplitClusterWorkflowTemplateDuplicateKeyIsReported verifies the strict
// duplicate-key error is propagated for ClusterWorkflowTemplates too (#9550).
func TestSplitClusterWorkflowTemplateDuplicateKeyIsReported(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	duplicateKeyCwft := []byte(`apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: duplicate-key
spec:
  templates:
  - name: a
    container:
      image: busybox
      command: [cowsay]
  templates:
  - name: b
    container:
      image: busybox
      command: [cowsay]
`)
	_, err := SplitClusterWorkflowTemplateYAMLFile(ctx, duplicateKeyCwft, true)
	require.ErrorContains(t, err, `key "templates" already set in map`)

	cwfts, err := SplitClusterWorkflowTemplateYAMLFile(ctx, duplicateKeyCwft, false)
	require.NoError(t, err)
	require.Len(t, cwfts, 1)
}

// TestParseObjectsJSONBodyAndStrictSuccess verifies the JSON input branch and a
// successful strict parse of a valid workflow (#9550).
func TestParseObjectsJSONBodyAndStrictSuccess(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	validJSON := []byte(`{"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow","metadata":{"generateName":"json-"},"spec":{"entrypoint":"whalesay","templates":[{"name":"whalesay","container":{"image":"busybox","command":["cowsay"]}}]}}`)
	res := ParseObjects(ctx, validJSON, true)
	require.Len(t, res, 1)
	require.NoError(t, res[0].Err)
	assert.Equal(t, "json-", res[0].Object.GetGenerateName())

	invalidJSON := []byte(`{"kind":"Workflow","spec": BROKEN`)
	res = ParseObjects(ctx, invalidJSON, true)
	require.Len(t, res, 1)
	require.Error(t, res[0].Err)

	// a leading empty document is skipped silently
	leadingEmpty := []byte("---\n" + validJSONWf)
	res = ParseObjects(ctx, leadingEmpty, true)
	require.Len(t, res, 1)
	require.NoError(t, res[0].Err)
}

// validJSONWf is the JSON form of a valid workflow, shared by JSON-branch tests.
var validJSONWf = `{"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow","metadata":{"generateName":"json-"},"spec":{"entrypoint":"whalesay","templates":[{"name":"whalesay","container":{"image":"busybox","command":["cowsay"]}}]}}`

// TestParseObjectsRemainingBranches covers the remaining ParseObjects paths: a JSON
// document that unmarshals to an error, empty YAML documents between separators, a
// kindless document whose strict conversion fails, the WorkflowEventBinding and
// WorkflowTaskSet kinds, and both strict JSON decoding error paths (#9550).
func TestParseObjectsRemainingBranches(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// JSON input that carries a kind but fails the typed decode: the empty typed
	// object is returned together with the error (obj != nil, err != nil branch)
	badTypedJSON := []byte(`{"kind":"Workflow","metadata":{"name":{"nested":"not a string"}}}`)
	res := ParseObjects(ctx, badTypedJSON, true)
	require.Len(t, res, 1)
	require.NotNil(t, res[0].Object)
	require.ErrorContains(t, res[0].Err, "cannot unmarshal object into Go struct field")

	// a JSON document that fails to unmarshal at all: the lenient unstructured
	// decode also fails, no kind is detected, and the document is returned as
	// {nil, err} so the linter reports it (line 33-35 branch, #9550)
	brokenJSON := []byte(`{"kind":"Workflow","spec":[[[`)
	res = ParseObjects(ctx, brokenJSON, true)
	require.Len(t, res, 1)
	assert.Nil(t, res[0].Object)
	require.Error(t, res[0].Err)

	// a JSON document whose kind field itself is malformed: unmarshal errors
	// with the kind unset, so it falls through to the strict conversion which
	// surfaces the type clash as an ObjectMeta strict decoding error
	clashingKind := []byte(`{"kind":{"nested":true}}`)
	res = ParseObjects(ctx, clashingKind, true)
	require.Len(t, res, 1)
	require.NotNil(t, res[0].Object)
	require.ErrorContains(t, res[0].Err, `unknown field "kind"`)

	// a JSON document where the kind is set but the body fails to unmarshal
	// into the typed object: the empty typed object travels with the error
	// (obj != nil, err != nil), which Split helpers then reject by error
	lateTypedFailure := []byte(`{"kind":"Workflow","apiVersion":123}`)
	res = ParseObjects(ctx, lateTypedFailure, true)
	require.Len(t, res, 1)
	require.NotNil(t, res[0].Object)
	require.Error(t, res[0].Err)
	_, splitErr := SplitWorkflowYAMLFile(ctx, lateTypedFailure, true)
	require.Error(t, splitErr)

	// an empty document between separators is skipped silently
	emptyMiddle := []byte(validWf + "\n---\n---\n" + validWf)
	res = ParseObjects(ctx, emptyMiddle, true)
	require.Len(t, res, 2)

	// a kindless document whose strict conversion fails returns {nil, err};
	// YAMLToJSONStrict silently accepts the duplicate key, and the error comes
	// from decoding into an ObjectMeta shell, which requires a kind
	kindlessDup := []byte("a: 1\na: 2\n")
	res = ParseObjects(ctx, kindlessDup, true)
	require.Len(t, res, 1)
	assert.Nil(t, res[0].Object)
	require.ErrorContains(t, res[0].Err, "Object 'Kind' is missing")

	// the remaining kinds produce typed objects
	eventBinding := []byte(`apiVersion: argoproj.io/v1alpha1
kind: WorkflowEventBinding
metadata:
  name: web
spec:
  event:
    selector: "true"
`)
	res = ParseObjects(ctx, eventBinding, true)
	require.Len(t, res, 1)
	require.NoError(t, res[0].Err)
	assert.Equal(t, "web", res[0].Object.GetName())

	taskSet := []byte(`apiVersion: argoproj.io/v1alpha1
kind: WorkflowTaskSet
metadata:
  name: tasks
spec:
  tasks:
    a:
      container:
        image: busybox
        command: [cowsay]
`)
	res = ParseObjects(ctx, taskSet, true)
	require.Len(t, res, 1)
	require.NoError(t, res[0].Err)
	assert.Equal(t, "tasks", res[0].Object.GetName())

	// a strict decoding type error (not a strictness error) still surfaces
	badType := []byte(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: bad-type-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: busybox
      command: [cowsay]
  scheduledTime: not-a-time
`)
	res = ParseObjects(ctx, badType, true)
	require.Len(t, res, 1)
	require.Error(t, res[0].Err)

	// unknown fields in strict mode surface with the object
	unknownField := []byte(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: unknown-field-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: busybox
      command: [cowsay]
  doesNotExist: true
`)
	res = ParseObjects(ctx, unknownField, true)
	require.Len(t, res, 1)
	require.NotNil(t, res[0].Object)
	require.ErrorContains(t, res[0].Err, "unknown field")
}
