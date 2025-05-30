package util

import (
	"bufio"
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	nruntime "runtime"
	"slices"
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
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/workflow/creator"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo-workflows/v3/util/cmd"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/logging"
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
func NewWorkflowInformer(dclient dynamic.Interface, ns string, resyncPeriod time.Duration, tweakListRequestListOptions internalinterfaces.TweakListOptionsFunc, tweakWatchRequestListOptions internalinterfaces.TweakListOptionsFunc, indexers cache.Indexers) cache.SharedIndexInformer {
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
		tweakListRequestListOptions,
		tweakWatchRequestListOptions,
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
	if wf.Labels != nil {
		return wf.Labels[common.LabelKeyCompleted] == "true"
	}
	return false
}

// SubmitWorkflow validates and submits a single workflow and overrides some of the fields of the workflow
func SubmitWorkflow(ctx context.Context, wfIf v1alpha1.WorkflowInterface, wfClientset wfclientset.Interface, namespace string, wf *wfv1.Workflow, wfDefaults *wfv1.Workflow, opts *wfv1.SubmitOpts) (*wfv1.Workflow, error) {
	err := ApplySubmitOpts(wf, opts)
	if err != nil {
		return nil, err
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates())

	err = validate.ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, wfDefaults, validate.ValidateOpts{Submit: true})
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
		var runWf *wfv1.Workflow
		err = waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
			var err error
			runWf, err = wfIf.Create(ctx, wf, metav1.CreateOptions{})
			return !errorsutil.IsTransientErr(err), err
		})
		return runWf, err
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
		passedAnnotations, err := cmdutil.ParseLabels(opts.Annotations)
		if err != nil {
			return fmt.Errorf("expected Annotations of the form: NAME1=VALUE2,NAME2=VALUE2. Received: %s: %w", opts.Labels, err)
		}
		for k, v := range passedAnnotations {
			wfAnnotations[k] = v
		}
	}
	wf.SetAnnotations(wfAnnotations)
	err := overrideParameters(wf, opts.Parameters)
	if err != nil {
		return err
	}
	if opts.GenerateName != "" {
		wf.GenerateName = opts.GenerateName
	}
	if opts.Name != "" {
		wf.Name = opts.Name
	}
	if opts.OwnerReference != nil {
		wf.SetOwnerReferences(append(wf.GetOwnerReferences(), *opts.OwnerReference))
	}
	return nil
}

