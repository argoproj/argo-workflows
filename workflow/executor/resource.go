package executor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/argoproj/argo/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ExecResource will run kubectl action against a manifest
func (we *WorkflowExecutor) ExecResource(action string, manifestPath string, isDelete bool) (string, string, error) {
	args := []string{
		action,
	}
	output := "json"
	if isDelete {
		args = append(args, "--ignore-not-found")
		output = "name"
	}

	if action == "patch" {
		mergeStrategy := "strategic"
		if we.Template.Resource.MergeStrategy != "" {
			mergeStrategy = we.Template.Resource.MergeStrategy
		}

		args = append(args, "--type")
		args = append(args, mergeStrategy)

		args = append(args, "-p")
		buff, err := ioutil.ReadFile(manifestPath)

		if err != nil {
			return "", "", errors.New(errors.CodeBadRequest, err.Error())
		}

		args = append(args, string(buff))
	}

	args = append(args, "-f")
	args = append(args, manifestPath)
	args = append(args, "-o")
	args = append(args, output)
	cmd := exec.Command("kubectl", args...)
	log.Info(strings.Join(cmd.Args, " "))
	out, err := cmd.Output()
	if err != nil {
		exErr := err.(*exec.ExitError)
		errMsg := strings.TrimSpace(string(exErr.Stderr))
		return "", "", errors.New(errors.CodeBadRequest, errMsg)
	}
	if action == "delete" {
		return "", "", nil
	}
	obj := unstructured.Unstructured{}
	err = json.Unmarshal(out, &obj)
	if err != nil {
		return "", "", err
	}
	resourceName := fmt.Sprintf("%s.%s/%s", obj.GroupVersionKind().Kind, obj.GroupVersionKind().Group, obj.GetName())
	log.Infof("%s/%s", obj.GetNamespace(), resourceName)
	return obj.GetNamespace(), resourceName, nil
}

// gjsonLabels is an implementation of labels.Labels interface
// which allows us to take advantage of k8s labels library
// for the purposes of evaluating fail and success conditions
type gjsonLabels struct {
	json []byte
}

// Has returns whether the provided label exists.
func (g gjsonLabels) Has(label string) bool {
	return gjson.GetBytes(g.json, label).Exists()
}

// Get returns the value for the provided label.
func (g gjsonLabels) Get(label string) string {
	return gjson.GetBytes(g.json, label).String()
}

// WaitResource waits for a specific resource to satisfy either the success or failure condition
func (we *WorkflowExecutor) WaitResource(resourceNamespace string, resourceName string) error {
	if we.Template.Resource.SuccessCondition == "" && we.Template.Resource.FailureCondition == "" {
		return nil
	}
	var successReqs labels.Requirements
	if we.Template.Resource.SuccessCondition != "" {
		successSelector, err := labels.Parse(we.Template.Resource.SuccessCondition)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "success condition '%s' failed to parse: %v", we.Template.Resource.SuccessCondition, err)
		}
		log.Infof("Waiting for conditions: %s", successSelector)
		successReqs, _ = successSelector.Requirements()
	}

	var failReqs labels.Requirements
	if we.Template.Resource.FailureCondition != "" {
		failSelector, err := labels.Parse(we.Template.Resource.FailureCondition)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "fail condition '%s' failed to parse: %v", we.Template.Resource.FailureCondition, err)
		}
		log.Infof("Failing for conditions: %s", failSelector)
		failReqs, _ = failSelector.Requirements()
	}

	// Start the condition result reader using PollImmediateInfinite
	// Poll intervall of 5 seconds serves as a backoff intervall in case of immediate result reader failure
	err := wait.PollImmediateInfinite(time.Second*5,
		func() (bool, error) {
			isErrRetry, err := checkResourceState(resourceNamespace, resourceName, successReqs, failReqs)

			if err == nil {
				log.Infof("Returning from successful wait for resource %s", resourceName)
				return true, nil
			}

			if isErrRetry {
				log.Infof("Waiting for resource %s resulted in retryable error %v", resourceName, err)
				return false, nil
			}

			log.Warnf("Waiting for resource %s resulted in non-retryable error %v", resourceName, err)
			return false, err
		})

	if err != nil {
		if err == wait.ErrWaitTimeout {
			log.Warnf("Waiting for resource %s resulted in timeout due to repeated errors", resourceName)
		} else {
			log.Warnf("Waiting for resource %s resulted in error %v", resourceName, err)
		}
	}

	return err
}

