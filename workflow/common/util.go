package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/template"
)

// FindOverlappingVolume looks an artifact path, checks if it overlaps with any
// user specified volumeMounts in the template, and returns the deepest volumeMount
// (if any). A return value of nil indicates the path is not under any volumeMount.
func FindOverlappingVolume(tmpl *wfv1.Template, path string) *apiv1.VolumeMount {
	volumeMounts := tmpl.GetVolumeMounts()
	sort.Slice(volumeMounts, func(i, j int) bool {
		return len(volumeMounts[i].MountPath) > len(volumeMounts[j].MountPath)
	})
	for _, mnt := range volumeMounts {
		// path is the mount itself or a descendant of it.
		if _, ok := relWithin(mnt.MountPath, path); ok {
			return &mnt
		}
	}
	return nil
}

// relWithin reports the path of target relative to base, and whether target is
// base itself (rel == ".") or a descendant of it. It uses filepath.Rel rather
// than hand-rolled string prefixing so path separators, trailing slashes and
// "." segments are handled per the host OS — notably so the comparison holds on
// Windows (where the supervisor also runs), where filesystem-resolved paths use
// backslashes. ok is false when target escapes base via ".." or the two cannot
// be related (e.g. different Windows volumes).
func relWithin(base, target string) (rel string, ok bool) {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return rel, false
	}
	return rel, true
}

// FindVolumeMountNestedUnderPath returns the first volume mount whose mount path
// is strictly nested beneath path (i.e. path is a proper ancestor directory of
// the mount). This is the opposite direction to FindOverlappingVolume, which
// finds a mount that *contains* path.
//
// It exists to detect a dangerous input-artifact configuration: an artifact path
// that is an ancestor of a mounted volume (e.g. artifact path /data with a volume
// mounted at /data/shared). In init-less mode the emissary clears art.Path before
// symlinking the artifact into place, and os.RemoveAll on such a path would
// recurse into and destroy the mounted volume. An exact path==mountPath match is
// NOT reported here — that is the ordinary overlap case handled by
// FindOverlappingVolume (the artifact is routed into the volume).
func FindVolumeMountNestedUnderPath(tmpl *wfv1.Template, path string) *apiv1.VolumeMount {
	for _, mnt := range tmpl.GetVolumeMounts() {
		// The mount is strictly beneath path: a descendant (ok), but not path
		// itself (rel == ".", the ordinary overlap case handled elsewhere).
		if rel, ok := relWithin(path, mnt.MountPath); ok && rel != "." {
			return &mnt
		}
	}
	return nil
}

// ExecPodContainer runs a command in a container in a pod and returns the remotecommand.Executor
func ExecPodContainer(ctx context.Context, restConfig *rest.Config, namespace string, pod string, container string, stdout bool, stderr bool, command ...string) (exec remotecommand.Executor, err error) {
	log := logging.RequireLoggerFromContext(ctx)
	defer func() {
		log.WithFields(logging.Fields{
			"namespace": namespace,
			"pod":       pod,
			"container": container,
			"command":   command,
		}).WithError(err).Debug(ctx, "exec container command")
	}()

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.InternalWrapError(err)
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

	log.Info(ctx, execRequest.URL().String())
	exec, err = remotecommand.NewSPDYExecutor(restConfig, "POST", execRequest.URL())
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return exec, nil
}

// GetExecutorOutput returns the output of an remotecommand.Executor
func GetExecutorOutput(ctx context.Context, exec remotecommand.Executor) (*bytes.Buffer, *bytes.Buffer, error) {
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer
	err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdOut,
		Stderr: &stdErr,
		Tty:    false,
	})
	if err != nil {
		return nil, nil, errors.InternalWrapError(err)
	}
	return &stdOut, &stdErr, nil
}

func overwriteWithDefaultParams(inParam *wfv1.Parameter) {
	if inParam.Value == nil && inParam.Default != nil {
		inParam.Value = inParam.Default
	}
}

func overwriteWithArguments(argParam, inParam *wfv1.Parameter) {
	if argParam != nil {
		if argParam.Value != nil {
			inParam.Value = argParam.Value
			inParam.ValueFrom = nil
		} else {
			inParam.ValueFrom = argParam.ValueFrom
			inParam.Value = nil
		}
	}
}