func overrideParameters(wf *wfv1.Workflow, parameters []string) error {
	if len(parameters) > 0 {
		newParams := make([]wfv1.Parameter, 0)
		passedParams := make(map[string]bool)
		for _, paramStr := range parameters {
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
		if wf.Status.StoredWorkflowSpec != nil {
			wf.Status.StoredWorkflowSpec.Arguments.Parameters = newParams
		}
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
		body, err = os.ReadFile(file)
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
			wf.Spec.Suspend = ptr.To(true)
			creator.LabelActor(ctx, wf, creator.ActionSuspend)
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

func OverrideOutputParametersWithDefault(outputs *wfv1.Outputs) error {
	if outputs == nil {
		return nil
	}
	for i, param := range outputs.Parameters {
		if param.ValueFrom != nil && param.ValueFrom.Supplied != nil {
			if param.ValueFrom.Default != nil {
				outputs.Parameters[i].Value = param.ValueFrom.Default
				outputs.Parameters[i].ValueFrom = nil
			} else {
				return fmt.Errorf("raw output parameter '%s' has not been set and does not have a default value", param.Name)
			}
		}
	}
	return nil
}

// ResumeWorkflow resumes a workflow by setting spec.suspend to nil and any suspended nodes to Successful.
// Retries conflict errors
func ResumeWorkflow(ctx context.Context, wfIf v1alpha1.WorkflowInterface, hydrator hydrator.Interface, workflowName string, nodeFieldSelector string) error {
	uiMsg := ""
	uim := creator.UserInfoMap(ctx)
	if uim != nil {
		uiMsg = fmt.Sprintf("Resumed by: %v", uim)
	}
	if len(nodeFieldSelector) > 0 {
		return updateSuspendedNode(ctx, wfIf, hydrator, workflowName, nodeFieldSelector, SetOperationValues{Phase: wfv1.NodeSucceeded, Message: uiMsg}, creator.ActionResume)
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
					if err := OverrideOutputParametersWithDefault(node.Outputs); err != nil {
						return false, err
					}
					node.Phase = wfv1.NodeSucceeded
					if node.Message != "" {
						uiMsg = node.Message + "; " + uiMsg
					}
					node.Message = uiMsg
					node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
					wf.Status.Nodes.Set(nodeID, node)
					workflowUpdated = true
				}
			}

			if workflowUpdated {
				err := hydrator.Dehydrate(wf)
				if err != nil {
					return false, fmt.Errorf("unable to compress or offload workflow nodes: %s", err)
				}
				creator.LabelActor(ctx, wf, creator.ActionResume)
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
		"templateName": GetTemplateFromNode(node),
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

func AddParamToGlobalScope(ctx context.Context, wf *wfv1.Workflow, log logging.Logger, param wfv1.Parameter) bool {
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
		log.Infof(ctx, "setting %s: '%s'", paramName, param.Value)
		gParam := wfv1.Parameter{Name: param.GlobalName, Value: param.Value}
		wf.Status.Outputs.Parameters = append(wf.Status.Outputs.Parameters, gParam)
		wfUpdated = true
	} else {
		prevVal := wf.Status.Outputs.Parameters[index].Value
		if prevVal == nil || (param.Value != nil && *prevVal != *param.Value) {
			log.Infof(ctx, "overwriting %s: '%s' -> '%s'", paramName, wf.Status.Outputs.Parameters[index].Value, param.Value)
			wf.Status.Outputs.Parameters[index].Value = param.Value
			wfUpdated = true
		}
	}
	return wfUpdated
}

func updateSuspendedNode(ctx context.Context, wfIf v1alpha1.WorkflowInterface, hydrator hydrator.Interface, workflowName string, nodeFieldSelector string, values SetOperationValues, action creator.ActionType) error {
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
									log := logging.NewSlogLogger()
									AddParamToGlobalScope(ctx, wf, log, node.Outputs.Parameters[i])
									break
								}
							}
							if !hit {
								return true, fmt.Errorf("node is not expecting output parameter '%s'", name)
							}
						}
					}
					wf.Status.Nodes.Set(nodeID, node)
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
		creator.LabelActor(ctx, wf, action)
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
func FormulateResubmitWorkflow(ctx context.Context, wf *wfv1.Workflow, memoized bool, parameters []string) (*wfv1.Workflow, error) {
	newWF := wfv1.Workflow{}
	newWF.TypeMeta = wf.TypeMeta

	// Resubmitted workflow will use generated names
	if wf.GenerateName != "" {
		newWF.GenerateName = wf.GenerateName
	} else {
		newWF.GenerateName = wf.Name + "-"
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
		newWF.Name = newWF.GenerateName + RandSuffix()
	}

	// carry over the unmodified spec
	newWF.Spec = wf.Spec

	if newWF.Spec.ActiveDeadlineSeconds != nil && *newWF.Spec.ActiveDeadlineSeconds == 0 {
		// if it was terminated, unset the deadline
		newWF.Spec.ActiveDeadlineSeconds = nil
	}

	newWF.Spec.Shutdown = ""

	// carry over user labels and annotations from previous workflow.
	if newWF.Labels == nil {
		newWF.Labels = make(map[string]string)
	}
	for key, val := range wf.Labels {
		switch key {
		case common.LabelKeyCreator, common.LabelKeyCreatorEmail, common.LabelKeyCreatorPreferredUsername,
			common.LabelKeyPhase, common.LabelKeyCompleted, common.LabelKeyWorkflowArchivingStatus:
			// ignore
		default:
			newWF.Labels[key] = val
		}
	}
	// Apply creator labels based on the authentication information of the current request,
	// regardless of the creator labels of the original Workflow.
	creator.LabelCreator(ctx, &newWF)
	// Append an additional label so it's easy for user to see the
	// name of the original workflow that has been resubmitted.
	newWF.Labels[common.LabelKeyPreviousWorkflowName] = wf.Name
	if newWF.Annotations == nil {
		newWF.Annotations = make(map[string]string)
	}
	for key, val := range wf.Annotations {
		newWF.Annotations[key] = val
	}

	// Setting OwnerReference from original Workflow
	newWF.OwnerReferences = append(newWF.OwnerReferences, wf.OwnerReferences...)

	// Override parameters
	if parameters != nil {
		if _, ok := wf.Labels[common.LabelKeyPreviousWorkflowName]; ok || memoized {
			log.Warnln("Overriding parameters on memoized or resubmitted workflows may have unexpected results")
		}
		err := overrideParameters(&newWF, parameters)
		if err != nil {
			return nil, err
		}
	}

	if !memoized {
		return &newWF, nil
	}

	// Iterate the previous nodes.
	replaceRegexp := regexp.MustCompile("^" + wf.Name)
	newWF.Status.Nodes = make(map[string]wfv1.NodeStatus)
	onExitNodeName := wf.Name + ".onExit"
	err := packer.DecompressWorkflow(wf)
	if err != nil {
		log.Panic(err)
	}
	for _, node := range wf.Status.Nodes {
		newNode := node.DeepCopy()
		if strings.HasPrefix(node.Name, onExitNodeName) {
			continue
		}
		originalID := node.ID
		newNode.Name = replaceRegexp.ReplaceAllString(node.Name, newWF.Name)
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
		} else if newNode.Type == wfv1.NodeTypeSkipped && !isDescendantNodeSucceeded(wf, node, make(map[string]bool)) {
			newWF.Status.Nodes.Delete(newNode.ID)
			continue
		} else {
			newNode.Phase = wfv1.NodePending
			newNode.Message = ""
		}
		newWF.Status.Nodes.Set(newNode.ID, *newNode)
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
	newNodeName := regex.ReplaceAllString(node.Name, newWf.Name)
	return newWf.NodeID(newNodeName)
}

