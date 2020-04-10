package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/util/retry"
	unstructutil "github.com/argoproj/argo/util/unstructured"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/packer"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/validate"
)

// NewWorkflowInformer returns the workflow informer used by the controller. This is actually
// a custom built UnstructuredInformer which is in actuality returning unstructured.Unstructured
// objects. We no longer return WorkflowInformer due to:
// https://github.com/kubernetes/kubernetes/issues/57705
// https://github.com/argoproj/argo/issues/632
func NewWorkflowInformer(cfg *rest.Config, ns string, resyncPeriod time.Duration, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	dclient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

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
		cache.Indexers{},
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

// FromUnstructured converts an unstructured object to a workflow
func FromUnstructured(un *unstructured.Unstructured) (*wfv1.Workflow, error) {
	var wf wfv1.Workflow
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, &wf)
	if wf.Spec.TTLSecondsAfterFinished != nil {
		if wf.Spec.TTLStrategy == nil {
			ttlstrategy := wfv1.TTLStrategy{SecondsAfterCompletion: wf.Spec.TTLSecondsAfterFinished}
			wf.Spec.TTLStrategy = &ttlstrategy
		} else if wf.Spec.TTLStrategy.SecondsAfterCompletion == nil {
			wf.Spec.TTLStrategy.SecondsAfterCompletion = wf.Spec.TTLSecondsAfterFinished
		}
	}
	return &wf, err
}

// ToUnstructured converts an workflow to an Unstructured object
func ToUnstructured(wf *wfv1.Workflow) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(wf)
	return &unstructured.Unstructured{Object: obj}, err
}

// IsWorkflowCompleted returns whether or not a workflow is considered completed
func IsWorkflowCompleted(wf *wfv1.Workflow) bool {
	if wf.ObjectMeta.Labels != nil {
		return wf.ObjectMeta.Labels[common.LabelKeyCompleted] == "true"
	}
	return false
}

// SubmitOpts are workflow submission options
type SubmitOpts struct {
	Name           string                 // --name
	GenerateName   string                 // --generate-name
	InstanceID     string                 // --instanceid
	Entrypoint     string                 // --entrypoint
	Parameters     []string               // --parameter
	ParameterFile  string                 // --parameter-file
	ServiceAccount string                 // --serviceaccount
	DryRun         bool                   // --dry-run
	ServerDryRun   bool                   // --server-dry-run
	Labels         string                 // --labels
	OwnerReference *metav1.OwnerReference // useful if your custom controller creates argo workflow resources
}

// SubmitWorkflow validates and submit a single workflow and override some of the fields of the workflow
func SubmitWorkflow(wfIf v1alpha1.WorkflowInterface, wfClientset wfclientset.Interface, namespace string, wf *wfv1.Workflow, opts *SubmitOpts) (*wfv1.Workflow, error) {

	err := ApplySubmitOpts(wf, opts)
	if err != nil {
		return nil, err
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(namespace))
	_, err = validate.ValidateWorkflow(wftmplGetter, wf, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}
	if opts.DryRun {
		return wf, nil
	} else if opts.ServerDryRun {
		wf, err := CreateServerDryRun(wf, wfClientset)
		if err != nil {
			return nil, err
		}
		return wf, err
	} else {
		return wfIf.Create(wf)
	}
}

// CreateServerDryRun fills the workflow struct with the server's representation without creating it and returns an error, if there is any
func CreateServerDryRun(wf *wfv1.Workflow, wfClientset wfclientset.Interface) (*wfv1.Workflow, error) {
	// Keep the workflow metadata because it will be overwritten by the Post request
	workflowTypeMeta := wf.TypeMeta
	err := wfClientset.ArgoprojV1alpha1().RESTClient().Post().
		Namespace(wf.Namespace).
		Resource("workflows").
		Body(wf).
		Param("dryRun", "All").
		Do().
		Into(wf)
	wf.TypeMeta = workflowTypeMeta
	return wf, err
}

