package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/util/retry"
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
func GetSecrets(clientSet kubernetes.Interface, namespace, name, key string) ([]byte, error) {

	secretsIf := clientSet.CoreV1().Secrets(namespace)
	var secret *apiv1.Secret
	var err error
	_ = wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		secret, err = secretsIf.Get(name, metav1.GetOptions{})
		if err != nil {
			log.Warnf("Failed to get secret '%s': %v", name, err)
			if !retry.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
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
func WriteTeriminateMessage(message string) {
	err := ioutil.WriteFile("/dev/termination-log", []byte(message), 0644)
	if err != nil {
		panic(err)
	}
}

// In versions <v2.5.0 we allowed the step.arguments.parameters.value field to be a number as well as a string.
// In version >=v2.5.0 a custom unmarshaller was implemented that enforced stricter type checking, but we still want to
// add backwards-compatibility to workflows. Here we manually attempt to convert a number to a string field
func MustCastParameterValuesToString(candidate []map[string]interface{}) ([]byte, error) {
	type replaceMapEntry struct {
		stepIndex  int
		paramIndex int
		value      string
	}
	var replaceMap []replaceMapEntry

	// See if any step.arguments.parameters.value are present
	for stepIndex, step := range candidate {
		if args, ok := step["arguments"]; ok {
			if params, ok := args.(map[string]interface{})["parameters"]; ok {
				parameters := params.([]interface{})
				for paramIndex, parameter := range parameters {
					if value, ok := parameter.(map[string]interface{})["value"]; ok {
						// Value is present, attempt to cast it
						switch value.(type) {
						case int, float32, float64:
							// Cast successful
							replaceMap = append(replaceMap, replaceMapEntry{
								stepIndex:  stepIndex,
								paramIndex: paramIndex,
								value:      fmt.Sprint(value),
							})
						case string:
							// Do nothing, value is already string
						default:
							// Forbidden value type. This should be unreachable at this point
							return nil, fmt.Errorf("invalid value type")
						}
					}
				}
			}
		}
	}

	if len(replaceMap) == 0 {
		// No replacements were found. Return early
		return nil, fmt.Errorf("no replacements were made")
	}

	// Make replacements
	for _, replace := range replaceMap {
		candidate[replace.stepIndex]["arguments"].(map[string]interface{})["parameters"].([]interface{})[replace.paramIndex].(map[string]interface{})["value"] = replace.value
	}

	strCandidate, innerErr := json.Marshal(candidate)
	if innerErr != nil {
		return nil, innerErr
	}

	return strCandidate, nil
}