// Function to do the kubectl get -w command and then waiting on json reading.
func checkResourceState(resourceNamespace string, resourceName string, successReqs labels.Requirements, failReqs labels.Requirements) (bool, error) {

	cmd, reader, err := startKubectlWaitCmd(resourceNamespace, resourceName)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = cmd.Process.Kill()
	}()

	for {
		jsonBytes, err := readJSON(reader)

		if err != nil {
			resultErr := err
			log.Warnf("Json reader returned error %v. Calling kill (usually superfluous)", err)
			// We don't want to write OS specific code so we don't want to call syscall package code. But that means
			// there is no way to figure out if a process is running or not in an asynchronous manner. exec.Wait will
			// always block and we need to call that to get the exit code of the process. So we will unconditionally
			// call exec.Process.Kill and then assume that wait will not block after that. Two things may happen:
			// 1. Process already exited and kill does nothing (returns error which we ignore) and then we call
			//    Wait and get the proper return value
			// 2. Process is running gets, killed with exec.Process.Kill call and Wait returns an error code and we give up
			//    and don't retry
			_ = cmd.Process.Kill()

			log.Warnf("Command for kubectl get -w for %s exited. Getting return value using Wait", resourceName)
			err = cmd.Wait()
			if err != nil {
				log.Warnf("cmd.Wait for kubectl get -w command for resource %s returned error %v",
					resourceName, err)
				resultErr = err
			} else {
				log.Infof("readJSon failed for resource %s but cmd.Wait for kubectl get -w command did not error", resourceName)
			}
			return true, resultErr
		}

		log.Info(string(jsonBytes))
		ls := gjsonLabels{json: jsonBytes}
		for _, req := range failReqs {
			failed := req.Matches(ls)
			msg := fmt.Sprintf("failure condition '%s' evaluated %v", req, failed)
			log.Infof(msg)
			if failed {
				// TODO: need a better error code instead of BadRequest
				return false, errors.Errorf(errors.CodeBadRequest, msg)
			}
		}
		numMatched := 0
		for _, req := range successReqs {
			matched := req.Matches(ls)
			log.Infof("success condition '%s' evaluated %v", req, matched)
			if matched {
				numMatched++
			}
		}
		log.Infof("%d/%d success conditions matched", numMatched, len(successReqs))
		if numMatched >= len(successReqs) {
			return false, nil
		}
	}
}

// Start Kubectl command Get with -w return error if unable to start command
func startKubectlWaitCmd(resourceNamespace string, resourceName string) (*exec.Cmd, *bufio.Reader, error) {
	args := []string{"get", resourceName, "-w", "-o", "json"}
	if resourceNamespace != "" {
		args = append(args, "-n", resourceNamespace)
	}
	cmd := exec.Command("kubectl", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, errors.InternalWrapError(err)
	}
	reader := bufio.NewReader(stdout)
	log.Info(strings.Join(cmd.Args, " "))
	if err := cmd.Start(); err != nil {
		return nil, nil, errors.InternalWrapError(err)
	}

	return cmd, reader, nil
}

// readJSON reads from a reader line-by-line until it reaches "}\n" indicating end of json
func readJSON(reader *bufio.Reader) ([]byte, error) {
	var buffer bytes.Buffer
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		isDelimiter := len(line) == 2 && line[0] == byte('}')
		line = bytes.TrimSpace(line)
		_, err = buffer.Write(line)
		if err != nil {
			return nil, err
		}
		if isDelimiter {
			break
		}
	}
	return buffer.Bytes(), nil
}

// SaveResourceParameters will save any resource output parameters
func (we *WorkflowExecutor) SaveResourceParameters(resourceNamespace string, resourceName string) error {
	if len(we.Template.Outputs.Parameters) == 0 {
		log.Infof("No output parameters")
		return nil
	}
	log.Infof("Saving resource output parameters")
	for i, param := range we.Template.Outputs.Parameters {
		if param.ValueFrom == nil {
			continue
		}
		var cmd *exec.Cmd
		if param.ValueFrom.JSONPath != "" {
			args := []string{"get", resourceName, "-o", fmt.Sprintf("jsonpath=%s", param.ValueFrom.JSONPath)}
			if resourceNamespace != "" {
				args = append(args, "-n", resourceNamespace)
			}
			cmd = exec.Command("kubectl", args...)
		} else if param.ValueFrom.JQFilter != "" {
			resArgs := []string{resourceName}
			if resourceNamespace != "" {
				resArgs = append(resArgs, "-n", resourceNamespace)
			}
			cmdStr := fmt.Sprintf("kubectl get %s -o json | jq -c '%s'", strings.Join(resArgs, " "), param.ValueFrom.JQFilter)
			cmd = exec.Command("sh", "-c", cmdStr)
		} else {
			continue
		}
		log.Info(cmd.Args)
		out, err := cmd.Output()
		if err != nil {
			if exErr, ok := err.(*exec.ExitError); ok {
				log.Errorf("`%s` stderr:\n%s", cmd.Args, string(exErr.Stderr))
			}
			return errors.InternalWrapError(err)
		}
		output := string(out)
		we.Template.Outputs.Parameters[i].Value = &output
		log.Infof("Saved output parameter: %s, value: %s", param.Name, output)
	}
	err := we.AnnotateOutputs(nil)
	if err != nil {
		return err
	}
	return nil
}
