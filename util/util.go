package util

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/retry"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
)

type Closer interface {
	Close() error
}

// Close is a convenience function to close a object that has a Close() method, ignoring any errors
// Used to satisfy errcheck lint
func Close(c Closer) {
	_ = c.Close()
}

// GetSecrets retrieves a secret value and memoizes the result
func GetSecrets(ctx context.Context, clientSet kubernetes.Interface, namespace, name, key string) ([]byte, error) {
	secretsIf := clientSet.CoreV1().Secrets(namespace)
	var secret *apiv1.Secret
	err := waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
		var err error
		secret, err = secretsIf.Get(ctx, name, metav1.GetOptions{})
		return !errorsutil.IsTransientErr(err), err
	})
	if err != nil {
		return []byte{}, errors.InternalWrapError(err)
	}
	val, ok := secret.Data[key]
	if !ok {
		return []byte{}, errors.Errorf(errors.CodeBadRequest, "secret '%s' does not have the key '%s'", name, key)
	}
	return val, nil
}

// Write the Terminate message in pod spec
func WriteTerminateMessage(message string) {
	err := os.WriteFile("/dev/termination-log", []byte(message), 0o600)
	if err != nil {
		println("unable to write termination log: " + err.Error())
	}
}

// Merge the two parameters Slice
// Merge the slices based on arguments order (first is high priority).
func MergeParameters(params ...[]wfv1.Parameter) []wfv1.Parameter {
	var resultParams []wfv1.Parameter
	passedParams := make(map[string]bool)
	for _, param := range params {
		for _, item := range param {
			if _, ok := passedParams[item.Name]; ok {
				continue
			}
			resultParams = append(resultParams, item)
			passedParams[item.Name] = true
		}
	}
	return resultParams
}

// MergeArtifacts merges artifact argument slices
// Merge the slices based on arguments order (first is high priority).
func MergeArtifacts(artifactSlices ...[]wfv1.Artifact) []wfv1.Artifact {
	var result []wfv1.Artifact
	alreadyMerged := make(map[string]bool)
	for _, artifacts := range artifactSlices {
		for _, item := range artifacts {
			if !alreadyMerged[item.Name] {
				result = append(result, item)
				alreadyMerged[item.Name] = true
			}
		}
	}
	return result
}

func RecoverIndexFromNodeName(name string) int {
	startIndex := strings.Index(name, "(")
	endIndex := strings.Index(name, ":")
	if startIndex < 0 || endIndex < 0 {
		return -1
	}
	out, err := strconv.Atoi(name[startIndex+1 : endIndex])
	if err != nil {
		return -1
	}
	return out
}

func GenerateFieldSelectorFromWorkflowName(wfName string) string {
	result := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", wfName)).String()
	compare := RecoverWorkflowNameFromSelectorStringIfAny(result)
	if wfName != compare {
		panic(fmt.Sprintf("Could not recover field selector from workflow name. Expected '%s' but got '%s'\n", wfName, compare))
	}
	return result
}

func RecoverWorkflowNameFromSelectorStringIfAny(selector string) string {
	const tag = "metadata.name="
	if starts := strings.Index(selector, tag); starts > -1 {
		suffix := selector[starts+len(tag):]
		if ends := strings.Index(suffix, ","); ends > -1 {
			return strings.TrimSpace(suffix[:ends])
		}
		return strings.TrimSpace(suffix)
	}
	return ""
}

// getDeletePropagation return the default or configured DeletePropagation policy
func GetDeletePropagation() *metav1.DeletionPropagation {
	propagationPolicy := metav1.DeletePropagationBackground
	envVal, ok := os.LookupEnv("WF_DEL_PROPAGATION_POLICY")
	if ok && envVal != "" {
		propagationPolicy = metav1.DeletionPropagation(envVal)
	}
	return &propagationPolicy
}

func RemoveFinalizer(finalizers []string, targetFinalizer string) []string {
	var updatedFinalizers []string
	for _, finalizer := range finalizers {
		if finalizer != targetFinalizer {
			updatedFinalizers = append(updatedFinalizers, finalizer)
		}
	}
	return updatedFinalizers
}