// Apply the Submit options into workflow object
func ApplySubmitOpts(wf *wfv1.Workflow, opts *SubmitOpts) error {
	if opts == nil {
		opts = &SubmitOpts{}
	}
	if opts.Entrypoint != "" {
		wf.Spec.Entrypoint = opts.Entrypoint
	}
	if opts.ServiceAccount != "" {
		wf.Spec.ServiceAccountName = opts.ServiceAccount
	}
	labels := wf.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	if opts.Labels != "" {
		passedLabels, err := cmdutil.ParseLabels(opts.Labels)
		if err != nil {
			return fmt.Errorf("Expected labels of the form: NAME1=VALUE2,NAME2=VALUE2. Received: %s", opts.Labels)
		}
		for k, v := range passedLabels {
			labels[k] = v
		}
	}
	if opts.InstanceID != "" {
		labels[common.LabelKeyControllerInstanceID] = opts.InstanceID
	}
	wf.SetLabels(labels)
	if len(opts.Parameters) > 0 || opts.ParameterFile != "" {
		newParams := make([]wfv1.Parameter, 0)
		passedParams := make(map[string]bool)
		for _, paramStr := range opts.Parameters {
			parts := strings.SplitN(paramStr, "=", 2)
			if len(parts) == 1 {
				return fmt.Errorf("Expected parameter of the form: NAME=VALUE. Received: %s", paramStr)
			}
			param := wfv1.Parameter{
				Name:  parts[0],
				Value: &parts[1],
			}
			newParams = append(newParams, param)
			passedParams[param.Name] = true
		}

		// Add parameters from a parameter-file, if one was provided
		if opts.ParameterFile != "" {
			var body []byte
			var err error
			if cmdutil.IsURL(opts.ParameterFile) {
				body, err = ReadFromUrl(opts.ParameterFile)
				if err != nil {
					return errors.InternalWrapError(err)
				}
			} else {
				body, err = ioutil.ReadFile(opts.ParameterFile)
				if err != nil {
					return errors.InternalWrapError(err)
				}
			}

			yamlParams := map[string]json.RawMessage{}
			err = yaml.Unmarshal(body, &yamlParams)
			if err != nil {
				return errors.InternalWrapError(err)
			}

			for k, v := range yamlParams {
				// We get quoted strings from the yaml file.
				value, err := strconv.Unquote(string(v))
				if err != nil {
					// the string is already clean.
					value = string(v)
				}
				param := wfv1.Parameter{
					Name:  k,
					Value: &value,
				}
				if _, ok := passedParams[param.Name]; ok {
					// this parameter was overridden via command line
					continue
				}
				newParams = append(newParams, param)
				passedParams[param.Name] = true
			}
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

// SuspendWorkflow suspends a workflow by setting spec.suspend to true. Retries conflict errors
func SuspendWorkflow(wfIf v1alpha1.WorkflowInterface, workflowName string) error {
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		wf, err := wfIf.Get(workflowName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if IsWorkflowCompleted(wf) {
			return false, errSuspendedCompletedWorkflow
		}
		if wf.Spec.Suspend == nil || !*wf.Spec.Suspend {
			wf.Spec.Suspend = pointer.BoolPtr(true)
			_, err = wfIf.Update(wf)
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

// ResumeWorkflow resumes a workflow by setting spec.suspend to nil and any suspended nodes to Successful.
// Retries conflict errors
func ResumeWorkflow(wfIf v1alpha1.WorkflowInterface, repo sqldb.OffloadNodeStatusRepo, workflowName string, nodeFieldSelector string) error {
	if len(nodeFieldSelector) > 0 {
		return updateWorkflowNodeByKey(wfIf, workflowName, nodeFieldSelector, wfv1.NodeSucceeded, "")
	} else {
		err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
			wf, err := wfIf.Get(workflowName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			err = packer.DecompressWorkflow(wf)
			if err != nil {
				return false, fmt.Errorf("unable to decompress workflow: %s", err)
			}

			workflowUpdated := false
			if wf.Spec.Suspend != nil && *wf.Spec.Suspend {
				wf.Spec.Suspend = nil
				workflowUpdated = true
			}

			nodes := wf.Status.Nodes
			if wf.Status.IsOffloadNodeStatus() {
				if !repo.IsEnabled() {
					return false, fmt.Errorf(sqldb.OffloadNodeStatusDisabled)
				}
				var err error
				nodes, err = repo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
				if err != nil {
					return false, fmt.Errorf("unable to retrieve offloaded nodes: %s", err)
				}
			}
			newNodes := nodes.DeepCopy()

			// To resume a workflow with a suspended node we simply mark the node as Successful
			for nodeID, node := range nodes {
				if node.IsActiveSuspendNode() {
					node.Phase = wfv1.NodeSucceeded
					node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
					newNodes[nodeID] = node
					workflowUpdated = true
				}
			}

			if workflowUpdated {
				if wf.Status.IsOffloadNodeStatus() {
					if !repo.IsEnabled() {
						return false, fmt.Errorf(sqldb.OffloadNodeStatusDisabled)
					}
					offloadVersion, err := repo.Save(string(wf.UID), wf.Namespace, newNodes)
					if err != nil {
						return false, fmt.Errorf("unable to save offloaded nodes: %s", err)
					}
					wf.Status.OffloadNodeStatusVersion = offloadVersion
					wf.Status.CompressedNodes = ""
					wf.Status.Nodes = nil
				} else {
					wf.Status.Nodes = newNodes
				}

				err = packer.CompressWorkflowIfNeeded(wf)
				if err != nil {
					return false, fmt.Errorf("unable to compress workflow: %s", err)
				}

				_, err = wfIf.Update(wf)
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

func updateWorkflowNodeByKey(wfIf v1alpha1.WorkflowInterface, workflowName string, nodeFieldSelector string, phase wfv1.NodePhase, message string) error {
	selector, err := fields.ParseSelector(nodeFieldSelector)

	if err != nil {
		return err
	}
	err = wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		wf, err := wfIf.Get(workflowName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		err = packer.DecompressWorkflow(wf)
		if err != nil {
			log.Fatal(err)
		}

		nodeUpdated := false
		for nodeID, node := range wf.Status.Nodes {
			if node.IsActiveSuspendNode() {
				nodeFields := fields.Set{
					"displayName": node.DisplayName,
				}
				if node.Inputs != nil {
					for _, inParam := range node.Inputs.Parameters {
						nodeFields[fmt.Sprintf("inputs.parameters.%s.value", inParam.Name)] = *inParam.Value
					}
				}

				if selector.Matches(nodeFields) {
					node.Phase = phase
					node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
					if len(message) > 0 {
						node.Message = message
					}
					wf.Status.Nodes[nodeID] = node
					nodeUpdated = true
				}
			}
		}
		if nodeUpdated {
			_, err = wfIf.Update(wf)
			if err != nil {
				if apierr.IsConflict(err) {
					return false, nil
				}
				return false, err
			}
		} else {
			return true, fmt.Errorf("No nodes matching nodeFieldSelector: %s", nodeFieldSelector)
		}
		return true, nil
	})
	return err
}

const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
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
		case wfv1.NodeFailed, wfv1.NodeError:
		default:
			return nil, errors.Errorf(errors.CodeBadRequest, "workflow must be Failed/Error to resubmit in memoized mode")
		}
		newWF.ObjectMeta.Name = newWF.ObjectMeta.GenerateName + randString(5)
	}

	// carry over the unmodified spec
	newWF.Spec = wf.Spec

	if newWF.Spec.ActiveDeadlineSeconds != nil && *newWF.Spec.ActiveDeadlineSeconds == 0 {
		// if it was terminated, unset the deadline
		newWF.Spec.ActiveDeadlineSeconds = nil
	}

	newWF.Spec.Shutdown = ""

	// carry over user labels and annotations from previous workflow.
	// skip any argoproj.io labels except for the controller instanceID label.
	for key, val := range wf.ObjectMeta.Labels {
		if strings.HasPrefix(key, workflow.WorkflowFullName+"/") && key != common.LabelKeyControllerInstanceID {
			continue
		}
		if newWF.ObjectMeta.Labels == nil {
			newWF.ObjectMeta.Labels = make(map[string]string)
		}
		newWF.ObjectMeta.Labels[key] = val
	}
	for key, val := range wf.ObjectMeta.Annotations {
		if newWF.ObjectMeta.Annotations == nil {
			newWF.ObjectMeta.Annotations = make(map[string]string)
		}
		newWF.ObjectMeta.Annotations[key] = val
	}

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
		if !newNode.Successful() && newNode.Type == wfv1.NodeTypePod {
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
		if newNode.Successful() && newNode.Type == wfv1.NodeTypePod {
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

	newWF.Status.Conditions.UpsertCondition(wfv1.WorkflowCondition{Status: metav1.ConditionFalse, Type: wfv1.WorkflowConditionCompleted})
	newWF.Status.Phase = wfv1.NodePending

	return &newWF, nil
}

// convertNodeID converts an old nodeID to a new nodeID
func convertNodeID(newWf *wfv1.Workflow, regex *regexp.Regexp, oldNodeID string, oldNodes map[string]wfv1.NodeStatus) string {
	node := oldNodes[oldNodeID]
	newNodeName := regex.ReplaceAllString(node.Name, newWf.ObjectMeta.Name)
	return newWf.NodeID(newNodeName)
}

// RetryWorkflow updates a workflow, deleting all failed steps as well as the onExit node (and children)
func RetryWorkflow(kubeClient kubernetes.Interface, repo sqldb.OffloadNodeStatusRepo, wfClient v1alpha1.WorkflowInterface, wf *wfv1.Workflow) (*wfv1.Workflow, error) {
	switch wf.Status.Phase {
	case wfv1.NodeFailed, wfv1.NodeError:
	default:
		return nil, errors.Errorf(errors.CodeBadRequest, "workflow must be Failed/Error to retry")
	}

	err := packer.DecompressWorkflow(wf)
	if err != nil {
		return nil, fmt.Errorf("unable to decompress workflow: %s", err)
	}

	newWF := wf.DeepCopy()
	podIf := kubeClient.CoreV1().Pods(wf.ObjectMeta.Namespace)

	// Delete/reset fields which indicate workflow completed
	delete(newWF.Labels, common.LabelKeyCompleted)
	newWF.Status.Conditions.UpsertCondition(wfv1.WorkflowCondition{Status: metav1.ConditionFalse, Type: wfv1.WorkflowConditionCompleted})
	newWF.ObjectMeta.Labels[common.LabelKeyPhase] = string(wfv1.NodeRunning)
	newWF.Status.Phase = wfv1.NodeRunning
	newWF.Status.Message = ""
	newWF.Status.FinishedAt = metav1.Time{}
	if newWF.Spec.ActiveDeadlineSeconds != nil && *newWF.Spec.ActiveDeadlineSeconds == 0 {
		// if it was terminated, unset the deadline
		newWF.Spec.ActiveDeadlineSeconds = nil
	}

	// Iterate the previous nodes. If it was successful Pod carry it forward
	newNodes := make(map[string]wfv1.NodeStatus)
	onExitNodeName := wf.ObjectMeta.Name + ".onExit"
	nodes := wf.Status.Nodes
	if wf.Status.IsOffloadNodeStatus() {
		if !repo.IsEnabled() {
			return nil, fmt.Errorf(sqldb.OffloadNodeStatusDisabled)
		}
		var err error
		nodes, err = repo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve offloaded nodes: %s", err)
		}
	}

	for _, node := range nodes {
		switch node.Phase {
		case wfv1.NodeSucceeded, wfv1.NodeSkipped:
			if !strings.HasPrefix(node.Name, onExitNodeName) {
				newNodes[node.ID] = node
				continue
			}
		case wfv1.NodeError, wfv1.NodeFailed:
			if !strings.HasPrefix(node.Name, onExitNodeName) && (node.Type == wfv1.NodeTypeDAG || node.Type == wfv1.NodeTypeStepGroup) {
				newNode := node.DeepCopy()
				newNode.Phase = wfv1.NodeRunning
				newNode.Message = ""
				newNode.FinishedAt = metav1.Time{}
				newNodes[newNode.ID] = *newNode
				continue
			}
			// do not add this status to the node. pretend as if this node never existed.
		default:
			// Do not allow retry of workflows with pods in Running/Pending phase
			return nil, errors.InternalErrorf("Workflow cannot be retried with node %s in %s phase", node.Name, node.Phase)
		}
		if node.Type == wfv1.NodeTypePod {
			log.Infof("Deleting pod: %s", node.ID)
			err := podIf.Delete(node.ID, &metav1.DeleteOptions{})
			if err != nil && !apierr.IsNotFound(err) {
				return nil, errors.InternalWrapError(err)
			}
		} else if node.Name == wf.ObjectMeta.Name {
			newNode := node.DeepCopy()
			newNode.Phase = wfv1.NodeRunning
			newNode.Message = ""
			newNode.FinishedAt = metav1.Time{}
			newNodes[newNode.ID] = *newNode
			continue
		}
	}

	if wf.Status.IsOffloadNodeStatus() {
		if !repo.IsEnabled() {
			return nil, fmt.Errorf(sqldb.OffloadNodeStatusDisabled)
		}
		offloadVersion, err := repo.Save(string(newWF.UID), newWF.Namespace, newNodes)
		if err != nil {
			return nil, fmt.Errorf("unable to save offloaded nodes: %s", err)
		}
		newWF.Status.OffloadNodeStatusVersion = offloadVersion
		newWF.Status.CompressedNodes = ""
		newWF.Status.Nodes = nil
	} else {
		newWF.Status.Nodes = newNodes
	}

	newWF.Status.StoredTemplates = make(map[string]wfv1.Template)
	for id, tmpl := range wf.Status.StoredTemplates {
		newWF.Status.StoredTemplates[id] = tmpl
	}

	err = packer.CompressWorkflowIfNeeded(newWF)
	if err != nil {
		return nil, fmt.Errorf("unable to compress workflow: %s", err)
	}

	return wfClient.Update(newWF)
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
func TerminateWorkflow(wfClient v1alpha1.WorkflowInterface, name string) error {
	patchObj := map[string]interface{}{
		"spec": map[string]interface{}{
			"shutdown": wfv1.ShutdownStrategyTerminate,
		},
	}
	var err error
	patch, err := json.Marshal(patchObj)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	_ = wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		_, err = wfClient.Patch(name, types.MergePatchType, patch)
		if err != nil {
			if !apierr.IsConflict(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
	return err
}

// StopWorkflow terminates a workflow by setting its spec.shutdown to ShutdownStrategyStop
// Or terminates a single resume step referenced by nodeFieldSelector
func StopWorkflow(wfClient v1alpha1.WorkflowInterface, name string, nodeFieldSelector string, message string) error {
	if len(nodeFieldSelector) > 0 {
		return updateWorkflowNodeByKey(wfClient, name, nodeFieldSelector, wfv1.NodeFailed, message)
	} else {
		patchObj := map[string]interface{}{
			"spec": map[string]interface{}{
				"shutdown": wfv1.ShutdownStrategyStop,
			},
		}
		var err error
		patch, err := json.Marshal(patchObj)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		for attempt := 0; attempt < 10; attempt++ {
			_, err = wfClient.Patch(name, types.MergePatchType, patch)
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
	response, err := http.Get(url)
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
			body, err = ioutil.ReadFile(filePathOrUrl)
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
	var wfPatch, tmplPatch, mergedPatch string
	var err error

	if wf.Spec.HasPodSpecPatch() {
		wfPatch, err = ConvertYAMLToJSON(wf.Spec.PodSpecPatch)
		if err != nil {
			return "", err
		}
	}
	if tmpl.HasPodSpecPatch() {
		tmplPatch, err = ConvertYAMLToJSON(tmpl.PodSpecPatch)
		if err != nil {
			return "", err
		}

		if wfPatch != "" {
			mergedByte, err := strategicpatch.StrategicMergePatch([]byte(wfPatch), []byte(tmplPatch), apiv1.PodSpec{})
			if err != nil {
				return "", err
			}
			mergedPatch = string(mergedByte)
		} else {
			mergedPatch = tmplPatch
		}
	} else {
		mergedPatch = wfPatch
	}
	return mergedPatch, nil
}

func ValidateJsonStr(jsonStr string, schema interface{}) bool {
	err := json.Unmarshal([]byte(jsonStr), &schema)
	return err == nil
}