func isDescendantNodeSucceeded(wf *wfv1.Workflow, node wfv1.NodeStatus, nodeIDsToReset map[string]bool) bool {
	for _, child := range node.Children {
		childStatus, err := wf.Status.Nodes.Get(child)
		if err != nil {
			log.Panicf("Coudn't obtain child for %s, panicking", child)
		}
		_, present := nodeIDsToReset[child]
		if (!present && childStatus.Phase == wfv1.NodeSucceeded) || isDescendantNodeSucceeded(wf, *childStatus, nodeIDsToReset) {
			return true
		}
	}
	return false
}

func deletePodNodeDuringRetryWorkflow(wf *wfv1.Workflow, node wfv1.NodeStatus, deletedPods map[string]bool, podsToDelete []string) (map[string]bool, []string) {
	templateName := GetTemplateFromNode(node)
	version := GetWorkflowPodNameVersion(wf)
	podName := GeneratePodName(wf.Name, node.Name, templateName, node.ID, version)
	if _, ok := deletedPods[podName]; !ok {
		deletedPods[podName] = true
		podsToDelete = append(podsToDelete, podName)
	}
	return deletedPods, podsToDelete
}

func createNewRetryWorkflow(wf *wfv1.Workflow, parameters []string) (*wfv1.Workflow, error) {
	newWF := wf.DeepCopy()

	// Delete/reset fields which indicate workflow completed
	delete(newWF.Labels, common.LabelKeyCompleted)
	delete(newWF.Labels, common.LabelKeyWorkflowArchivingStatus)
	newWF.Status.Conditions.UpsertCondition(wfv1.Condition{Status: metav1.ConditionFalse, Type: wfv1.ConditionTypeCompleted})
	newWF.Labels[common.LabelKeyPhase] = string(wfv1.NodeRunning)
	newWF.Status.Phase = wfv1.WorkflowRunning
	newWF.Status.Nodes = make(wfv1.Nodes)
	newWF.Status.Message = ""
	newWF.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	newWF.Status.FinishedAt = metav1.Time{}
	if newWF.Status.StoredWorkflowSpec != nil {
		newWF.Status.StoredWorkflowSpec.Shutdown = ""
	}
	newWF.Spec.Shutdown = ""
	newWF.Status.PersistentVolumeClaims = []apiv1.Volume{}
	if newWF.Spec.ActiveDeadlineSeconds != nil && *newWF.Spec.ActiveDeadlineSeconds == 0 {
		// if it was terminated, unset the deadline
		newWF.Spec.ActiveDeadlineSeconds = nil
	}
	// Override parameters
	if parameters != nil {
		if _, ok := wf.Labels[common.LabelKeyPreviousWorkflowName]; ok {
			log.Warnln("Overriding parameters on resubmitted workflows may have unexpected results")
		}
		err := overrideParameters(newWF, parameters)
		if err != nil {
			return nil, err
		}
	}
	return newWF, nil
}

