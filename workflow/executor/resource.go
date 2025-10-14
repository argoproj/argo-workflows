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
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	envutil "github.com/argoproj/argo-workflows/v3/util/env"
	argoerr "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

// ExecResource will run kubectl action against a manifest
func (we *WorkflowExecutor) ExecResource(ctx context.Context, action string, manifestPath string, flags []string) (string, string, string, error) {
	args, err := we.getKubectlArguments(action, manifestPath, flags)
	if err != nil {
		return "", "", "", err
	}

	var out []byte
	err = retry.OnError(retry.DefaultBackoff, func(err error) bool {
		return argoerr.IsTransientErr(ctx, err)
	}, func() error {
		out, err = runKubectl(ctx, args...)
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
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(logging.Fields{"namespace": obj.GetNamespace(), "resource": resourceFullName, "selfLink": selfLink}).Info(ctx, "Resource")
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
		}
		args = append(args, "--type")
		args = append(args, mergeStrategy)

		args = append(args, "--patch-file")
		args = append(args, manifestPath)

		// if there are flags and the manifest has no `kind`, assume: `kubectl patch <kind> <name> --patch-file <path>`
		// json patches also use patch files by definition and so require resource arguments
		// the other form in our case is `kubectl patch -f <path> --patch-file <path>`
		if mergeStrategy == "json" {
			appendFileFlag = false
		} else {
			var obj map[string]interface{}
			err = yaml.Unmarshal(buff, &obj)
			if err != nil {
				return []string{}, errors.New(errors.CodeBadRequest, err.Error())
			}
			if len(flags) != 0 && obj["kind"] == nil {
				appendFileFlag = false
			}
		}
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
	logger := logging.RequireLoggerFromContext(ctx)
	if we.Template.Resource.SuccessCondition == "" && we.Template.Resource.FailureCondition == "" {
		return nil
	}
	var successReqs labels.Requirements
	if we.Template.Resource.SuccessCondition != "" {
		successSelector, err := labels.Parse(we.Template.Resource.SuccessCondition)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "success condition '%s' failed to parse: %v", we.Template.Resource.SuccessCondition, err)
		}
		logger.WithField("conditions", successSelector).Info(ctx, "Waiting for conditions")
		successReqs, _ = successSelector.Requirements()
	}

	var failReqs labels.Requirements
	if we.Template.Resource.FailureCondition != "" {
		failSelector, err := labels.Parse(we.Template.Resource.FailureCondition)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "fail condition '%s' failed to parse: %v", we.Template.Resource.FailureCondition, err)
		}
		logger.WithField("conditions", failSelector).Info(ctx, "Failing for conditions")
		failReqs, _ = failSelector.Requirements()
	}
	err := wait.PollUntilContextCancel(ctx, envutil.LookupEnvDurationOr(ctx, "RESOURCE_STATE_CHECK_INTERVAL", time.Second*5),
		true,
		func(ctx context.Context) (bool, error) {
			isErrRetryable, err := we.checkResourceState(ctx, selfLink, successReqs, failReqs)
			if err == nil {
				logger.WithFields(logging.Fields{"name": resourceName, "namespace": resourceNamespace}).Info(ctx, "Returning from successful wait for resource")
				return true, nil
			}
			if isErrRetryable || argoerr.IsTransientErr(ctx, err) {
				logger.WithFields(logging.Fields{"name": resourceName, "namespace": resourceNamespace, "error": err}).Info(ctx, "Waiting for resource resulted in retryable error")
				return false, nil
			}

			logger.WithField("name", resourceName).WithField("namespace", resourceNamespace).WithError(err).Warn(ctx, "Waiting for resource resulted in non-retryable error")
			return false, err
		})
	if err != nil {
		if wait.Interrupted(err) {
			logger.WithField("name", resourceName).Warn(ctx, "Waiting for resource resulted in timeout due to repeated errors")
		} else {
			logger.WithField("name", resourceName).WithError(err).Warn(ctx, "Waiting for resource resulted in error")
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
	logging.RequireLoggerFromContext(ctx).Debug(ctx, jsonString)
	if !gjson.Valid(jsonString) {
		return false, errors.Errorf(errors.CodeNotFound, "Encountered invalid JSON response when checking resource status. Will not be retried: %q", jsonString)
	}
	return matchConditions(ctx, jsonBytes, successReqs, failReqs)
}

// matchConditions checks whether the returned JSON bytes match success or failure conditions.
func matchConditions(ctx context.Context, jsonBytes []byte, successReqs labels.Requirements, failReqs labels.Requirements) (bool, error) {
	logger := logging.RequireLoggerFromContext(ctx)
	ls := gjsonLabels{json: jsonBytes}
	for _, req := range failReqs {
		failed := req.Matches(ls)
		msg := fmt.Sprintf("failure condition '%s' evaluated %v", req, failed)
		logger.Info(ctx, msg)
		if failed {
			// We return false here to not retry when failure conditions met.
			return false, errors.Errorf(errors.CodeBadRequest, "%s", msg)
		}
	}
	numMatched := 0
	for _, req := range successReqs {
		matched := req.Matches(ls)
		logger.WithFields(logging.Fields{"condition": req, "matched": matched}).Info(ctx, "success condition evaluated")
		if matched {
			numMatched++
		}
	}
	logger.WithFields(logging.Fields{"numMatched": numMatched, "total": len(successReqs)}).Info(ctx, "success conditions matched")
	if numMatched >= len(successReqs) {
		return false, nil
	}

	return true, errors.Errorf(errors.CodeNotFound, "Neither success condition nor the failure condition has been matched. Retrying...")
}

// SaveResourceParameters will save any resource output parameters
func (we *WorkflowExecutor) SaveResourceParameters(ctx context.Context, resourceNamespace string, resourceName string) error {
	logger := logging.RequireLoggerFromContext(ctx)
	if len(we.Template.Outputs.Parameters) == 0 {
		logger.Info(ctx, "No output parameters")
		return nil
	}
	logger.Info(ctx, "Saving resource output parameters")
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
		out, err := runKubectl(ctx, args...)
		logger.WithError(err).WithField("out", string(out)).WithField("args", args).Info(ctx, "kubectl")
		if err != nil {
			return err
		}
		output := string(out)
		if param.ValueFrom.JQFilter != "" {
			output, err = jqFilter(ctx, out, param.ValueFrom.JQFilter)
			logger.WithError(err).WithField("out", string(out)).WithField("filter", param.ValueFrom.JQFilter).Info(ctx, "gojq")
			if err != nil {
				return err
			}
		}

		we.Template.Outputs.Parameters[i].Value = wfv1.AnyStringPtr(output)
		logger.WithFields(logging.Fields{"name": param.Name, "value": output}).Info(ctx, "Saved output parameter")
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

func runKubectl(ctx context.Context, args ...string) ([]byte, error) {
	logging.RequireLoggerFromContext(ctx).Info(ctx, strings.Join(args, " "))
	osArgs := append([]string{}, os.Args...)
	os.Args = args
	defer func() {
		os.Args = osArgs
	}()

	var fatalErr error
	// catch `os.Exit(1)` from kubectl
	kubectlutil.BehaviorOnFatal(func(msg string, code int) {
		fatalErr = errors.New(fmt.Sprint(code), msg)
	})

	var buf bytes.Buffer
	if err := kubectlcmd.NewKubectlCommand(kubectlcmd.KubectlOptions{
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
	if fatalErr != nil {
		return nil, fatalErr
	}
	return buf.Bytes(), nil
}
