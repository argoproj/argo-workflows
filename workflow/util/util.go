package util

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	nruntime "runtime"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo-workflows/v3/util/cmd"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/retry"
	unstructutil "github.com/argoproj/argo-workflows/v3/util/unstructured"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v3/workflow/packer"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
)

// NewWorkflowInformer returns the workflow informer used by the controller. This is actually
// a custom built UnstructuredInformer which is in actuality returning unstructured.Unstructured
// objects. We no longer return WorkflowInformer due to:
// https://github.com/kubernetes/kubernetes/issues/57705
// https://github.com/argoproj/argo-workflows/issues/632
func NewWorkflowInformer(dclient dynamic.Interface, ns string, resyncPeriod time.Duration, tweakListOptions internalinterfaces.TweakListOptionsFunc, indexers cache.Indexers) cache.SharedIndexInformer {
	resource := schema.GroupVersionResource{
		Group:    workflow.Group,
		Version:  "v1alpha1",
		Resource: workflow.WorkflowPlural,
	}
	informer := unstructutil.NewFilteredUnstructuredInformer(
		resource,
		dclient,
		ns,
		resyncPeriod,
		indexers,
		tweakListOptions,
	)
	return informer
}

// InstanceIDRequirement returns the label requirement to filter against a controller instance (or not)
func InstanceIDRequirement(instanceID string) labels.Requirement {
	var instanceIDReq *labels.Requirement
	var err error
	if instanceID != "" {
		instanceIDReq, err = labels.NewRequirement(common.LabelKeyControllerInstanceID, selection.Equals, []string{instanceID})
	} else {
		instanceIDReq, err = labels.NewRequirement(common.LabelKeyControllerInstanceID, selection.DoesNotExist, nil)
	}
	if err != nil {
		panic(err)
	}
	return *instanceIDReq
}

// WorkflowLister implements the List() method of v1alpha.WorkflowLister interface but does so using
// an Unstructured informer and converting objects to workflows. Ignores objects that failed to convert.
type WorkflowLister interface {
	List() ([]*wfv1.Workflow, error)
}

type workflowLister struct {
	informer cache.SharedIndexInformer
}

func (l *workflowLister) List() ([]*wfv1.Workflow, error) {
	workflows := make([]*wfv1.Workflow, 0)
	for _, m := range l.informer.GetStore().List() {
		wf, err := FromUnstructured(m.(*unstructured.Unstructured))
		if err != nil {
			log.Warnf("Failed to unmarshal workflow %v object: %v", m, err)
			continue
		}
		workflows = append(workflows, wf)
	}
	return workflows, nil
}

// NewWorkflowLister returns a new workflow lister
func NewWorkflowLister(informer cache.SharedIndexInformer) WorkflowLister {
	return &workflowLister{
		informer: informer,
	}
}

// FromUnstructured converts an unstructured object to a workflow.
// This function performs a lot of allocations and con resulting in a lot of memory
// being used. Users should avoid invoking this function if the data they need is
// available from `unstructured.Unstructured`. especially if they're looping.
// Available values include: `GetLabels()`, `GetName()`, `GetNamespace()` etc.
// Single values can be accessed using `unstructured.Nested*`, e.g.
// `unstructured.NestedString(un.Object, "spec", "phase")`.
func FromUnstructured(un *unstructured.Unstructured) (*wfv1.Workflow, error) {
	var wf wfv1.Workflow
	err := FromUnstructuredObj(un, &wf)
	return &wf, err
}

func FromUnstructuredObj(un *unstructured.Unstructured, v interface{}) error {
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, v)
	if err != nil {
		if err.Error() == "cannot convert int64 to v1alpha1.AnyString" {
			data, err := json.Marshal(un)
			if err != nil {
				return err
			}
			return json.Unmarshal(data, v)
		}
		return err
	}
	return nil
}

// ToUnstructured converts an workflow to an Unstructured object
func ToUnstructured(wf *wfv1.Workflow) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(wf)
	if err != nil {
		return nil, err
	}
	un := &unstructured.Unstructured{Object: obj}
	// we need to add these values so that the `EventRecorder` does not error
	un.SetKind(workflow.WorkflowKind)
	un.SetAPIVersion(workflow.APIVersion)
	return un, nil
}

// IsWorkflowCompleted returns whether or not a workflow is considered completed
func IsWorkflowCompleted(wf *wfv1.Workflow) bool {
	if wf.ObjectMeta.Labels != nil {
		return wf.ObjectMeta.Labels[common.LabelKeyCompleted] == "true"
	}
	return false
}