type dagNode struct {
	n        *wfv1.NodeStatus
	parent   *dagNode
	children []*dagNode
}

func newWorkflowsDag(wf *wfv1.Workflow) ([]*dagNode, error) {
	nodes := make(map[string]*dagNode)
	parentsMap := make(map[string]*wfv1.NodeStatus)

	// create mapping from node to parent
	// as well as creating temp mappings from nodeID to node
	for _, wfNode := range wf.Status.Nodes {
		n := dagNode{}
		n.n = &wfNode
		nodes[wfNode.ID] = &n
		for _, child := range wfNode.Children {
			parentsMap[child] = &wfNode
		}
	}

	for _, wfNode := range wf.Status.Nodes {
		parentWfNode, ok := parentsMap[wfNode.ID]
		if !ok && wfNode.Name != wf.Name && !strings.HasPrefix(wfNode.Name, wf.Name+".onExit") {
			return nil, fmt.Errorf("couldn't find parent node for %s", wfNode.ID)
		}

		var parentNode *dagNode
		if parentWfNode != nil {
			parentNode = nodes[parentWfNode.ID]
		}

		children := make([]*dagNode, 0)

		for _, childID := range wfNode.Children {
			childNode, ok := nodes[childID]
			if !ok {
				return nil, fmt.Errorf("coudln't obtain child %s", childID)
			}
			children = append(children, childNode)
		}
		nodes[wfNode.ID].parent = parentNode
		nodes[wfNode.ID].children = children
	}

	values := make([]*dagNode, 0)
	for _, v := range nodes {
		values = append(values, v)
	}
	return values, nil
}

func singularPath(nodes []*dagNode, toNode string) ([]*dagNode, error) {
	if len(nodes) <= 0 {
		return nil, fmt.Errorf("expected at least 1 node")
	}
	var root *dagNode
	var leaf *dagNode
	for i := range nodes {
		if nodes[i].n.ID == toNode {
			leaf = nodes[i]
		}
		if nodes[i].parent == nil {
			root = nodes[i]
		}
	}

	if root == nil {
		return nil, fmt.Errorf("was unable to find root")
	}

	if leaf == nil {
		return nil, fmt.Errorf("was unable to find %s", toNode)
	}

	curr := leaf

	reverseNodes := make([]*dagNode, 0)
	for {
		reverseNodes = append(reverseNodes, curr)
		if curr.n.ID == root.n.ID {
			break
		}
		if curr.parent == nil {
			return nil, fmt.Errorf("parent was nil but curr is not the root node")
		}
		curr = curr.parent
	}

	slices.Reverse(reverseNodes)
	return reverseNodes, nil
}

func getChildren(n *dagNode) map[string]bool {
	children := make(map[string]bool)
	queue := list.New()
	queue.PushBack(n)
	for {
		currNode := queue.Front()
		if currNode == nil {
			break
		}

		curr := currNode.Value.(*dagNode)
		for i := range curr.children {
			children[curr.children[i].n.ID] = true
			queue.PushBack(curr.children[i])
		}
		queue.Remove(currNode)
	}
	return children
}

type resetFn func(string)
type deleteFn func(string)
type matchFn func(*dagNode) bool

func matchNodeType(nodeType wfv1.NodeType) matchFn {
	return func(n *dagNode) bool {
		return n.n.Type == nodeType
	}
}

