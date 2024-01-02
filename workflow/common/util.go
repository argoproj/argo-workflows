package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/template"
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
		normalizedMountPath := strings.TrimRight(mnt.MountPath, "/")
		if path == normalizedMountPath || isSubPath(path, normalizedMountPath) {
			return &mnt
		}
	}
	return nil
}

func isSubPath(path string, normalizedMountPath string) bool {
	return strings.HasPrefix(path, normalizedMountPath+"/")
}

type RoundTripCallback func(conn *websocket.Conn, resp *http.Response, err error) error

type WebsocketRoundTripper struct {
	Dialer *websocket.Dialer
	Do     RoundTripCallback
}

func (d *WebsocketRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	conn, resp, err := d.Dialer.Dial(r.URL.String(), r.Header)
	if err == nil {
		defer util.Close(conn)
	}
	return resp, d.Do(conn, resp, err)
}

// ExecPodContainer runs a command in a container in a pod and returns the remotecommand.Executor
func ExecPodContainer(restConfig *rest.Config, namespace string, pod string, container string, stdout bool, stderr bool, command ...string) (exec remotecommand.Executor, err error) {
	defer func() {
		log.WithField("namespace", namespace).
			WithField("pod", pod).
			WithField("container", container).
			WithField("command", command).
			WithError(err).
			Debug("exec container command")
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

	log.Info(execRequest.URL())
	exec, err = remotecommand.NewSPDYExecutor(restConfig, "POST", execRequest.URL())
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return exec, nil
}

// GetExecutorOutput returns the output of an remotecommand.Executor
func GetExecutorOutput(exec remotecommand.Executor) (*bytes.Buffer, *bytes.Buffer, error) {
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer
	err := exec.Stream(remotecommand.StreamOptions{
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

func substituteAndGetConfigMapValue(inParam *wfv1.Parameter, globalParams Parameters, namespace string, configMapStore ConfigMapStore) error {
	if inParam.ValueFrom != nil && inParam.ValueFrom.ConfigMapKeyRef != nil {
		if configMapStore != nil {
			// SubstituteParams is called only at the end of this method. To support parametrization of the configmap
			// we need to perform a substitution here over the name and the key of the ConfigMapKeyRef.
			cmName, err := substituteConfigMapKeyRefParam(inParam.ValueFrom.ConfigMapKeyRef.Name, globalParams)
			if err != nil {
				log.WithError(err).Error("unable to substitute name for ConfigMapKeyRef")
				return err
			}
			cmKey, err := substituteConfigMapKeyRefParam(inParam.ValueFrom.ConfigMapKeyRef.Key, globalParams)
			if err != nil {
				log.WithError(err).Error("unable to substitute key for ConfigMapKeyRef")
				return err
			}

			cmValue, err := GetConfigMapValue(configMapStore, namespace, cmName, cmKey)
			if err != nil {
				if inParam.ValueFrom.Default != nil && errors.IsCode(errors.CodeNotFound, err) {
					inParam.Value = inParam.ValueFrom.Default
				} else {
					return errors.Errorf(errors.CodeBadRequest, "unable to retrieve inputs.parameters.%s from ConfigMap: %s", inParam.Name, err)
				}
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
func ProcessArgs(tmpl *wfv1.Template, args wfv1.ArgumentsProvider, globalParams, localParams Parameters, validateOnly bool, namespace string, configMapStore ConfigMapStore) (*wfv1.Template, error) {
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
		overwriteWithArguments(argParam, &inParam)

		// substitute configmap string and get value from store
		err := substituteAndGetConfigMapValue(&inParam, globalParams, namespace, configMapStore)
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

	return SubstituteParams(newTmpl, globalParams, localParams)
}

// substituteConfigMapKeyRefParam check if ConfigMapKeyRef's key is a param and perform the substitution.
func substituteConfigMapKeyRefParam(in string, globalParams Parameters) (string, error) {
	if strings.HasPrefix(in, "{{") && strings.HasSuffix(in, "}}") {
		k := strings.TrimSuffix(strings.TrimPrefix(in, "{{"), "}}")
		k = strings.Trim(k, " ")

		v, ok := globalParams[k]
		if !ok {
			err := errors.InternalError(fmt.Sprintf("parameter %s not found", k))
			log.WithError(err).Error()
			return "", err
		}
		return v, nil
	}
	return in, nil
}

// SubstituteParams returns a new copy of the template with global, pod, and input parameters substituted
func SubstituteParams(tmpl *wfv1.Template, globalParams, localParams Parameters) (*wfv1.Template, error) {
	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	// First replace globals & locals, then replace inputs because globals could be referenced in the inputs
	replaceMap := globalParams.Merge(localParams)
	globalReplacedTmplStr, err := template.Replace(string(tmplBytes), replaceMap, true)
	if err != nil {
		return nil, err
	}
	var globalReplacedTmpl wfv1.Template
	err = json.Unmarshal([]byte(globalReplacedTmplStr), &globalReplacedTmpl)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	// Now replace the rest of substitutions (the ones that can be made) in the template
	replaceMap = make(map[string]string)
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

	s, err := template.Replace(globalReplacedTmplStr, replaceMap, true)
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
	} else {
		return fmt.Sprintf("%T invalid (https://argo-workflows.readthedocs.io/en/latest/templates/)", tmplHolder)
	}
}

func GenerateOnExitNodeName(parentNodeName string) string {
	return fmt.Sprintf("%s.onExit", parentNodeName)
}

func IsDone(un *unstructured.Unstructured) bool {
	return un.GetDeletionTimestamp() == nil &&
		un.GetLabels()[LabelKeyCompleted] == "true" &&
		un.GetLabels()[LabelKeyWorkflowArchivingStatus] != "Pending"
}

// Check whether child hooked nodes Fulfilled
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