// SubmitWorkflow validates and submits a single workflow and overrides some of the fields of the workflow
func SubmitWorkflow(ctx context.Context, wfIf v1alpha1.WorkflowInterface, wfClientset wfclientset.Interface, namespace string, wf *wfv1.Workflow, opts *wfv1.SubmitOpts) (*wfv1.Workflow, error) {
	err := ApplySubmitOpts(wf, opts)
	if err != nil {
		return nil, err
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates())

	_, err = validate.ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, validate.ValidateOpts{Submit: true})
	if err != nil {
		return nil, err
	}
	if opts.DryRun {
		return wf, nil
	} else if opts.ServerDryRun {
		wf, err := CreateServerDryRun(ctx, wf, wfClientset)
		if err != nil {
			return nil, err
		}
		return wf, err
	} else {
		return wfIf.Create(ctx, wf, metav1.CreateOptions{})
	}
}

// CreateServerDryRun fills the workflow struct with the server's representation without creating it and returns an error, if there is any
func CreateServerDryRun(ctx context.Context, wf *wfv1.Workflow, wfClientset wfclientset.Interface) (*wfv1.Workflow, error) {
	// Keep the workflow metadata because it will be overwritten by the Post request
	workflowTypeMeta := wf.TypeMeta
	err := wfClientset.ArgoprojV1alpha1().RESTClient().Post().
		Namespace(wf.Namespace).
		Resource("workflows").
		Body(wf).
		Param("dryRun", "All").
		Do(ctx).
		Into(wf)
	wf.TypeMeta = workflowTypeMeta
	return wf, err
}

func PopulateSubmitOpts(command *cobra.Command, submitOpts *wfv1.SubmitOpts, parameterFile *string, includeDryRun bool) {
	command.Flags().StringVar(&submitOpts.Name, "name", "", "override metadata.name")
	command.Flags().StringVar(&submitOpts.GenerateName, "generate-name", "", "override metadata.generateName")
	command.Flags().StringVar(&submitOpts.Entrypoint, "entrypoint", "", "override entrypoint")
	command.Flags().StringArrayVarP(&submitOpts.Parameters, "parameter", "p", []string{}, "pass an input parameter")
	command.Flags().StringVar(&submitOpts.ServiceAccount, "serviceaccount", "", "run all pods in the workflow using specified serviceaccount")
	command.Flags().StringVarP(parameterFile, "parameter-file", "f", "", "pass a file containing all input parameters")
	command.Flags().StringVarP(&submitOpts.Labels, "labels", "l", "", "Comma separated labels to apply to the workflow. Will override previous values.")

	if includeDryRun {
		command.Flags().BoolVar(&submitOpts.DryRun, "dry-run", false, "modify the workflow on the client-side without creating it")
		command.Flags().BoolVar(&submitOpts.ServerDryRun, "server-dry-run", false, "send request to server with dry-run flag which will modify the workflow without creating it")
	}
}

// Apply the Submit options into workflow object
func ApplySubmitOpts(wf *wfv1.Workflow, opts *wfv1.SubmitOpts) error {
	if wf == nil {
		return fmt.Errorf("workflow cannot be nil")
	}
	if opts == nil {
		opts = &wfv1.SubmitOpts{}
	}
	if opts.Entrypoint != "" {
		wf.Spec.Entrypoint = opts.Entrypoint
	}
	if opts.ServiceAccount != "" {
		wf.Spec.ServiceAccountName = opts.ServiceAccount
	}
	if opts.PodPriorityClassName != "" {
		wf.Spec.PodPriorityClassName = opts.PodPriorityClassName
	}

	if opts.Priority != nil {
		wf.Spec.Priority = opts.Priority
	}

	wfLabels := wf.GetLabels()
	if wfLabels == nil {
		wfLabels = make(map[string]string)
	}
	if opts.Labels != "" {
		passedLabels, err := cmdutil.ParseLabels(opts.Labels)
		if err != nil {
			return fmt.Errorf("expected labels of the form: NAME1=VALUE2,NAME2=VALUE2. Received: %s: %w", opts.Labels, err)
		}
		for k, v := range passedLabels {
			wfLabels[k] = v
		}
	}
	wf.SetLabels(wfLabels)
	wfAnnotations := wf.GetAnnotations()
	if wfAnnotations == nil {
		wfAnnotations = make(map[string]string)
	}
	if opts.Annotations != "" {
		fmt.Println(opts.Annotations)
		passedAnnotations, err := cmdutil.ParseLabels(opts.Annotations)
		if err != nil {
			return fmt.Errorf("expected Annotations of the form: NAME1=VALUE2,NAME2=VALUE2. Received: %s: %w", opts.Labels, err)
		}
		for k, v := range passedAnnotations {
			wfAnnotations[k] = v
		}
	}
	wf.SetAnnotations(wfAnnotations)
	if len(opts.Parameters) > 0 {
		newParams := make([]wfv1.Parameter, 0)
		passedParams := make(map[string]bool)
		for _, paramStr := range opts.Parameters {
			parts := strings.SplitN(paramStr, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("expected parameter of the form: NAME=VALUE. Received: %s", paramStr)
			}
			param := wfv1.Parameter{Name: parts[0], Value: wfv1.AnyStringPtr(parts[1])}
			newParams = append(newParams, param)
			passedParams[param.Name] = true
		}
		for _, param := range wf.Spec.Arguments.Parameters {
			if _, ok := passedParams[param.Name]; ok {
				// this parameter was overridden via command line
				continue
			}
			newParams = append(newParams, param)
		}
		wf.Spec.Arguments.Parameters = newParams
	}
	if opts.GenerateName != "" {
		wf.ObjectMeta.GenerateName = opts.GenerateName
	}
	if opts.Name != "" {
		wf.ObjectMeta.Name = opts.Name
	}
	if opts.OwnerReference != nil {
		wf.SetOwnerReferences(append(wf.GetOwnerReferences(), *opts.OwnerReference))
	}
	return nil
}

