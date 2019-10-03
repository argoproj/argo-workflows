package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasttemplate"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util"
)

// FindOverlappingVolume looks an artifact path, checks if it overlaps with any
// user specified volumeMounts in the template, and returns the deepest volumeMount
// (if any). A return value of nil indicates the path is not under any volumeMount.
func FindOverlappingVolume(tmpl *wfv1.Template, path string) *apiv1.VolumeMount {
	var volMounts []apiv1.VolumeMount
	if tmpl.Container != nil {
		volMounts = tmpl.Container.VolumeMounts
	} else if tmpl.Script != nil {
		volMounts = tmpl.Script.VolumeMounts
	} else {
		return nil
	}
	var volMnt *apiv1.VolumeMount
	deepestLen := 0
	for _, mnt := range volMounts {
		if path != mnt.MountPath && !strings.HasPrefix(path, mnt.MountPath+"/") {
			continue
		}
		if len(mnt.MountPath) > deepestLen {
			volMnt = &mnt
			deepestLen = len(mnt.MountPath)
		}
	}
	return volMnt
}

// KillPodContainer is a convenience function to issue a kill signal to a container in a pod
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

// ContainerLogStream returns an io.ReadCloser for a container's log stream using the websocket
// interface. This was implemented in the hopes that we could selectively choose stdout from stderr,
// but due to https://github.com/kubernetes/kubernetes/issues/28167, it is not possible to discern
// stdout from stderr using the K8s API server, so this function is unused, instead preferring the
// pod logs interface from client-go. It's left as a reference for when issue #28167 is eventually
// resolved.
func ContainerLogStream(config *rest.Config, namespace string, pod string, container string) (io.ReadCloser, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	logRequest := clientset.CoreV1().RESTClient().Get().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("log").
		Param("container", container)
	u := logRequest.URL()
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return nil, errors.Errorf("Malformed URL %s", u.String())
	}

	log.Info(u.String())
	wsrc := websocketReadCloser{
		&bytes.Buffer{},
	}

	wrappedRoundTripper, err := roundTripperFromConfig(config, wsrc.WebsocketCallback)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}

	// Send the request and let the callback do its work
	req := &http.Request{
		Method: http.MethodGet,
		URL:    u,
	}
	_, err = wrappedRoundTripper.RoundTrip(req)
	if err != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		return nil, errors.InternalWrapError(err)
	}
	return &wsrc, nil
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

func (w *websocketReadCloser) WebsocketCallback(ws *websocket.Conn, resp *http.Response, err error) error {
	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusOK {
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(resp.Body)
			return errors.InternalErrorf("Can't connect to log endpoint (%d): %s", resp.StatusCode, buf.String())
		}
		return errors.InternalErrorf("Can't connect to log endpoint: %s", err.Error())
	}

	for {
		_, body, err := ws.ReadMessage()
		if len(body) > 0 {
			//log.Debugf("%d: %s", msgType, string(body))
			_, writeErr := w.Write(body)
			if writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			if err == io.EOF {
				log.Infof("websocket closed: %v", err)
				return nil
			}
			log.Warnf("websocket error: %v", err)
			return err
		}
	}
}

func roundTripperFromConfig(config *rest.Config, callback RoundTripCallback) (http.RoundTripper, error) {
	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return nil, err
	}
	// Create a roundtripper which will pass in the final underlying websocket connection to a callback
	wsrt := &WebsocketRoundTripper{
		Do: callback,
		Dialer: &websocket.Dialer{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: tlsConfig,
		},
	}
	// Make sure we inherit all relevant security headers
	return rest.HTTPWrappersForConfig(config, wsrt)
}

type websocketReadCloser struct {
	*bytes.Buffer
}

func (w *websocketReadCloser) Close() error {
	//return w.conn.Close()
	return nil
}

// ExecPodContainer runs a command in a container in a pod and returns the remotecommand.Executor
func ExecPodContainer(restConfig *rest.Config, namespace string, pod string, container string, stdout bool, stderr bool, command ...string) (remotecommand.Executor, error) {
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
	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execRequest.URL())
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

