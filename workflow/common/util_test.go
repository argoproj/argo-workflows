package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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
	validConfigMapRefWf = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-template-configmapkeyselector-substitution
spec:
  entrypoint: whalesay
  arguments:
    parameters:
    - name: name
      value: simple-parameters
    - name: key
      value: msg
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
        valueFrom:
          configMapKeyRef:
            name: "{{ workflow.parameters.name }}"
            key: "{{ workflow.parameters.key }}"
    container:
      image: argoproj/argosay:v2
      args:
        - echo
        - "{{inputs.parameters.message}}"
`
	invalidConfigMapRefWf = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-template-configmapkeyselector-substitution
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
        valueFrom:
          configMapKeyRef:
            name: "{{ workflow.parameters.name }}"
            key: "{{ workflow.parameters.key }}"
    container:
      image: argoproj/argosay:v2
      args:
        - echo
        - "{{inputs.parameters.message}}"
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

func TestUnknownFieldEnforcerForWorkflowStep(t *testing.T) {
	_, err := SplitWorkflowYAMLFile([]byte(validWf), false)
	require.NoError(t, err)

	_, err = SplitWorkflowYAMLFile([]byte(invalidWf), false)
	require.EqualError(t, err, `json: unknown field "doesNotExist"`)
}

func TestParseObjects(t *testing.T) {
	assert.Len(t, ParseObjects([]byte(validWf), false), 1)

	res := ParseObjects([]byte(invalidWf), false)
	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Object)
	require.EqualError(t, res[0].Err, "json: unknown field \"doesNotExist\"")

	invalidObj := []byte(`<div class="blah" style="display: none; outline: none;" tabindex="0"></div>`)
	assert.Empty(t, ParseObjects(invalidObj, false))
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
	assert.True(t, IsDone(&unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				LabelKeyCompleted: "true",
			},
		},
	}}))
	assert.False(t, IsDone(&unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				LabelKeyCompleted:               "true",
				LabelKeyWorkflowArchivingStatus: "Pending",
			},
		},
	}}))
}

func TestSubstituteConfigMapKeyRefParam(t *testing.T) {
	res := ParseObjects([]byte(validConfigMapRefWf), false)
	assert.Len(t, res, 1)

	obj, ok := res[0].Object.(*wfv1.Workflow)
	assert.True(t, ok)
	assert.NotNil(t, obj)

	globalParams := Parameters{
		"workflow.parameters.name": "simple-parameters",
		"workflow.parameters.key":  "msg",
	}

	for _, inParam := range obj.GetTemplateByName("whalesay").Inputs.Parameters {
		cmName, _ := substituteConfigMapKeyRefParam(inParam.ValueFrom.ConfigMapKeyRef.Name, globalParams)
		assert.Equal(t, "simple-parameters", cmName, "it should be equal")

		cmKey, _ := substituteConfigMapKeyRefParam(inParam.ValueFrom.ConfigMapKeyRef.Key, globalParams)
		assert.Equal(t, "msg", cmKey, "it should be equal")
	}
}

func TestSubstituteConfigMapKeyRefParamWithNoParamsDefined(t *testing.T) {
	res := ParseObjects([]byte(invalidConfigMapRefWf), false)
	assert.Len(t, res, 1)

	obj, ok := res[0].Object.(*wfv1.Workflow)
	assert.True(t, ok)
	assert.NotNil(t, obj)

	globalParams := Parameters{}

	for _, inParam := range obj.GetTemplateByName("whalesay").Inputs.Parameters {
		cmName, err := substituteConfigMapKeyRefParam(inParam.ValueFrom.ConfigMapKeyRef.Name, globalParams)
		require.EqualError(t, err, "parameter workflow.parameters.name not found")
		assert.Equal(t, "", cmName)

		cmKey, err := substituteConfigMapKeyRefParam(inParam.ValueFrom.ConfigMapKeyRef.Key, globalParams)
		require.EqualError(t, err, "parameter workflow.parameters.key not found")
		assert.Equal(t, "", cmKey)
	}
}

func TestOverridableDefaultInputArts(t *testing.T) {
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

	newTmpl, err := ProcessArgs(&tmpl, &inputs, globalParams, localParams, false, "", nil)
	require.NoError(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Artifacts[0].Raw.Data, rawArt.Data)

	inputs.Artifacts = []wfv1.Artifact{inputArt}
	newTmpl, err = ProcessArgs(&tmpl, &inputs, globalParams, localParams, false, "", nil)
	require.NoError(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Artifacts[0].Raw.Data, inputRawArt.Data)
}

type mockConfigMapStore struct {
	getByKey func(key string) (interface{}, bool, error)
}

func (cs mockConfigMapStore) GetByKey(key string) (interface{}, bool, error) {
	return cs.getByKey(key)
}

func TestOverridableTemplateInputParamsValue(t *testing.T) {
	tmpl := wfv1.Template{}
	tmpl.Name = "artifact-printing"

	paramName := "value-from-param"

	overrideConfigMapName := "override-config-map-name"
	overrideConfigMapKey := "override-config-map-key"
	overrideConfigMapValue := "override-config-map-value"

	configMapStore := mockConfigMapStore{}
	configMapStore.getByKey = func(key string) (interface{}, bool, error) {
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

	newTmpl, err := ProcessArgs(&tmpl, &valueArgs, globalParams, localParams, false, "", configMapStore)
	require.NoError(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Parameters[0].Value.String(), valueArgs.Parameters[0].Value.String())

	newTmpl, err = ProcessArgs(&tmpl, &valueFromArgs, globalParams, localParams, false, "", configMapStore)
	require.NoError(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Parameters[0].Value.String(), overrideConfigMapValue)
}

func TestOverridableTemplateInputParamsValueFrom(t *testing.T) {
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
	configMapStore.getByKey = func(key string) (interface{}, bool, error) {
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

	newTmpl, err := ProcessArgs(&tmpl, &valueArgs, globalParams, localParams, false, "", configMapStore)
	require.NoError(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Parameters[0].Value.String(), valueArgs.Parameters[0].Value.String())

	newTmpl, err = ProcessArgs(&tmpl, &valueFromArgs, globalParams, localParams, false, "", configMapStore)
	require.NoError(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Parameters[0].Value.String(), overrideConfigMapValue)
}