func ReadParametersFile(file string, opts *wfv1.SubmitOpts) error {
	var body []byte
	var err error
	if cmdutil.IsURL(file) {
		body, err = ReadFromUrl(file)
		if err != nil {
			return err
		}
	} else {
		body, err = ioutil.ReadFile(file)
		if err != nil {
			return err
		}
	}

	yamlParams := map[string]json.RawMessage{}
	err = yaml.Unmarshal(body, &yamlParams)
	if err != nil {
		return err
	}

	for k, v := range yamlParams {
		// We get quoted strings from the yaml file.
		value, err := strconv.Unquote(string(v))
		if err != nil {
			// the string is already clean.
			value = string(v)
		}
		opts.Parameters = append(opts.Parameters, fmt.Sprintf("%s=%s", k, value))
	}
	return nil
}

// SuspendWorkflow suspends a workflow by setting spec.suspend to true. Retries conflict errors
func SuspendWorkflow(ctx context.Context, wfIf v1alpha1.WorkflowInterface, workflowName string) error {
	err := waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
		wf, err := wfIf.Get(ctx, workflowName, metav1.GetOptions{})
		if err != nil {
			return !errorsutil.IsTransientErr(err), err
		}
		if IsWorkflowCompleted(wf) {
			return false, errSuspendedCompletedWorkflow
		}
		if wf.Spec.Suspend == nil || !*wf.Spec.Suspend {
			wf.Spec.Suspend = pointer.BoolPtr(true)
			_, err := wfIf.Update(ctx, wf, metav1.UpdateOptions{})
			if apierr.IsConflict(err) {
				return false, nil
			}
			return !errorsutil.IsTransientErr(err), err
		}
		return true, nil
	})
	return err
}

// ResumeWorkflow resumes a workflow by setting spec.suspend to nil and any suspended nodes to Successful.
// Retries conflict errors
func ResumeWorkflow(ctx context.Context, wfIf v1alpha1.WorkflowInterface, hydrator hydrator.Interface, workflowName string, nodeFieldSelector string) error {
	if len(nodeFieldSelector) > 0 {
		return updateSuspendedNode(ctx, wfIf, hydrator, workflowName, nodeFieldSelector, SetOperationValues{Phase: wfv1.NodeSucceeded})
	} else {
		err := waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
			wf, err := wfIf.Get(ctx, workflowName, metav1.GetOptions{})
			if err != nil {
				return !errorsutil.IsTransientErr(err), err
			}

			err = hydrator.Hydrate(wf)
			if err != nil {
				return true, err
			}

			workflowUpdated := false
			if wf.Spec.Suspend != nil && *wf.Spec.Suspend {
				wf.Spec.Suspend = nil
				workflowUpdated = true
			}

			// To resume a workflow with a suspended node we simply mark the node as Successful
			for nodeID, node := range wf.Status.Nodes {
				if node.IsActiveSuspendNode() {
					if node.Outputs != nil {
						for i, param := range node.Outputs.Parameters {
							if param.ValueFrom != nil && param.ValueFrom.Supplied != nil {
								if param.ValueFrom.Default != nil {
									node.Outputs.Parameters[i].Value = param.ValueFrom.Default
									node.Outputs.Parameters[i].ValueFrom = nil
								} else {
									return false, fmt.Errorf("raw output parameter '%s' has not been set and does not have a default value", param.Name)
								}
							}
						}
					}
					node.Phase = wfv1.NodeSucceeded
					node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
					wf.Status.Nodes[nodeID] = node
					workflowUpdated = true
				}
			}

			if workflowUpdated {
				err := hydrator.Dehydrate(wf)
				if err != nil {
					return false, fmt.Errorf("unable to compress or offload workflow nodes: %s", err)
				}

				_, err = wfIf.Update(ctx, wf, metav1.UpdateOptions{})
				if err != nil {
					if apierr.IsConflict(err) {
						return false, nil
					}
					return false, err
				}
			}
			return true, nil
		})
		return err
	}
}

func SelectorMatchesNode(selector fields.Selector, node wfv1.NodeStatus) bool {
	nodeFields := fields.Set{
		"displayName":  node.DisplayName,
		"templateName": node.TemplateName,
		"phase":        string(node.Phase),
		"name":         node.Name,
		"id":           node.ID,
	}
	if node.TemplateRef != nil {
		nodeFields["templateRef.name"] = node.TemplateRef.Name
		nodeFields["templateRef.template"] = node.TemplateRef.Template
	}
	if node.Inputs != nil {
		for _, inParam := range node.Inputs.Parameters {
			nodeFields[fmt.Sprintf("inputs.parameters.%s.value", inParam.Name)] = inParam.Value.String()
		}
	}

	return selector.Matches(nodeFields)
}