func substituteAndGetConfigMapValue(ctx context.Context, inParam *wfv1.Parameter, globalParams Parameters, namespace string, configMapStore ConfigMapStore) error {
	log := logging.RequireLoggerFromContext(ctx)
	if inParam.ValueFrom != nil && inParam.ValueFrom.ConfigMapKeyRef != nil {
		if configMapStore != nil {
			replaceMap := make(map[string]any)
			for k, v := range globalParams {
				replaceMap[k] = v
			}

			// SubstituteParams is called only at the end of this method. To support parametrization of the configmap
			// we need to perform a substitution here over the name and the key of the ConfigMapKeyRef.
			cmName, err := substituteConfigMapKeyRefParam(ctx, inParam.ValueFrom.ConfigMapKeyRef.Name, replaceMap)
			if err != nil {
				log.WithError(err).Error(ctx, "unable to substitute name for ConfigMapKeyRef")
				return err
			}
			cmKey, err := substituteConfigMapKeyRefParam(ctx, inParam.ValueFrom.ConfigMapKeyRef.Key, replaceMap)
			if err != nil {
				log.WithError(err).Error(ctx, "unable to substitute key for ConfigMapKeyRef")
				return err
			}

			cmValue, err := GetConfigMapValue(configMapStore, namespace, cmName, cmKey)
			if err != nil {
				if inParam.ValueFrom.Default == nil || !errors.IsCode(errors.CodeNotFound, err) {
					return errors.Errorf(errors.CodeBadRequest, "unable to retrieve inputs.parameters.%s from ConfigMap: %s", inParam.Name, err)
				}
				inParam.Value = inParam.ValueFrom.Default
			} else {
				inParam.Value = wfv1.AnyStringPtr(cmValue)
			}
		}
	} else {
		if inParam.Value == nil {
			return errors.Errorf(errors.CodeBadRequest, "inputs.parameters.%s was not supplied", inParam.Name)
		}
	}
	return nil
}

// ProcessArgs sets in the inputs, the values either passed via arguments, or the hardwired values
// It substitutes:
// * parameters in the template from the arguments
// * global parameters (e.g. {{workflow.parameters.XX}}, {{workflow.name}}, {{workflow.status}})
// * local parameters (e.g. {{pod.name}})
func ProcessArgs(ctx context.Context, tmpl *wfv1.Template, args wfv1.ArgumentsProvider, globalParams, localParams Parameters, validateOnly bool, namespace string, configMapStore ConfigMapStore) (*wfv1.Template, error) {
	// For each input parameter:
	// 1) check if was supplied as argument. if so use the supplied value from arg
	// 2) if not, use default value.
	// 3) if no default value, it is an error
	newTmpl := tmpl.DeepCopy()
	for i, inParam := range newTmpl.Inputs.Parameters {
		// first set to default value
		overwriteWithDefaultParams(&inParam)

		// overwrite value from argument (if supplied)
		argParam := args.GetParameterByName(inParam.Name)
		if argParam != nil && argParam.Value != nil && argParam.Value.String() == AbsentOptionalArgumentValue {
			// The argument was a pure reference to a skipped/omitted node's output with no producer
			// default (see AbsentOptionalArgumentValue): treat it as unsupplied so the input's own
			// default (already applied above) or ValueFrom source takes over. With neither, the
			// absence is unhandled — fail terminally with the real cause; the message must not
			// match template.IsMissingVariableErr, which would requeue forever.
			if inParam.Value == nil && inParam.ValueFrom == nil {
				return nil, errors.Errorf(errors.CodeBadRequest, "inputs.parameters.%s: argument references an absent optional (skipped/omitted node output with no default)", inParam.Name)
			}
		} else {
			overwriteWithArguments(argParam, &inParam)
		}

		// substitute configmap string and get value from store
		err := substituteAndGetConfigMapValue(ctx, &inParam, globalParams, namespace, configMapStore)
		if err != nil {
			return nil, err
		}

		newTmpl.Inputs.Parameters[i] = inParam
	}

	// Performs substitutions of input artifacts
	artifacts := newTmpl.Inputs.Artifacts
	for i, inArt := range artifacts {
		argArt := args.GetArtifactByName(inArt.Name)

		if !inArt.Optional && !inArt.HasLocationOrKey() {
			// artifact must be supplied
			if argArt == nil {
				return nil, errors.Errorf(errors.CodeBadRequest, "inputs.artifacts.%s was not supplied", inArt.Name)
			}
			if (argArt.From == "" || argArt.FromExpression == "") && !argArt.HasLocationOrKey() && !validateOnly {
				return nil, errors.Errorf(errors.CodeBadRequest, "inputs.artifacts.%s missing location information", inArt.Name)
			}
		}
		if argArt != nil {
			artifacts[i] = *argArt
			artifacts[i].Path = inArt.Path
			artifacts[i].Mode = inArt.Mode
			artifacts[i].RecurseMode = inArt.RecurseMode
		}
	}

	return SubstituteParams(ctx, newTmpl, globalParams, localParams)
}

