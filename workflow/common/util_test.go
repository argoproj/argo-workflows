package common

import (
	"testing"

	"github.com/valyala/fasttemplate"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// TestFindOverlappingVolume tests logic of TestFindOverlappingVolume
func TestFindOverlappingVolume(t *testing.T) {
	volMnt := corev1.VolumeMount{
		Name:      "workdir",
		MountPath: "/user-mount",
	}
	templateWithVolMount := &wfv1.Template{
		Container: &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{volMnt},
		},
	}
	assert.Equal(t, &volMnt, FindOverlappingVolume(templateWithVolMount, "/user-mount"))
	assert.Equal(t, &volMnt, FindOverlappingVolume(templateWithVolMount, "/user-mount/subdir"))
	assert.Nil(t, FindOverlappingVolume(templateWithVolMount, "/user-mount-coincidental-prefix"))
}

func TestUnknownFieldEnforcerForWorkflowStep(t *testing.T) {
	validWf := `apiVersion: argoproj.io/v1alpha1
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
	_, err := SplitWorkflowYAMLFile([]byte(validWf), false)
	assert.NoError(t, err)

	invalidWf := `apiVersion: argoproj.io/v1alpha1
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
	_, err = SplitWorkflowYAMLFile([]byte(invalidWf), false)
	assert.EqualError(t, err, `error unmarshaling JSON: while decoding JSON: json: unknown field "doesNotExist"`)
}

func TestDeletePod(t *testing.T) {
	kube := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: v1.ObjectMeta{Name: "my-pod", Namespace: "my-ns"},
	})
	t.Run("Exists", func(t *testing.T) {
		err := DeletePod(kube, "my-pod", "my-ms")
		assert.NoError(t, err)
	})
	t.Run("NotExists", func(t *testing.T) {
		err := DeletePod(kube, "not-exists", "my-ms")
		assert.NoError(t, err)
	})
}

func TestNestedReplaceString(t *testing.T) {

	replaceMap := map[string]string{"inputs.parameters.message": "hello world"}

	test := `{{- with secret "{{inputs.parameters.message}}" -}}
    {{ .Data.data.gitcreds }}
  {{- end }}`
	fstTmpl, err := fasttemplate.NewTemplate(test, "{{", "}}")
	if assert.NoError(t, err) {
		replacement, err := Replace(fstTmpl, replaceMap, true)
		if assert.NoError(t, err) {
			assert.Equal(t, "{{- with secret \"hello world\" -}}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", replacement)
		}
	}

	test = `{{- with {{ secret "{{inputs.parameters.message}}" -}}
    {{ .Data.data.gitcreds }}
  {{- end }}`
	fstTmpl, err = fasttemplate.NewTemplate(test, "{{", "}}")
	if assert.NoError(t, err) {
		replacement, err := Replace(fstTmpl, replaceMap, true)
		if assert.NoError(t, err) {
			assert.Equal(t, "{{- with {{ secret \"hello world\" -}}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", replacement)
		}
	}

	test = `{{- with {{ secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`
	fstTmpl, err = fasttemplate.NewTemplate(test, "{{", "}}")
	if assert.NoError(t, err) {
		replacement, err := Replace(fstTmpl, replaceMap, true)
		if assert.NoError(t, err) {
			assert.Equal(t, "{{- with {{ secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", replacement)
		}
	}

	test = `{{- with secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`
	fstTmpl, err = fasttemplate.NewTemplate(test, "{{", "}}")
	if assert.NoError(t, err) {
		replacement, err := Replace(fstTmpl, replaceMap, true)
		if assert.NoError(t, err) {
			assert.Equal(t, "{{- with secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", replacement)
		}
	}

	test = `{{- with {{ {{ }} secret "{{inputs.parameters.message}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`
	fstTmpl, err = fasttemplate.NewTemplate(test, "{{", "}}")
	if assert.NoError(t, err) {
		replacement, err := Replace(fstTmpl, replaceMap, true)
		if assert.NoError(t, err) {
			assert.Equal(t, "{{- with {{ {{ }} secret \"hello world\" -}} }}\n    {{ .Data.data.gitcreds }}\n  {{- end }}", replacement)
		}
	}

	test = `{{- with {{ {{ }} secret "{{does-not-exist}}" -}} }}
    {{ .Data.data.gitcreds }}
  {{- end }}`
	fstTmpl, err = fasttemplate.NewTemplate(test, "{{", "}}")
	if assert.NoError(t, err) {
		replacement, err := Replace(fstTmpl, replaceMap, true)
		if assert.NoError(t, err) {
			assert.Equal(t, test, replacement)
		}
	}
}

func TestReplaceStringWithWhiteSpace(t *testing.T) {

	replaceMap := map[string]string{"inputs.parameters.message": "hello world"}

	test := `{{ inputs.parameters.message }}`
	fstTmpl, err := fasttemplate.NewTemplate(test, "{{", "}}")
	if assert.NoError(t, err) {
		replacement, err := Replace(fstTmpl, replaceMap, true)
		if assert.NoError(t, err) {
			assert.Equal(t, "hello world", replacement)
		}
	}
}