type SetOperationValues struct {
	Phase            wfv1.NodePhase
	Message          string
	OutputParameters map[string]string
}

func AddParamToGlobalScope(wf *wfv1.Workflow, log *log.Entry, param wfv1.Parameter) bool {
	wfUpdated := false
	if param.GlobalName == "" {
		return wfUpdated
	}
	index := -1
	if wf.Status.Outputs != nil {
		for i, gParam := range wf.Status.Outputs.Parameters {
			if gParam.Name == param.GlobalName {
				index = i
				break
			}
		}
	} else {
		wf.Status.Outputs = &wfv1.Outputs{}
	}
	paramName := fmt.Sprintf("workflow.outputs.parameters.%s", param.GlobalName)
	if index == -1 {
		log.Infof("setting %s: '%s'", paramName, param.Value)
		gParam := wfv1.Parameter{Name: param.GlobalName, Value: param.Value}
		wf.Status.Outputs.Parameters = append(wf.Status.Outputs.Parameters, gParam)
		wfUpdated = true
	} else {
		prevVal := wf.Status.Outputs.Parameters[index].Value
		if prevVal == nil || (param.Value != nil && *prevVal != *param.Value) {
			log.Infof("overwriting %s: '%s' -> '%s'", paramName, wf.Status.Outputs.Parameters[index].Value, param.Value)
			wf.Status.Outputs.Parameters[index].Value = param.Value
			wfUpdated = true
		}
	}
	return wfUpdated
}

func updateSuspendedNode(ctx context.Context, wfIf v1alpha1.WorkflowInterface, hydrator hydrator.Interface, workflowName string, nodeFieldSelector string, values SetOperationValues) error {
	selector, err := fields.ParseSelector(nodeFieldSelector)
	if err != nil {
		return err
	}
	err = waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
		wf, err := wfIf.Get(ctx, workflowName, metav1.GetOptions{})
		if err != nil {
			return !errorsutil.IsTransientErr(err), err
		}

		err = hydrator.Hydrate(wf)
		if err != nil {
			return false, err
		}

		nodeUpdated := false
		for nodeID, node := range wf.Status.Nodes {
			if node.IsActiveSuspendNode() {
				if SelectorMatchesNode(selector, node) {

					// Update phase
					if values.Phase != "" {
						node.Phase = values.Phase
						if values.Phase.Fulfilled() {
							node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
						}
						nodeUpdated = true
					}

					// Update message
					if values.Message != "" {
						node.Message = values.Message
						nodeUpdated = true
					}

					// Update output parameters
					if len(values.OutputParameters) > 0 {
						if node.Outputs == nil {
							return true, fmt.Errorf("cannot set output parameters because node is not expecting any raw parameters")
						}
						for name, val := range values.OutputParameters {
							hit := false
							for i, param := range node.Outputs.Parameters {
								if param.Name == name {
									if param.ValueFrom == nil || param.ValueFrom.Supplied == nil {
										return true, fmt.Errorf("cannot set output parameter '%s' because it does not use valueFrom.raw or it was already set", param.Name)
									}
									node.Outputs.Parameters[i].Value = wfv1.AnyStringPtr(val)
									node.Outputs.Parameters[i].ValueFrom = nil
									nodeUpdated = true
									hit = true
									AddParamToGlobalScope(wf, log.NewEntry(log.StandardLogger()), node.Outputs.Parameters[i])
									break
								}
							}
							if !hit {
								return true, fmt.Errorf("node is not expecting output parameter '%s'", name)
							}
						}
					}

					wf.Status.Nodes[nodeID] = node
				}
			}
		}

		if !nodeUpdated {
			return true, fmt.Errorf("currently, set only targets suspend nodes: no suspend nodes matching nodeFieldSelector: %s", nodeFieldSelector)
		}

		err = hydrator.Dehydrate(wf)
		if err != nil {
			return true, fmt.Errorf("unable to compress or offload workflow nodes: %s", err)
		}

		_, err = wfIf.Update(ctx, wf, metav1.UpdateOptions{})
		if err != nil {
			if apierr.IsConflict(err) {
				// Try again if we have a conflict
				return false, nil
			}
			return true, err
		}

		return true, nil
	})
	return err
}

const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

func init() {
	rand.Seed(time.Now().UnixNano())
}

// generates an insecure random string
func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))] //nolint:gosec
	}
	return string(b)
}

// RandSuffix generates a random suffix suitable for suffixing resource name.
func RandSuffix() string {
	return randString(5)
}