// substituteConfigMapKeyRefParam performs template substitution for ConfigMapKeyRef
func substituteConfigMapKeyRefParam(ctx context.Context, in string, replaceMap map[string]any) (string, error) {
	tmpl, err := template.NewTemplate(in)
	if err != nil {
		return "", err
	}
	replacedString, err := tmpl.Replace(ctx, replaceMap, false)
	if err != nil {
		return "", fmt.Errorf("failed to substitute configMapKeyRef: %w", err)
	}
	return replacedString, nil
}

// SubstituteParams returns a new copy of the template with global, pod, and input parameters substituted
func SubstituteParams(ctx context.Context, tmpl *wfv1.Template, globalParams, localParams Parameters) (*wfv1.Template, error) {
	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	// First replace globals & locals, then replace inputs because globals could be referenced in the inputs
	replaceMap := template.ToAnyMap(globalParams.Merge(localParams))
	globalReplacedTmplStr, err := template.Replace(ctx, string(tmplBytes), replaceMap, true)
	if err != nil {
		return nil, err
	}
	var globalReplacedTmpl wfv1.Template
	err = json.Unmarshal([]byte(globalReplacedTmplStr), &globalReplacedTmpl)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	// Now replace the rest of substitutions (the ones that can be made) in the template
	for _, inParam := range globalReplacedTmpl.Inputs.Parameters {
		if inParam.Value == nil && inParam.ValueFrom == nil {
			return nil, errors.InternalErrorf("inputs.parameters.%s had no value", inParam.Name)
		} else if inParam.Value != nil {
			replaceMap["inputs.parameters."+inParam.Name] = inParam.Value.String()
		}
	}
	// allow {{inputs.parameters}} to fetch the entire input parameters list as JSON
	jsonInputParametersBytes, err := json.Marshal(globalReplacedTmpl.Inputs.Parameters)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	replaceMap["inputs.parameters"] = string(jsonInputParametersBytes)
	for _, inArt := range globalReplacedTmpl.Inputs.Artifacts {
		if inArt.Path != "" {
			replaceMap["inputs.artifacts."+inArt.Name+".path"] = inArt.Path
		}
	}
	for _, outArt := range globalReplacedTmpl.Outputs.Artifacts {
		if outArt.Path != "" {
			replaceMap["outputs.artifacts."+outArt.Name+".path"] = outArt.Path
		}
	}
	for _, param := range globalReplacedTmpl.Outputs.Parameters {
		if param.ValueFrom != nil && param.ValueFrom.Path != "" {
			replaceMap["outputs.parameters."+param.Name+".path"] = param.ValueFrom.Path
		}
	}

	s, err := template.Replace(ctx, globalReplacedTmplStr, replaceMap, true)
	if err != nil {
		return nil, err
	}
	var newTmpl wfv1.Template
	err = json.Unmarshal([]byte(s), &newTmpl)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &newTmpl, nil
}

// GetTemplateGetterString returns string of TemplateHolder.
func GetTemplateGetterString(getter wfv1.TemplateHolder) string {
	return fmt.Sprintf("%T (namespace=%s,name=%s)", getter, getter.GetNamespace(), getter.GetName())
}

// GetTemplateHolderString returns string of TemplateReferenceHolder.
func GetTemplateHolderString(tmplHolder wfv1.TemplateReferenceHolder) string {
	if tmplHolder.GetTemplate() != nil {
		return fmt.Sprintf("%T inlined", tmplHolder)
	} else if x := tmplHolder.GetTemplateName(); x != "" {
		return fmt.Sprintf("%T (%s)", tmplHolder, x)
	} else if x := tmplHolder.GetTemplateRef(); x != nil {
		return fmt.Sprintf("%T (%s/%s#%v)", tmplHolder, x.Name, x.Template, x.ClusterScope)
	}
	return fmt.Sprintf("%T invalid (https://argo-workflows.readthedocs.io/en/latest/templates/)", tmplHolder)
}

func GenerateOnExitNodeName(parentNodeName string) string {
	return fmt.Sprintf("%s.onExit", parentNodeName)
}

func IsDone(un *unstructured.Unstructured) bool {
	return un.GetDeletionTimestamp() == nil &&
		un.GetLabels()[LabelKeyCompleted] == "true" &&
		un.GetLabels()[LabelKeyWorkflowArchivingStatus] != "Pending"
}

// CheckAllHooksFullfilled checks whether child hooked nodes are fulfilled.
func CheckAllHooksFullfilled(node *wfv1.NodeStatus, nodes wfv1.Nodes) bool {
	childs := node.Children
	for _, id := range childs {
		n, ok := nodes[id]
		if !ok {
			continue
		}
		if n.NodeFlag != nil && n.NodeFlag.Hooked && !n.Fulfilled() {
			return false
		}
	}
	return true
}
