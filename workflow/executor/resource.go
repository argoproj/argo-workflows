package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/util/retry"
	"k8s.io/gengo/namer"
	gengotypes "k8s.io/gengo/types"
	kubectlcmd "k8s.io/kubectl/pkg/cmd"
	kubectlutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	envutil "github.com/argoproj/argo-workflows/v3/util/env"
	argoerr "github.com/argoproj/argo-workflows/v3/util/errors"
)

// ExecResource will run kubectl action against a manifest
func (we *WorkflowExecutor) ExecResource(action string, manifestPath string, flags []string) (string, string, string, error) {
	args, err := we.getKubectlArguments(action, manifestPath, flags)
	if err != nil {
		return "", "", "", err
	}

	var out []byte
	err = retry.OnError(retry.DefaultBackoff, argoerr.IsTransientErr, func() error {
		out, err = runKubectl(args...)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if exErr, ok := err.(*exec.ExitError); ok {
			errMsg := strings.TrimSpace(string(exErr.Stderr))
			err = errors.Wrap(err, errors.CodeBadRequest, errMsg)
		} else {
			err = errors.Wrap(err, errors.CodeBadRequest, err.Error())
		}
		err = errors.Wrap(err, errors.CodeBadRequest, "no more retries "+err.Error())
		return "", "", "", err
	}
	if action == "delete" {
		return "", "", "", nil
	}
	if action == "get" && len(out) == 0 {
		return "", "", "", nil
	}
	obj := unstructured.Unstructured{}
	err = json.Unmarshal(out, &obj)
	if err != nil {
		return "", "", "", err
	}
	resourceGroup := obj.GroupVersionKind().Group
	resourceName := obj.GetName()
	resourceKind := obj.GroupVersionKind().Kind
	if resourceName == "" || resourceKind == "" {
		return "", "", "", errors.New(errors.CodeBadRequest, "Kind and name are both required but at least one of them is missing from the manifest")
	}
	resourceFullName := fmt.Sprintf("%s.%s/%s", strings.ToLower(resourceKind), resourceGroup, resourceName)
	selfLink := inferObjectSelfLink(obj)
	log.Infof("Resource: %s/%s. SelfLink: %s", obj.GetNamespace(), resourceFullName, selfLink)
	return obj.GetNamespace(), resourceFullName, selfLink, nil
}

func inferObjectSelfLink(obj unstructured.Unstructured) string {
	gvk := obj.GroupVersionKind()
	// This is the best guess we can do here and is what `kubectl` uses under the hood. Hopefully future versions of the
	// REST client would remove the need to infer the plural name.
	lowercaseNamer := namer.NewAllLowercasePluralNamer(map[string]string{})
	pluralName := lowercaseNamer.Name(&gengotypes.Type{Name: gengotypes.Name{
		Name: gvk.Kind,
	}})

	var selfLinkPrefix string
	if gvk.Group == "" {
		selfLinkPrefix = "api"
	} else {
		selfLinkPrefix = "apis"
	}
	// We cannot use `obj.GetSelfLink()` directly since it is deprecated and will be removed after Kubernetes 1.21: https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/1164-remove-selflink
	var selfLink string
	if obj.GetNamespace() == "" {
		selfLink = fmt.Sprintf("%s/%s/%s/%s", selfLinkPrefix, obj.GetAPIVersion(), pluralName, obj.GetName())
	} else {
		selfLink = fmt.Sprintf("%s/%s/namespaces/%s/%s/%s",
			selfLinkPrefix, obj.GetAPIVersion(), obj.GetNamespace(), pluralName, obj.GetName())
	}
	return selfLink
}

func (we *WorkflowExecutor) getKubectlArguments(action string, manifestPath string, flags []string) ([]string, error) {
	buff, err := os.ReadFile(filepath.Clean(manifestPath))
	if err != nil {
		return []string{}, errors.New(errors.CodeBadRequest, err.Error())
	}
	if len(buff) == 0 && len(flags) == 0 {
		return []string{}, errors.New(errors.CodeBadRequest, "Must provide at least one of flags or manifest.")
	}

	args := []string{
		"kubectl",
		action,
	}
	output := "json"

	if action == "delete" {
		args = append(args, "--ignore-not-found")
		output = "name"
	}

	appendFileFlag := true
	if action == "patch" {
		mergeStrategy := "strategic"
		if we.Template.Resource.MergeStrategy != "" {
			mergeStrategy = we.Template.Resource.MergeStrategy
			if mergeStrategy == "json" {
				// Action "patch" require flag "-p" with resource arguments.
				// But kubectl disallow specify both "-f" flag and resource arguments.
				// Flag "-f" should be excluded for action "patch" here if it's a json patch.
				appendFileFlag = false
			}
		}

		args = append(args, "--type")
		args = append(args, mergeStrategy)

		args = append(args, "-p")
		args = append(args, string(buff))
	}

	if len(flags) != 0 {
		args = append(args, flags...)
	}

	if len(buff) != 0 && appendFileFlag {
		args = append(args, "-f")
		args = append(args, manifestPath)
	}
	args = append(args, "-o")
	args = append(args, output)

	return args, nil
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
func (we *WorkflowExecutor) WaitResource(ctx context.Context, resourceNamespace, resourceName, selfLink string) error {
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
	err := wait.PollImmediateInfinite(envutil.LookupEnvDurationOr("RESOURCE_STATE_CHECK_INTERVAL", time.Second*5),
		func() (bool, error) {
			isErrRetryable, err := we.checkResourceState(ctx, selfLink, successReqs, failReqs)
			if err == nil {
				log.Infof("Returning from successful wait for resource %s in namespace %s", resourceName, resourceNamespace)
				return true, nil
			}
			if isErrRetryable || argoerr.IsTransientErr(err) {
				log.Infof("Waiting for resource %s in namespace %s resulted in retryable error: %v", resourceName, resourceNamespace, err)
				return false, nil
			}

			log.Warnf("Waiting for resource %s in namespace %s resulted in non-retryable error: %v", resourceName, resourceNamespace, err)
			return false, err
		})
	if err != nil {
		if err == wait.ErrWaitTimeout {
			log.Warnf("Waiting for resource %s resulted in timeout due to repeated errors", resourceName)
		} else {
			log.Warnf("Waiting for resource %s resulted in error %v", resourceName, err)
		}
		return err
	}
	return nil
}

// checkResourceState performs resource status checking and then waiting on json reading.
// The returning boolean indicates whether we should retry.
func (we *WorkflowExecutor) checkResourceState(ctx context.Context, selfLink string, successReqs labels.Requirements, failReqs labels.Requirements) (bool, error) {
	request := we.RESTClient.Get().RequestURI(selfLink)
	stream, err := request.Stream(ctx)

	if err != nil {
		err = errors.Cause(err)
		if apierr.IsNotFound(err) {
			return false, errors.Errorf(errors.CodeNotFound, "The resource has been deleted while its status was still being checked. Will not be retried: %v", err)
		}
		return false, err
	}

	defer func() { _ = stream.Close() }()
	jsonBytes, err := io.ReadAll(stream)
	if err != nil {
		return false, err
	}
	jsonString := string(jsonBytes)
	log.Debug(jsonString)
	if !gjson.Valid(jsonString) {
		return false, errors.Errorf(errors.CodeNotFound, "Encountered invalid JSON response when checking resource status. Will not be retried: %q", jsonString)
	}
	return matchConditions(jsonBytes, successReqs, failReqs)
}

// matchConditions checks whether the returned JSON bytes match success or failure conditions.
func matchConditions(jsonBytes []byte, successReqs labels.Requirements, failReqs labels.Requirements) (bool, error) {
	ls := gjsonLabels{json: jsonBytes}
	for _, req := range failReqs {
		failed := req.Matches(ls)
		msg := fmt.Sprintf("failure condition '%s' evaluated %v", req, failed)
		log.Infof(msg)
		if failed {
			// We return false here to not retry when failure conditions met.
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

	return true, errors.Errorf(errors.CodeNotFound, "Neither success condition nor the failure condition has been matched. Retrying...")
}

// SaveResourceParameters will save any resource output parameters
func (we *WorkflowExecutor) SaveResourceParameters(ctx context.Context, resourceNamespace string, resourceName string) error {
	if len(we.Template.Outputs.Parameters) == 0 {
		log.Infof("No output parameters")
		return nil
	}
	log.Infof("Saving resource output parameters")
	for i, param := range we.Template.Outputs.Parameters {
		if param.ValueFrom == nil {
			continue
		}
		if resourceNamespace == "" && resourceName == "" {
			output := ""
			if param.ValueFrom.Default != nil {
				output = param.ValueFrom.Default.String()
			}
			we.Template.Outputs.Parameters[i].Value = wfv1.AnyStringPtr(output)
			continue
		}
		outputFormat := ""
		if param.ValueFrom.JSONPath != "" {
			outputFormat = fmt.Sprintf("jsonpath=%s", param.ValueFrom.JSONPath)
		} else if param.ValueFrom.JQFilter != "" {
			outputFormat = "json"
		} else {
			continue
		}
		args := []string{"kubectl", "-n", resourceNamespace, "get", resourceName, "-o", outputFormat}
		out, err := runKubectl(args...)
		log.WithError(err).WithField("out", string(out)).WithField("args", args).Info("kubectl")
		if err != nil {
			return err
		}
		output := string(out)
		if param.ValueFrom.JQFilter != "" {
			output, err = jqFilter(ctx, out, param.ValueFrom.JQFilter)
			log.WithError(err).WithField("out", string(out)).WithField("filter", param.ValueFrom.JQFilter).Info("gojq")
			if err != nil {
				return err
			}
		}

		we.Template.Outputs.Parameters[i].Value = wfv1.AnyStringPtr(output)
		log.Infof("Saved output parameter: %s, value: %s", param.Name, output)
	}
	err := we.ReportOutputs(ctx, nil)
	return err
}

func jqFilter(ctx context.Context, input []byte, filter string) (string, error) {
	var v interface{}
	if err := json.Unmarshal(input, &v); err != nil {
		return "", err
	}
	q, err := gojq.Parse(filter)
	if err != nil {
		return "", err
	}
	iter := q.RunWithContext(ctx, v)
	var buf strings.Builder
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return "", err
		}
		if s, ok := v.(string); ok {
			buf.WriteString(s)
		} else {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			buf.Write(b)
		}
		buf.WriteString("\n")
	}
	return strings.TrimSpace(buf.String()), nil
}

func runKubectl(args ...string) ([]byte, error) {
	log.Info(strings.Join(args, " "))
	osArgs := append([]string{}, os.Args...)
	os.Args = args
	defer func() {
		os.Args = osArgs
	}()

	var buf bytes.Buffer
	var err error
	// catch `os.Exit(1)` from kubectl
	kubectlutil.BehaviorOnFatal(func(msg string, code int) {
		log.Info("fatal error: %s", msg)
		err = errors.New(string(code), msg)
	})
	if err = kubectlcmd.NewKubectlCommand(kubectlcmd.KubectlOptions{
		Arguments: args,
		// TODO(vadasambar): use `DefaultConfigFlags` variable from upstream
		// as value for `ConfigFlags` once https://github.com/kubernetes/kubernetes/pull/120024 is merged
		ConfigFlags: genericclioptions.NewConfigFlags(true).
			WithDeprecatedPasswordFlag().
			WithDiscoveryBurst(300).
			WithDiscoveryQPS(50.0),
		IOStreams: genericclioptions.IOStreams{Out: &buf, ErrOut: os.Stderr},
	}).Execute(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