// FormulateResubmitWorkflow formulate a new workflow from a previous workflow, optionally re-using successful nodes
func FormulateResubmitWorkflow(wf *wfv1.Workflow, memoized bool) (*wfv1.Workflow, error) {
	newWF := wfv1.Workflow{}
	newWF.TypeMeta = wf.TypeMeta

	// Resubmitted workflow will use generated names
	if wf.ObjectMeta.GenerateName != "" {
		newWF.ObjectMeta.GenerateName = wf.ObjectMeta.GenerateName
	} else {
		newWF.ObjectMeta.GenerateName = wf.ObjectMeta.Name + "-"
	}
	// When resubmitting workflow with memoized nodes, we need to use a predetermined workflow name
	// in order to formulate the node statuses. Which means we cannot reuse metadata.generateName
	// The following simulates the behavior of generateName
	if memoized {
		switch wf.Status.Phase {
		case wfv1.WorkflowFailed, wfv1.WorkflowError:
		default:
			return nil, errors.Errorf(errors.CodeBadRequest, "workflow must be Failed/Error to resubmit in memoized mode")
		}
		newWF.ObjectMeta.Name = newWF.ObjectMeta.GenerateName + RandSuffix()
	}

	// carry over the unmodified spec
	newWF.Spec = wf.Spec

	if newWF.Spec.ActiveDeadlineSeconds != nil && *newWF.Spec.ActiveDeadlineSeconds == 0 {
		// if it was terminated, unset the deadline
		newWF.Spec.ActiveDeadlineSeconds = nil
	}

	newWF.Spec.Shutdown = ""

	// carry over user labels and annotations from previous workflow.
	if newWF.ObjectMeta.Labels == nil {
		newWF.ObjectMeta.Labels = make(map[string]string)
	}
	for key, val := range wf.ObjectMeta.Labels {
		switch key {
		case common.LabelKeyCreator, common.LabelKeyPhase, common.LabelKeyCompleted, common.LabelKeyWorkflowArchivingStatus:
			// ignore
		default:
			newWF.ObjectMeta.Labels[key] = val
		}
	}
	// Append an additional label so it's easy for user to see the
	// name of the original workflow that has been resubmitted.
	newWF.ObjectMeta.Labels[common.LabelKeyPreviousWorkflowName] = wf.ObjectMeta.Name
	if newWF.ObjectMeta.Annotations == nil {
		newWF.ObjectMeta.Annotations = make(map[string]string)
	}
	for key, val := range wf.ObjectMeta.Annotations {
		newWF.ObjectMeta.Annotations[key] = val
	}

	// Setting OwnerReference from original Workflow
	newWF.OwnerReferences = append(newWF.OwnerReferences, wf.OwnerReferences...)

	if !memoized {
		return &newWF, nil
	}

	// Iterate the previous nodes.
	replaceRegexp := regexp.MustCompile("^" + wf.ObjectMeta.Name)
	newWF.Status.Nodes = make(map[string]wfv1.NodeStatus)
	onExitNodeName := wf.ObjectMeta.Name + ".onExit"
	err := packer.DecompressWorkflow(wf)
	if err != nil {
		log.Fatal(err)
	}
	for _, node := range wf.Status.Nodes {
		newNode := node.DeepCopy()
		if strings.HasPrefix(node.Name, onExitNodeName) {
			continue
		}
		originalID := node.ID
		newNode.Name = replaceRegexp.ReplaceAllString(node.Name, newWF.ObjectMeta.Name)
		newNode.ID = newWF.NodeID(newNode.Name)
		if node.BoundaryID != "" {
			newNode.BoundaryID = convertNodeID(&newWF, replaceRegexp, node.BoundaryID, wf.Status.Nodes)
		}
		if newNode.FailedOrError() && newNode.Type == wfv1.NodeTypePod {
			newNode.StartedAt = metav1.Time{}
			newNode.FinishedAt = metav1.Time{}
		} else {
			newNode.StartedAt = metav1.Time{Time: time.Now().UTC()}
			newNode.FinishedAt = newNode.StartedAt
		}
		newChildren := make([]string, len(node.Children))
		for i, childID := range node.Children {
			newChildren[i] = convertNodeID(&newWF, replaceRegexp, childID, wf.Status.Nodes)
		}
		newNode.Children = newChildren
		newOutboundNodes := make([]string, len(node.OutboundNodes))
		for i, outboundID := range node.OutboundNodes {
			newOutboundNodes[i] = convertNodeID(&newWF, replaceRegexp, outboundID, wf.Status.Nodes)
		}
		newNode.OutboundNodes = newOutboundNodes
		if !newNode.FailedOrError() && newNode.Type == wfv1.NodeTypePod {
			newNode.Phase = wfv1.NodeSkipped
			newNode.Type = wfv1.NodeTypeSkipped
			newNode.Message = fmt.Sprintf("original pod: %s", originalID)
		} else {
			newNode.Phase = wfv1.NodePending
			newNode.Message = ""
		}
		newWF.Status.Nodes[newNode.ID] = *newNode
	}

	newWF.Status.StoredTemplates = make(map[string]wfv1.Template)
	for id, tmpl := range wf.Status.StoredTemplates {
		newWF.Status.StoredTemplates[id] = tmpl
	}

	newWF.Status.Conditions = wfv1.Conditions{{Status: metav1.ConditionFalse, Type: wfv1.ConditionTypeCompleted}}
	newWF.Status.Phase = wfv1.WorkflowUnknown

	return &newWF, nil
}