func resetUntil(n *dagNode, matchFunc matchFn, resetFunc resetFn) (*dagNode, error) {
	curr := n
	for {
		if curr == nil {
			return nil, fmt.Errorf("was seeking node but ran out of nodes to explore")
		}

		if match := matchFunc(curr); match {
			resetFunc(curr.n.ID)
			return curr, nil
		}
		curr = curr.parent
	}
}

func matchBoundaryID(boundaryID string) matchFn {
	return func(n *dagNode) bool {
		return n.n.ID == boundaryID
	}
}

func resetBoundaries(n *dagNode, resetFunc resetFn) (*dagNode, error) {
	curr := n
	for {
		if curr == nil {
			return curr, nil
		}
		if curr.parent != nil && curr.parent.n.Type == wfv1.NodeTypeRetry {
			resetFunc(curr.parent.n.ID)
			curr = curr.parent
		}
		if curr.parent != nil && curr.parent.n.Type == wfv1.NodeTypeStepGroup {
			resetFunc(curr.parent.n.ID)
		}
		seekingBoundaryID := curr.n.BoundaryID
		if seekingBoundaryID == "" {
			return curr.parent, nil
		}
		var err error
		curr, err = resetUntil(curr, matchBoundaryID(seekingBoundaryID), resetFunc)
		if err != nil {
			return nil, err
		}
	}
}

// resetPod is only called in the event a Container was found. This implies that there is a parent pod.
func resetPod(n *dagNode, resetFunc resetFn, addToDelete deleteFn) (*dagNode, error) {
	// this sets to reset but resets are overridden by deletes in the final FormulateRetryWorkflow logic.
	curr, err := resetUntil(n, matchNodeType(wfv1.NodeTypePod), resetFunc)
	if err != nil {
		return nil, err
	}
	addToDelete(curr.n.ID)
	children := getChildren(curr)
	for childID := range children {
		addToDelete(childID)
	}
	return curr, nil
}

func resetPath(allNodes []*dagNode, startNode string) (map[string]bool, map[string]bool, error) {
	nodes, err := singularPath(allNodes, startNode)

	curr := nodes[len(nodes)-1]
	if len(nodes) > 0 {
		// remove startNode
		nodes = nodes[:len(nodes)-1]
	}

	nodesToDelete := getChildren(curr)
	nodesToDelete[curr.n.ID] = true

	nodesToReset := make(map[string]bool)

	if err != nil {
		return nil, nil, err
	}
	l := len(nodes)
	if l <= 0 {
		return nodesToReset, nodesToDelete, nil
	}

	// safe to reset the startNode since deletions
	// override resets.
	addToReset := func(nodeID string) {
		nodesToReset[nodeID] = true
	}

	addToDelete := func(nodeID string) {
		nodesToDelete[nodeID] = true
	}

	for curr != nil {

		switch {
		case isGroupNodeType(curr.n.Type):
			addToReset(curr.n.ID)
			curr, err = resetBoundaries(curr, addToReset)
			if err != nil {
				return nil, nil, err
			}
			continue
		case curr.n.Type == wfv1.NodeTypeRetry:
			addToReset(curr.n.ID)
		case curr.n.Type == wfv1.NodeTypeContainer:
			curr, err = resetPod(curr, addToReset, addToDelete)
			if err != nil {
				return nil, nil, err
			}
			continue
		}

		curr = curr.parent
	}
	return nodesToReset, nodesToDelete, nil
}

func setUnion[T comparable](m1 map[T]bool, m2 map[T]bool) map[T]bool {
	res := make(map[T]bool)

	for k, v := range m1 {
		res[k] = v
	}

	for k, v := range m2 {
		if _, ok := m1[k]; !ok {
			res[k] = v
		}
	}
	return res
}

func isGroupNodeType(nodeType wfv1.NodeType) bool {
	return nodeType == wfv1.NodeTypeDAG || nodeType == wfv1.NodeTypeTaskGroup || nodeType == wfv1.NodeTypeStepGroup || nodeType == wfv1.NodeTypeSteps
}