// ProcessArgs sets in the inputs, the values either passed via arguments, or the hardwired values
// It substitutes:
// * parameters in the template from the arguments
// * global parameters (e.g. {{workflow.parameters.XX}}, {{workflow.name}}, {{workflow.status}})
// * local parameters (e.g. {{pod.name}})
func ProcessArgs(tmpl *wfv1.Template, args wfv1.ArgumentsProvider, globalParams, localParams map[string]string, validateOnly bool) (*wfv1.Template, error) {
	// For each input parameter:
	// 1) check if was supplied as argument. if so use the supplied value from arg
	// 2) if not, use default value.
	// 3) if no default value, it is an error
	newTmpl := tmpl.DeepCopy()
	for i, inParam := range newTmpl.Inputs.Parameters {
		if inParam.Default != nil {
			// first set to default value
			inParam.Value = inParam.Default
		}
		// overwrite value from argument (if supplied)
		argParam := args.GetParameterByName(inParam.Name)
		if argParam != nil && argParam.Value != nil {
			newValue := *argParam.Value
			inParam.Value = &newValue
		}
		if inParam.Value == nil {
			return nil, errors.Errorf(errors.CodeBadRequest, "inputs.parameters.%s was not supplied", inParam.Name)
		}
		newTmpl.Inputs.Parameters[i] = inParam
	}

	// Performs substitutions of input artifacts
	newInputArtifacts := make([]wfv1.Artifact, len(newTmpl.Inputs.Artifacts))
	for i, inArt := range newTmpl.Inputs.Artifacts {
		// if artifact has hard-wired location, we prefer that
		if inArt.HasLocation() {
			newInputArtifacts[i] = inArt
			continue
		}
		argArt := args.GetArtifactByName(inArt.Name)
		if !inArt.Optional {
			// artifact must be supplied
			if argArt == nil {
				return nil, errors.Errorf(errors.CodeBadRequest, "inputs.artifacts.%s was not supplied", inArt.Name)
			}
			if !argArt.HasLocation() && !validateOnly {
				return nil, errors.Errorf(errors.CodeBadRequest, "inputs.artifacts.%s missing location information", inArt.Name)
			}
		}
		if argArt != nil {
			argArt.Path = inArt.Path
			argArt.Mode = inArt.Mode
			newInputArtifacts[i] = *argArt
		} else {
			newInputArtifacts[i] = inArt
		}
	}
	newTmpl.Inputs.Artifacts = newInputArtifacts

	return substituteParams(newTmpl, globalParams, localParams)
}

// substituteParams returns a new copy of the template with global, pod, and input parameters substituted
func substituteParams(tmpl *wfv1.Template, globalParams, localParams map[string]string) (*wfv1.Template, error) {
	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	// First replace globals & locals, then replace inputs because globals could be referenced in the inputs
	replaceMap := make(map[string]string)
	for k, v := range globalParams {
		replaceMap[k] = v
	}
	for k, v := range localParams {
		replaceMap[k] = v
	}
	fstTmpl := fasttemplate.New(string(tmplBytes), "{{", "}}")
	globalReplacedTmplStr, err := Replace(fstTmpl, replaceMap, true)
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
		if inParam.Value == nil {
			return nil, errors.InternalErrorf("inputs.parameters.%s had no value", inParam.Name)
		}
		replaceMap["inputs.parameters."+inParam.Name] = *inParam.Value
	}
	//allow {{inputs.parameters}} to fetch the entire input parameters list as JSON
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

	fstTmpl = fasttemplate.New(globalReplacedTmplStr, "{{", "}}")
	s, err := Replace(fstTmpl, replaceMap, true)
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

// Replace executes basic string substitution of a template with replacement values.
// allowUnresolved indicates whether or not it is acceptable to have unresolved variables
// remaining in the substituted template.
func Replace(fstTmpl *fasttemplate.Template, replaceMap map[string]string, allowUnresolved bool) (string, error) {
	var unresolvedErr error
	replacedTmpl := fstTmpl.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		replacement, ok := replaceMap[tag]
		if !ok {
			if allowUnresolved {
				// just write the same string back
				return w.Write([]byte(fmt.Sprintf("{{%s}}", tag)))
			}
			unresolvedErr = errors.Errorf(errors.CodeBadRequest, "failed to resolve {{%s}}", tag)
			return 0, nil
		}
		// The following escapes any special characters (e.g. newlines, tabs, etc...)
		// in preparation for substitution
		replacement = strconv.Quote(replacement)
		replacement = replacement[1 : len(replacement)-1]
		return w.Write([]byte(replacement))
	})
	if unresolvedErr != nil {
		return "", unresolvedErr
	}
	return replacedTmpl, nil
}