// convertNodeID converts an old nodeID to a new nodeID
func convertNodeID(newWf *wfv1.Workflow, regex *regexp.Regexp, oldNodeID string, oldNodes map[string]wfv1.NodeStatus) string {
	node := oldNodes[oldNodeID]
	newNodeName := regex.ReplaceAllString(node.Name, newWf.ObjectMeta.Name)
	return newWf.NodeID(newNodeName)
}

// RetryWorkflow updates a workflow, deleting all failed steps as well as the onExit node (and children)
func RetryWorkflow(ctx context.Context, kubeClient kubernetes.Interface, hydrator hydrator.Interface, wfClient v1alpha1.WorkflowInterface, name string, restartSuccessful bool, nodeFieldSelector string) (*wfv1.Workflow, error) {
	var updated *wfv1.Workflow
	err := waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
		var err error
		updated, err = retryWorkflow(ctx, kubeClient, hydrator, wfClient, name, restartSuccessful, nodeFieldSelector)
		return !errorsutil.IsTransientErr(err), err
	})
	if err != nil {
		return nil, err
	}
	return updated, err
}

func retryWorkflow(ctx context.Context, kubeClient kubernetes.Interface, hydrator hydrator.Interface, wfClient v1alpha1.WorkflowInterface, name string, restartSuccessful bool, nodeFieldSelector string) (*wfv1.Workflow, error) {
	wf, err := wfClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	switch wf.Status.Phase {
	case wfv1.WorkflowFailed, wfv1.WorkflowError:
	default:
		return nil, errors.Errorf(errors.CodeBadRequest, "workflow must be Failed/Error to retry")
	}
	err = hydrator.Hydrate(wf)
	if err != nil {
		return nil, err
	}

	newWF := wf.DeepCopy()
	podIf := kubeClient.CoreV1().Pods(wf.ObjectMeta.Namespace)

	// Delete/reset fields which indicate workflow completed
	delete(newWF.Labels, common.LabelKeyCompleted)
	delete(newWF.Labels, common.LabelKeyWorkflowArchivingStatus)
	newWF.Status.Conditions.UpsertCondition(wfv1.Condition{Status: metav1.ConditionFalse, Type: wfv1.ConditionTypeCompleted})
	newWF.ObjectMeta.Labels[common.LabelKeyPhase] = string(wfv1.NodeRunning)
	newWF.Status.Phase = wfv1.WorkflowRunning
	newWF.Status.Nodes = make(wfv1.Nodes)
	newWF.Status.Message = ""
	newWF.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	newWF.Status.FinishedAt = metav1.Time{}
	newWF.Spec.Shutdown = ""
	if newWF.Spec.ActiveDeadlineSeconds != nil && *newWF.Spec.ActiveDeadlineSeconds == 0 {
		// if it was terminated, unset the deadline
		newWF.Spec.ActiveDeadlineSeconds = nil
	}

	onExitNodeName := wf.ObjectMeta.Name + ".onExit"
	// Get all children of nodes that match filter
	nodeIDsToReset, err := getNodeIDsToReset(restartSuccessful, nodeFieldSelector, wf.Status.Nodes)
	if err != nil {
		return nil, err
	}

	// Iterate the previous nodes. If it was successful Pod carry it forward
	deletedNodes := make(map[string]bool)
	for _, node := range wf.Status.Nodes {
		doForceResetNode := false
		if _, present := nodeIDsToReset[node.ID]; present {
			// if we are resetting this node then don't carry it across regardless of its phase
			doForceResetNode = true
		}
		switch node.Phase {
		case wfv1.NodeSucceeded, wfv1.NodeSkipped:
			if !strings.HasPrefix(node.Name, onExitNodeName) && !doForceResetNode {
				newWF.Status.Nodes[node.ID] = node
				continue
			}
		case wfv1.NodeError, wfv1.NodeFailed, wfv1.NodeOmitted:
			if !strings.HasPrefix(node.Name, onExitNodeName) && (node.Type == wfv1.NodeTypeDAG || node.Type == wfv1.NodeTypeTaskGroup || node.Type == wfv1.NodeTypeStepGroup) {
				newNode := node.DeepCopy()
				newNode.Phase = wfv1.NodeRunning
				newNode.Message = ""
				newNode.StartedAt = metav1.Time{Time: time.Now().UTC()}
				newNode.FinishedAt = metav1.Time{}
				newWF.Status.Nodes[newNode.ID] = *newNode
				continue
			} else {
				deletedNodes[node.ID] = true
			}
			// do not add this status to the node. pretend as if this node never existed.
		default:
			// Do not allow retry of workflows with pods in Running/Pending phase
			return nil, errors.InternalErrorf("Workflow cannot be retried with node %s in %s phase", node.Name, node.Phase)
		}
		if node.Type == wfv1.NodeTypePod {
			templateName := getTemplateFromNode(node)
			version := GetWorkflowPodNameVersion(wf)
			podName := PodName(wf.Name, node.Name, templateName, node.ID, version)
			log.Infof("Deleting pod: %s", podName)
			err := podIf.Delete(ctx, podName, metav1.DeleteOptions{})
			if err != nil && !apierr.IsNotFound(err) {
				return nil, errors.InternalWrapError(err)
			}
		} else if node.Name == wf.ObjectMeta.Name {
			newNode := node.DeepCopy()
			newNode.Phase = wfv1.NodeRunning
			newNode.Message = ""
			newNode.StartedAt = metav1.Time{Time: time.Now().UTC()}
			newNode.FinishedAt = metav1.Time{}
			newWF.Status.Nodes[newNode.ID] = *newNode
			continue
		}
	}

	if len(deletedNodes) > 0 {
		for _, node := range newWF.Status.Nodes {
			var newChildren []string
			for _, child := range node.Children {
				if !deletedNodes[child] {
					newChildren = append(newChildren, child)
				}
			}
			node.Children = newChildren

			var outboundNodes []string
			for _, outboundNode := range node.OutboundNodes {
				if !deletedNodes[outboundNode] {
					outboundNodes = append(outboundNodes, outboundNode)
				}
			}
			node.OutboundNodes = outboundNodes

			newWF.Status.Nodes[node.ID] = node
		}
	}

	err = hydrator.Dehydrate(newWF)
	if err != nil {
		return nil, fmt.Errorf("unable to compress or offload workflow nodes: %s", err)
	}

	newWF.Status.StoredTemplates = make(map[string]wfv1.Template)
	for id, tmpl := range wf.Status.StoredTemplates {
		newWF.Status.StoredTemplates[id] = tmpl
	}

	return wfClient.Update(ctx, newWF, metav1.UpdateOptions{})
}