func isExecutionNodeType(nodeType wfv1.NodeType) bool {
	return nodeType == wfv1.NodeTypeContainer || nodeType == wfv1.NodeTypePod || nodeType == wfv1.NodeTypeHTTP || nodeType == wfv1.NodeTypePlugin
}

// dagSortedNodes sorts the nodes based on topological order, omits onExitNode
func dagSortedNodes(nodes []*dagNode, rootNodeName string) []*dagNode {
	sortedNodes := make([]*dagNode, 0)

	if len(nodes) == 0 {
		return sortedNodes
	}

	queue := make([]*dagNode, 0)

	for _, n := range nodes {
		if n.n.Name == rootNodeName {
			queue = append(queue, n)
			break
		}
	}

	if len(queue) != 1 {
		panic("couldn't find root node")
	}

	for len(queue) > 0 {
		curr := queue[0]
		sortedNodes = append(sortedNodes, curr)
		queue = queue[1:]
		queue = append(queue, curr.children...)
	}

	return sortedNodes
}

// FormulateRetryWorkflow attempts to retry a workflow
// The logic is as follows:
// create a DAG
// topological sort
// iterate through all must delete nodes: iterator $node
// obtain singular path to each $node
// reset all "reset points" to $node
func FormulateRetryWorkflow(ctx context.Context, wf *wfv1.Workflow, restartSuccessful bool, nodeFieldSelector string, parameters []string) (*wfv1.Workflow, []string, error) {

	switch wf.Status.Phase {
	case wfv1.WorkflowFailed, wfv1.WorkflowError:
	case wfv1.WorkflowSucceeded:
		if !restartSuccessful || len(nodeFieldSelector) <= 0 {
			return nil, nil, errors.Errorf(errors.CodeBadRequest, "To retry a succeeded workflow, set the options restartSuccessful and nodeFieldSelector")
		}
	default:
		return nil, nil, errors.Errorf(errors.CodeBadRequest, "Cannot retry a workflow in phase %s", wf.Status.Phase)
	}

	onExitNodeName := wf.Name + ".onExit"

	newWf, err := createNewRetryWorkflow(wf, parameters)
	if err != nil {
		return nil, nil, err
	}

	deleteNodesMap, err := getNodeIDsToReset(restartSuccessful, nodeFieldSelector, wf.Status.Nodes)
	if err != nil {
		return nil, nil, err
	}

	failed := make(map[string]bool)
	for nodeID, node := range wf.Status.Nodes {
		if node.FailedOrError() && isExecutionNodeType(node.Type) {
			// Check its parent if current node is retry node
			if node.NodeFlag != nil && node.NodeFlag.Retried {
				node = *wf.Status.Nodes.Find(func(nodeStatus wfv1.NodeStatus) bool {
					return nodeStatus.HasChild(node.ID)
				})
			}
			if !isDescendantNodeSucceeded(wf, node, deleteNodesMap) {
				failed[nodeID] = true
			}
		}
	}
	for failedNode := range failed {
		deleteNodesMap[failedNode] = true
	}

	nodes, err := newWorkflowsDag(wf)
	if err != nil {
		return nil, nil, err
	}

	toReset := make(map[string]bool)
	toDelete := make(map[string]bool)

	nodesMap := make(map[string]*dagNode)
	for i := range nodes {
		nodesMap[nodes[i].n.ID] = nodes[i]
	}

	nodes = dagSortedNodes(nodes, wf.Name)

	deleteNodes := make([]*dagNode, 0)

	// deleteNodes will not contain an exit node
	for i := range nodes {
		if _, ok := deleteNodesMap[nodes[i].n.ID]; ok {
			deleteNodes = append(deleteNodes, nodes[i])
		}
	}

	// this is kind of complex
	// we rely on deleteNodes being topologically sorted.
	// this is done via a breadth first search on the dag via `dagSortedNodes`
	// This is because nodes at the top take precedence over nodes at the bottom.
	// if a failed node was declared to be scheduled for deletion, we should
	// never execute resetPath on that node.
	// we ensure this behaviour via calling resetPath in topological order.
	for i := range deleteNodes {
		currNode := deleteNodes[i]
		shouldDelete := toDelete[currNode.n.ID]
		if shouldDelete {
			continue
		}
		pathToReset, pathToDelete, err := resetPath(nodes, currNode.n.ID)
		if err != nil {
			return nil, nil, err
		}
		toReset = setUnion(toReset, pathToReset)
		toDelete = setUnion(toDelete, pathToDelete)
	}

	for nodeID := range toReset {
		// avoid resetting nodes that are marked for deletion
		if in := toDelete[nodeID]; in {
			continue
		}

		n := wf.Status.Nodes[nodeID]

		newWf.Status.Nodes.Set(nodeID, resetNode(*n.DeepCopy()))
	}

	deletedPods := make(map[string]bool)
	podsToDelete := []string{}

	for nodeID := range toDelete {
		n := wf.Status.Nodes[nodeID]
		if n.Type == wfv1.NodeTypePod {
			deletedPods, podsToDelete = deletePodNodeDuringRetryWorkflow(wf, n, deletedPods, podsToDelete)
		}
	}

	for id, n := range wf.Status.Nodes {
		shouldDelete := toDelete[id] || strings.HasPrefix(n.Name, onExitNodeName)
		if _, err := newWf.Status.Nodes.Get(id); err != nil && !shouldDelete {
			newWf.Status.Nodes.Set(id, *n.DeepCopy())
		}
		if n.Name == onExitNodeName {
			queue := list.New()
			queue.PushBack(&n)
			for {
				currNode := queue.Front()
				if currNode == nil {
					break
				}
				curr := currNode.Value.(*wfv1.NodeStatus)
				deletedPods, podsToDelete = deletePodNodeDuringRetryWorkflow(wf, *curr, deletedPods, podsToDelete)
				for i := range curr.Children {
					child, err := wf.Status.Nodes.Get(curr.Children[i])
					if err != nil {
						return nil, nil, err
					}
					queue.PushBack(child)
				}
				queue.Remove(currNode)
			}
		}
	}
	for id, oldWfNode := range wf.Status.Nodes {

		if !newWf.Status.Nodes.Has(id) {
			continue
		}

		newChildren := []string{}
		for _, childID := range oldWfNode.Children {
			if toDelete[childID] {
				continue
			}
			newChildren = append(newChildren, childID)
		}
		newOutboundNodes := []string{}

		for _, outBoundNodeID := range oldWfNode.OutboundNodes {
			if toDelete[outBoundNodeID] {
				continue
			}
			newOutboundNodes = append(newOutboundNodes, outBoundNodeID)
		}

		wfNode := newWf.Status.Nodes[id]
		wfNode.Children = newChildren
		wfNode.OutboundNodes = newOutboundNodes
		newWf.Status.Nodes.Set(id, *wfNode.DeepCopy())
	}

	return newWf, podsToDelete, nil
}

