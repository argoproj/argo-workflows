package common

import (
	"bytes"
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// FindOverlappingVolume looks an artifact path, checks if it overlaps with any
// user specified volumeMounts in the template, and returns the deepest volumeMount
// (if any).
func FindOverlappingVolume(tmpl *wfv1.Template, path string) *apiv1.VolumeMount {
	var volMnt *apiv1.VolumeMount
	deepestLen := 0
	for _, mnt := range tmpl.Container.VolumeMounts {
		if !strings.HasPrefix(path, mnt.MountPath) {
			continue
		}
		if len(mnt.MountPath) > deepestLen {
			volMnt = &mnt
			deepestLen = len(mnt.MountPath)
		}
	}
	return volMnt
}

// KillPodContainer is a convenience funtion to issue a kill signal to a container in a pod
// It gives a 15 second grace period before issuing SIGKILL
// NOTE: this only works with containers that have sh
func KillPodContainer(restConfig *rest.Config, namespace string, pod string, container string) error {
	exec, err := ExecPodContainer(restConfig, namespace, pod, container, true, true, "sh", "-c", "kill 1; sleep 15; kill -9 1")
	if err != nil {
		return err
	}
	// Stream will initiate the command. We do want to wait for the result so we launch as a goroutine
	go func() {
		_, _, err := GetExecutorOutput(exec)
		if err != nil {
			log.Warnf("Kill command failed (expected to fail with 137): %v", err)
			return
		}
		log.Infof("Kill of %s (%s) successfully issued", pod, container)
	}()
	return nil
}

// ExecPodContainer runs a command in a container in a pod and returns the remotecommand.Executor
func ExecPodContainer(restConfig *rest.Config, namespace string, pod string, container string, stdout bool, stderr bool, command ...string) (remotecommand.Executor, error) {
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		errors.InternalWrapError(err)
	}

	execRequest := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec").
		Param("container", container).
		Param("stdout", fmt.Sprintf("%v", stdout)).
		Param("stderr", fmt.Sprintf("%v", stderr)).
		Param("tty", "false")

	for _, cmd := range command {
		execRequest = execRequest.Param("command", cmd)
	}

	log.Info(execRequest.URL())
	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execRequest.URL())
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return exec, nil
}

// GetExecutorOutput returns the output of an remotecommand.Executor
func GetExecutorOutput(exec remotecommand.Executor) (string, string, error) {
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer
	err := exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdOut,
		Stderr: &stdErr,
		Tty:    false,
	})
	if err != nil {
		return "", "", errors.InternalWrapError(err)
	}
	return stdOut.String(), stdErr.String(), nil
}