func getTemplateFromNode(node wfv1.NodeStatus) string {
	if node.TemplateRef != nil {
		return node.TemplateRef.Template
	}
	return node.TemplateName
}

func getNodeIDsToReset(restartSuccessful bool, nodeFieldSelector string, nodes wfv1.Nodes) (map[string]bool, error) {
	nodeIDsToReset := make(map[string]bool)
	if !restartSuccessful || len(nodeFieldSelector) == 0 {
		return nodeIDsToReset, nil
	}

	selector, err := fields.ParseSelector(nodeFieldSelector)
	if err != nil {
		return nil, err
	} else {
		for _, node := range nodes {
			if SelectorMatchesNode(selector, node) {
				// traverse all children of the node
				var queue []string
				queue = append(queue, node.ID)

				for len(queue) > 0 {
					childNode := queue[0]
					// if the child isn't already in nodeIDsToReset then we add it and traverse its children
					if _, present := nodeIDsToReset[childNode]; !present {
						nodeIDsToReset[childNode] = true
						queue = append(queue, nodes[childNode].Children...)
					}
					queue = queue[1:]
				}
			}
		}
	}
	return nodeIDsToReset, nil
}

var errSuspendedCompletedWorkflow = errors.Errorf(errors.CodeBadRequest, "cannot suspend completed workflows")

// IsWorkflowSuspended returns whether or not a workflow is considered suspended
func IsWorkflowSuspended(wf *wfv1.Workflow) bool {
	if wf.Spec.Suspend != nil && *wf.Spec.Suspend {
		return true
	}
	for _, node := range wf.Status.Nodes {
		if node.IsActiveSuspendNode() {
			return true
		}
	}
	return false
}

// TerminateWorkflow terminates a workflow by setting its spec.shutdown to ShutdownStrategyTerminate
func TerminateWorkflow(ctx context.Context, wfClient v1alpha1.WorkflowInterface, name string) error {
	return patchShutdownStrategy(ctx, wfClient, name, wfv1.ShutdownStrategyTerminate)
}

// StopWorkflow terminates a workflow by setting its spec.shutdown to ShutdownStrategyStop
// Or terminates a single resume step referenced by nodeFieldSelector
func StopWorkflow(ctx context.Context, wfClient v1alpha1.WorkflowInterface, hydrator hydrator.Interface, name string, nodeFieldSelector string, message string) error {
	if len(nodeFieldSelector) > 0 {
		return updateSuspendedNode(ctx, wfClient, hydrator, name, nodeFieldSelector, SetOperationValues{Phase: wfv1.NodeFailed, Message: message})
	}
	return patchShutdownStrategy(ctx, wfClient, name, wfv1.ShutdownStrategyStop)
}

