package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