func resetNode(node wfv1.NodeStatus) wfv1.NodeStatus {
	// The previously supplied parameters needed to be reset. Otherwise, `argo node reset` would not work as expected.
	if node.Type == wfv1.NodeTypeSuspend {
		if node.Outputs != nil {
			for i, param := range node.Outputs.Parameters {
				node.Outputs.Parameters[i] = wfv1.Parameter{
					Name:      param.Name,
					Value:     nil,
					ValueFrom: &wfv1.ValueFrom{Supplied: &wfv1.SuppliedValueFrom{}},
				}
			}
		}
	}
	if node.Phase == wfv1.NodeSkipped {
		// The skipped nodes need to be kept as skipped. Otherwise, the workflow will be stuck on running.
		node.Phase = wfv1.NodeSkipped
	} else {
		node.Phase = wfv1.NodeRunning
	}
	node.Message = ""
	node.StartedAt = metav1.Time{Time: time.Now().UTC()}
	node.FinishedAt = metav1.Time{}
	return node
}

func GetTemplateFromNode(node wfv1.NodeStatus) string {
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
	}
	for _, node := range nodes {
		if SelectorMatchesNode(selector, node) {
			nodeIDsToReset[node.ID] = true
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
		return updateSuspendedNode(ctx, wfClient, hydrator, name, nodeFieldSelector, SetOperationValues{Phase: wfv1.NodeFailed, Message: message}, creator.ActionStop)
	}
	return patchShutdownStrategy(ctx, wfClient, name, wfv1.ShutdownStrategyStop)
}