// patchShutdownStrategy patches the shutdown strategy to a workflow.
func patchShutdownStrategy(ctx context.Context, wfClient v1alpha1.WorkflowInterface, name string, strategy wfv1.ShutdownStrategy) error {
	patchObj := map[string]interface{}{
		"spec": map[string]interface{}{
			"shutdown": strategy,
		},
	}
	var err error
	patch, err := json.Marshal(patchObj)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	err = waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
		_, err := wfClient.Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
		if apierr.IsConflict(err) {
			return false, nil
		}
		return !errorsutil.IsTransientErr(err), err
	})
	return err
}

func SetWorkflow(ctx context.Context, wfClient v1alpha1.WorkflowInterface, hydrator hydrator.Interface, name string, nodeFieldSelector string, values SetOperationValues) error {
	if nodeFieldSelector != "" {
		return updateSuspendedNode(ctx, wfClient, hydrator, name, nodeFieldSelector, values)
	}
	return fmt.Errorf("'set' currently only targets suspend nodes, use a node field selector to target them")
}

// Reads from stdin
func ReadFromStdin() ([]byte, error) {
	reader := bufio.NewReader(os.Stdin)
	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return []byte{}, err
	}
	return body, err
}

// Reads the content of a url
func ReadFromUrl(url string) ([]byte, error) {
	response, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		return nil, err
	}
	return body, err
}

// ReadFromFilePathsOrUrls reads the content of a single or a list of file paths and/or urls
func ReadFromFilePathsOrUrls(filePathsOrUrls ...string) ([][]byte, error) {
	var fileContents [][]byte
	var body []byte
	var err error
	for _, filePathOrUrl := range filePathsOrUrls {
		if cmdutil.IsURL(filePathOrUrl) {
			body, err = ReadFromUrl(filePathOrUrl)
			if err != nil {
				return [][]byte{}, err
			}
		} else {
			body, err = ioutil.ReadFile(filepath.Clean(filePathOrUrl))
			if err != nil {
				return [][]byte{}, err
			}
		}
		fileContents = append(fileContents, body)
	}
	return fileContents, err
}

// ReadManifest reads from stdin, a single file/url, or a list of files and/or urls
func ReadManifest(manifestPaths ...string) ([][]byte, error) {
	var manifestContents [][]byte
	var err error
	if len(manifestPaths) == 1 && manifestPaths[0] == "-" {
		body, err := ReadFromStdin()
		if err != nil {
			return [][]byte{}, err
		}
		manifestContents = append(manifestContents, body)
	} else {
		manifestContents, err = ReadFromFilePathsOrUrls(manifestPaths...)
		if err != nil {
			return [][]byte{}, err
		}
	}
	return manifestContents, err
}

func IsJSONStr(str string) bool {
	str = strings.TrimSpace(str)
	return len(str) > 0 && str[0] == '{'
}

func ConvertYAMLToJSON(str string) (string, error) {
	if !IsJSONStr(str) {
		jsonStr, err := yaml.YAMLToJSON([]byte(str))
		if err != nil {
			return str, err
		}
		return string(jsonStr), nil
	}
	return str, nil
}

// PodSpecPatchMerge will do strategic merge the workflow level PodSpecPatch and template level PodSpecPatch
func PodSpecPatchMerge(wf *wfv1.Workflow, tmpl *wfv1.Template) (string, error) {
	wfPatch, err := ConvertYAMLToJSON(wf.Spec.PodSpecPatch)
	if err != nil {
		return "", err
	}
	tmplPatch, err := ConvertYAMLToJSON(tmpl.PodSpecPatch)
	if err != nil {
		return "", err
	}
	data, err := strategicpatch.StrategicMergePatch([]byte(wfPatch), []byte(tmplPatch), apiv1.PodSpec{})
	return string(data), err
}

func GetNodeType(tmpl *wfv1.Template) wfv1.NodeType {
	return tmpl.GetNodeType()
}

// IsWindowsUNCPath checks if path is prefixed with \\
// This can be used to skip any processing of paths
// that point to SMB shares, local named pipes and local UNC path
func IsWindowsUNCPath(path string, tmpl *wfv1.Template) bool {
	if !HasWindowsOSNodeSelector(tmpl.NodeSelector) && nruntime.GOOS != "windows" {
		return false
	}
	// Check for UNC prefix \\
	if strings.HasPrefix(path, `\\`) {
		return true
	}
	return false
}

func HasWindowsOSNodeSelector(nodeSelector map[string]string) bool {
	if nodeSelector == nil {
		return false
	}

	if platform, keyExists := nodeSelector["kubernetes.io/os"]; keyExists && platform == "windows" {
		return true
	}

	return false
}

func FindWaitCtrIndex(pod *apiv1.Pod) (int, error) {
	waitCtrIndex := -1
	for i, ctr := range pod.Spec.Containers {
		switch ctr.Name {
		case common.WaitContainerName:
			waitCtrIndex = i
		}
	}
	if waitCtrIndex == -1 {
		err := errors.Errorf("-1", "Could not find wait container in pod spec")
		return -1, err
	}
	return waitCtrIndex, nil
}
