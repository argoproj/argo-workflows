package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

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