type AlreadyShutdownError struct {
	workflowName string
	namespace    string
}

func (e AlreadyShutdownError) Error() string {
	return fmt.Sprintf("cannot shutdown a completed workflow: workflow: %q, namespace: %q", e.workflowName, e.namespace)
}

// patchShutdownStrategy patches the shutdown strategy to a workflow.
func patchShutdownStrategy(ctx context.Context, wfClient v1alpha1.WorkflowInterface, name string, strategy wfv1.ShutdownStrategy) error {
	patchObj := map[string]interface{}{
		"spec": map[string]interface{}{
			"shutdown": strategy,
		},
	}
	var action creator.ActionType
	switch strategy {
	case wfv1.ShutdownStrategyTerminate:
		action = creator.ActionTerminate
	case wfv1.ShutdownStrategyStop:
		action = creator.ActionStop
	default:
		action = creator.ActionNone
	}
	userActionLabel := creator.UserActionLabel(ctx, action)
	if userActionLabel != nil {
		patchObj["metadata"] = map[string]interface{}{
			"labels": userActionLabel,
		}
	}
	var err error
	patch, err := json.Marshal(patchObj)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	err = waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
		wf, err := wfClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return !errorsutil.IsTransientErr(err), err
		}
		if wf.Status.Fulfilled() {
			return true, AlreadyShutdownError{wf.Name, wf.Namespace}
		}
		_, err = wfClient.Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
		if apierr.IsConflict(err) {
			return false, nil
		}
		return !errorsutil.IsTransientErr(err), err
	})
	return err
}

func SetWorkflow(ctx context.Context, wfClient v1alpha1.WorkflowInterface, hydrator hydrator.Interface, name string, nodeFieldSelector string, values SetOperationValues) error {
	if nodeFieldSelector != "" {
		return updateSuspendedNode(ctx, wfClient, hydrator, name, nodeFieldSelector, values, creator.ActionNone)
	}
	return fmt.Errorf("'set' currently only targets suspend nodes, use a node field selector to target them")
}

// Reads from stdin
func ReadFromStdin() ([]byte, error) {
	reader := bufio.NewReader(os.Stdin)
	body, err := io.ReadAll(reader)
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
	body, err := io.ReadAll(response.Body)
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
			body, err = os.ReadFile(filepath.Clean(filePathOrUrl))
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

func ApplyPodSpecPatch(podSpec apiv1.PodSpec, podSpecPatchYamls ...string) (*apiv1.PodSpec, error) {
	podSpecJson, err := json.Marshal(podSpec)
	if err != nil {
		return nil, errors.Wrap(err, "", "Failed to marshal the Pod spec")
	}

	for _, podSpecPatchYaml := range podSpecPatchYamls {
		// must convert to json because PodSpec has only json tags
		podSpecPatchJson, err := ConvertYAMLToJSON(podSpecPatchYaml)
		if err != nil {
			return nil, errors.Wrap(err, "", "Failed to convert the PodSpecPatch yaml to json")
		}

		// validate the patch to be a PodSpec
		if err := json.Unmarshal([]byte(podSpecPatchJson), &apiv1.PodSpec{}); err != nil {
			return nil, fmt.Errorf("invalid podSpecPatch %q: %w", podSpecPatchYaml, err)
		}

		podSpecJson, err = strategicpatch.StrategicMergePatch(podSpecJson, []byte(podSpecPatchJson), apiv1.PodSpec{})
		if err != nil {
			return nil, errors.Wrap(err, "", "Error occurred during strategic merge patch")
		}
	}

	var newPodSpec apiv1.PodSpec
	err = json.Unmarshal(podSpecJson, &newPodSpec)
	if err != nil {
		return nil, errors.Wrap(err, "", "Error in Unmarshalling after merge the patch")
	}
	return &newPodSpec, nil
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
