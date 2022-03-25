package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"

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
	assert.NoError(t, err)

	_, err = SplitWorkflowYAMLFile([]byte(invalidWf), false)
	assert.EqualError(t, err, `json: unknown field "doesNotExist"`)
}

func TestParseObjects(t *testing.T) {
	assert.Equal(t, 1, len(ParseObjects([]byte(validWf), false)))

	res := ParseObjects([]byte(invalidWf), false)
	assert.Equal(t, 1, len(res))
	assert.NotNil(t, res[0].Object)
	assert.EqualError(t, res[0].Err, "json: unknown field \"doesNotExist\"")

	invalidObj := []byte(`<div class="blah" style="display: none; outline: none;" tabindex="0"></div>`)
	assert.Empty(t, ParseObjects(invalidObj, false))
}

func TestDeletePod(t *testing.T) {
	ctx := context.Background()
	kube := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: v1.ObjectMeta{Name: "my-pod", Namespace: "my-ns"},
	})

	t.Run("Exists", func(t *testing.T) {
		err := DeletePod(ctx, kube, "my-pod", "my-ms")
		assert.NoError(t, err)
	})
	t.Run("NotExists", func(t *testing.T) {
		err := DeletePod(ctx, kube, "not-exists", "my-ms")
		assert.NoError(t, err)
	})
}

func TestGetTemplateHolderString(t *testing.T) {
	assert.Equal(t, "*v1alpha1.DAGTask invalid (https://argoproj.github.io/argo-workflows/templates/)", GetTemplateHolderString(&wfv1.DAGTask{}))
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
	assert.Equal(t, 1, len(res))

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
	assert.Equal(t, 1, len(res))

	obj, ok := res[0].Object.(*wfv1.Workflow)
	assert.True(t, ok)
	assert.NotNil(t, obj)

	globalParams := Parameters{}

	for _, inParam := range obj.GetTemplateByName("whalesay").Inputs.Parameters {
		cmName, err := substituteConfigMapKeyRefParam(inParam.ValueFrom.ConfigMapKeyRef.Name, globalParams)
		assert.Error(t, err)
		assert.EqualError(t, err, "parameter workflow.parameters.name not found")
		assert.Equal(t, "", cmName)

		cmKey, err := substituteConfigMapKeyRefParam(inParam.ValueFrom.ConfigMapKeyRef.Key, globalParams)
		assert.Error(t, err)
		assert.EqualError(t, err, "parameter workflow.parameters.key not found")
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
	assert.Nil(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Artifacts[0].Raw.Data, rawArt.Data)

	inputs.Artifacts = []wfv1.Artifact{inputArt}
	newTmpl, err = ProcessArgs(&tmpl, &inputs, globalParams, localParams, false, "", nil)
	assert.Nil(t, err)
	assert.NotNil(t, newTmpl)
	assert.Equal(t, newTmpl.Inputs.Artifacts[0].Raw.Data, inputRawArt.Data)
}