// RunCommand is a convenience function to run/log a command and log the stderr upon failure
func RunCommand(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmdStr := strings.Join(cmd.Args, " ")
	log.Info(cmdStr)
	_, err := cmd.Output()
	if err != nil {
		if exErr, ok := err.(*exec.ExitError); ok {
			errOutput := string(exErr.Stderr)
			log.Errorf("`%s` failed: %s", cmdStr, errOutput)
			return errors.InternalError(strings.TrimSpace(errOutput))
		}
		return errors.InternalWrapError(err)
	}
	return nil
}

const patchRetries = 5

// AddPodAnnotation adds an annotation to pod
func AddPodAnnotation(c kubernetes.Interface, podName, namespace, key, value string) error {
	return addPodMetadata(c, "annotations", podName, namespace, key, value)
}

// AddPodLabel adds an label to pod
func AddPodLabel(c kubernetes.Interface, podName, namespace, key, value string) error {
	return addPodMetadata(c, "labels", podName, namespace, key, value)
}

// addPodMetadata is helper to either add a pod label or annotation to the pod
func addPodMetadata(c kubernetes.Interface, field, podName, namespace, key, value string) error {
	metadata := map[string]interface{}{
		"metadata": map[string]interface{}{
			field: map[string]string{
				key: value,
			},
		},
	}
	var err error
	patch, err := json.Marshal(metadata)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	for attempt := 0; attempt < patchRetries; attempt++ {
		_, err = c.CoreV1().Pods(namespace).Patch(podName, types.MergePatchType, patch)
		if err != nil {
			if !apierr.IsConflict(err) {
				return err
			}
		} else {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return err
}

const deleteRetries = 3

// DeletePod deletes a pod. Ignores NotFound error
func DeletePod(c kubernetes.Interface, podName, namespace string) error {
	var err error
	for attempt := 0; attempt < deleteRetries; attempt++ {
		err = c.CoreV1().Pods(namespace).Delete(podName, &metav1.DeleteOptions{})
		if err == nil || apierr.IsNotFound(err) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return err
}

// IsPodTemplate returns whether the template corresponds to a pod
func IsPodTemplate(tmpl *wfv1.Template) bool {
	if tmpl.Container != nil || tmpl.Script != nil || tmpl.Resource != nil {
		return true
	}
	return false
}

var yamlSeparator = regexp.MustCompile(`\n---`)

// SplitWorkflowYAMLFile is a helper to split a body into multiple workflow objects
func SplitWorkflowYAMLFile(body []byte, strict bool) ([]wfv1.Workflow, error) {
	manifestsStrings := yamlSeparator.Split(string(body), -1)
	manifests := make([]wfv1.Workflow, 0)
	for _, manifestStr := range manifestsStrings {
		if strings.TrimSpace(manifestStr) == "" {
			continue
		}
		var wf wfv1.Workflow
		var opts []yaml.JSONOpt
		if strict {
			opts = append(opts, yaml.DisallowUnknownFields) // nolint
		}
		err := yaml.Unmarshal([]byte(manifestStr), &wf, opts...)
		if wf.Kind != "" && wf.Kind != workflow.WorkflowKind {
			log.Warnf("%s is not a workflow", wf.Kind)
			// If we get here, it was a k8s manifest which was not of type 'Workflow'
			// We ignore these since we only care about Workflow manifests.
			continue
		}
		if err != nil {
			return nil, errors.New(errors.CodeBadRequest, err.Error())
		}
		manifests = append(manifests, wf)
	}
	return manifests, nil
}

// SplitWorkflowTemplateYAMLFile is a helper to split a body into multiple workflow template objects
func SplitWorkflowTemplateYAMLFile(body []byte, strict bool) ([]wfv1.WorkflowTemplate, error) {
	manifestsStrings := yamlSeparator.Split(string(body), -1)
	manifests := make([]wfv1.WorkflowTemplate, 0)
	for _, manifestStr := range manifestsStrings {
		if strings.TrimSpace(manifestStr) == "" {
			continue
		}
		var wftmpl wfv1.WorkflowTemplate
		var opts []yaml.JSONOpt
		if strict {
			opts = append(opts, yaml.DisallowUnknownFields) // nolint
		}
		err := yaml.Unmarshal([]byte(manifestStr), &wftmpl, opts...)
		if wftmpl.Kind != "" && wftmpl.Kind != workflow.WorkflowTemplateKind {
			log.Warnf("%s is not a workflow template", wftmpl.Kind)
			// If we get here, it was a k8s manifest which was not of type 'WorkflowTemplate'
			// We ignore these since we only care about WorkflowTemplate manifests.
			continue
		}
		if err != nil {
			return nil, errors.New(errors.CodeBadRequest, err.Error())
		}
		manifests = append(manifests, wftmpl)
	}
	return manifests, nil
}

// MergeReferredTemplate merges a referred template to the receiver template.
func MergeReferredTemplate(tmpl *wfv1.Template, referred *wfv1.Template) (*wfv1.Template, error) {
	// Copy the referred template to deep copy template types.
	newTmpl := referred.DeepCopy()

	newTmpl.Name = tmpl.Name
	newTmpl.Outputs = *tmpl.Outputs.DeepCopy()

	if len(tmpl.NodeSelector) > 0 {
		m := make(map[string]string, len(tmpl.NodeSelector))
		for k, v := range tmpl.NodeSelector {
			m[k] = v
		}
		newTmpl.NodeSelector = m
	}
	if tmpl.Affinity != nil {
		newTmpl.Affinity = tmpl.Affinity.DeepCopy()
	}
	if len(newTmpl.Metadata.Annotations) > 0 || len(tmpl.Metadata.Labels) > 0 {
		newTmpl.Metadata = *tmpl.Metadata.DeepCopy()
	}
	if tmpl.Daemon != nil {
		v := *tmpl.Daemon
		newTmpl.Daemon = &v
	}
	if len(tmpl.Volumes) > 0 {
		volumes := make([]apiv1.Volume, len(tmpl.Volumes))
		copy(volumes, tmpl.Volumes)
		newTmpl.Volumes = volumes
	}
	if len(tmpl.InitContainers) > 0 {
		containers := make([]wfv1.UserContainer, len(tmpl.InitContainers))
		copy(containers, tmpl.InitContainers)
		newTmpl.InitContainers = containers
	}
	if len(tmpl.Sidecars) > 0 {
		containers := make([]wfv1.UserContainer, len(tmpl.Sidecars))
		copy(containers, tmpl.Sidecars)
		newTmpl.Sidecars = containers
	}
	if tmpl.ArchiveLocation != nil {
		newTmpl.ArchiveLocation = tmpl.ArchiveLocation.DeepCopy()
	}
	if tmpl.ActiveDeadlineSeconds != nil {
		v := *tmpl.ActiveDeadlineSeconds
		newTmpl.ActiveDeadlineSeconds = &v
	}
	if tmpl.RetryStrategy != nil {
		newTmpl.RetryStrategy = tmpl.RetryStrategy.DeepCopy()
	}
	if tmpl.Parallelism != nil {
		v := *tmpl.Parallelism
		newTmpl.Parallelism = &v
	}
	if len(tmpl.Tolerations) != 0 {
		tolerations := make([]apiv1.Toleration, len(tmpl.Tolerations))
		copy(tolerations, tmpl.Tolerations)
		newTmpl.Tolerations = tolerations
	}
	if tmpl.SchedulerName != "" {
		newTmpl.SchedulerName = tmpl.SchedulerName
	}
	if tmpl.PriorityClassName != "" {
		newTmpl.PriorityClassName = tmpl.PriorityClassName
	}
	if tmpl.Priority != nil {
		v := *tmpl.Priority
		newTmpl.Priority = &v
	}

	return newTmpl, nil
}

// GetTemplateGetterString returns string of TemplateGetter.
func GetTemplateGetterString(getter wfv1.TemplateGetter) string {
	return fmt.Sprintf("%T (namespace=%s,name=%s)", getter, getter.GetNamespace(), getter.GetName())
}

// GetTemplateHolderString returns string of TemplateHolder.
func GetTemplateHolderString(tmplHolder wfv1.TemplateHolder) string {
	tmplName := tmplHolder.GetTemplateName()
	tmplRef := tmplHolder.GetTemplateRef()
	if tmplRef != nil {
		return fmt.Sprintf("%T (%s/%s)", tmplHolder, tmplRef.Name, tmplRef.Template)
	} else {
		return fmt.Sprintf("%T (%s)", tmplHolder, tmplName)
	}
}
