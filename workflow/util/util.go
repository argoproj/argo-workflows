package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/util/file"
	"github.com/argoproj/argo/util/retry"
	unstructutil "github.com/argoproj/argo/util/unstructured"
	"github.com/argoproj/argo/workflow/common"
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
		Resource: "workflows",
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
	OwnerReference *metav1.OwnerReference // useful if your custom controller creates argo workflow resources
}

// SubmitWorkflow validates and submit a single workflow and override some of the fields of the workflow
func SubmitWorkflow(wfIf v1alpha1.WorkflowInterface, wf *wfv1.Workflow, opts *SubmitOpts) (*wfv1.Workflow, error) {
	if opts == nil {
		opts = &SubmitOpts{}
	}
	if opts.Entrypoint != "" {
		wf.Spec.Entrypoint = opts.Entrypoint
	}
	if opts.ServiceAccount != "" {
		wf.Spec.ServiceAccountName = opts.ServiceAccount
	}
	if opts.InstanceID != "" {
		labels := wf.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[common.LabelKeyControllerInstanceID] = opts.InstanceID
		wf.SetLabels(labels)
	}
	if len(opts.Parameters) > 0 || opts.ParameterFile != "" {
		newParams := make([]wfv1.Parameter, 0)
		passedParams := make(map[string]bool)
		for _, paramStr := range opts.Parameters {
			parts := strings.SplitN(paramStr, "=", 2)
			if len(parts) == 1 {
				return nil, fmt.Errorf("Expected parameter of the form: NAME=VALUE. Received: %s", paramStr)
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
				response, err := http.Get(opts.ParameterFile)
				if err != nil {
					return nil, errors.InternalWrapError(err)
				}
				body, err = ioutil.ReadAll(response.Body)
				_ = response.Body.Close()
				if err != nil {
					return nil, errors.InternalWrapError(err)
				}
			} else {
				body, err = ioutil.ReadFile(opts.ParameterFile)
				if err != nil {
					return nil, errors.InternalWrapError(err)
				}
			}

			yamlParams := map[string]interface{}{}
			err = yaml.Unmarshal(body, &yamlParams)
			if err != nil {
				return nil, errors.InternalWrapError(err)
			}

			for k, v := range yamlParams {
				value := fmt.Sprintf("%v", v)
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

	err := validate.ValidateWorkflow(wf, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}
	return wfIf.Create(wf)
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
		if wf.Spec.Suspend == nil || *wf.Spec.Suspend != true {
			wf.Spec.Suspend = pointer.BoolPtr(true)
			wf, err = wfIf.Update(wf)
			if err != nil {
				if apierr.IsConflict(err) {
					return false, nil
				}
				return false, err
			}
		}
		return true, nil
	})
	if err != nil {
		return err
	}
	return nil
}

// ResumeWorkflow resumes a workflow by setting spec.suspend to nil and any suspended nodes to Successful.
// Retries conflict errors
func ResumeWorkflow(wfIf v1alpha1.WorkflowInterface, workflowName string) error {
	err := wait.ExponentialBackoff(retry.DefaultRetry, func() (bool, error) {
		wf, err := wfIf.Get(workflowName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		updated := false
		if wf.Spec.Suspend != nil && *wf.Spec.Suspend {
			wf.Spec.Suspend = nil
			updated = true
		}
		// To resume a workflow with a suspended node we simply mark the node as Successful
		for nodeID, node := range wf.Status.Nodes {
			if node.Type == wfv1.NodeTypeSuspend && node.Phase == wfv1.NodeRunning {
				node.Phase = wfv1.NodeSucceeded
				node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
				wf.Status.Nodes[nodeID] = node
				updated = true
			}
		}
		if updated {
			wf, err = wfIf.Update(wf)
			if err != nil {
				if apierr.IsConflict(err) {
					return false, nil
				}
				return false, err
			}
		}
		return true, nil
	})
	if err != nil {
		return err
	}
	return nil
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

	// carry over user labels and annotations from previous workflow.
	// skip any argoproj.io labels except for the controller instanceID label.
	for key, val := range wf.ObjectMeta.Labels {
		if strings.HasPrefix(key, workflow.FullName+"/") && key != common.LabelKeyControllerInstanceID {
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

	// Iterate the previous nodes. If it was successful Pod carry it forward
	replaceRegexp := regexp.MustCompile("^" + wf.ObjectMeta.Name)
	newWF.Status.Nodes = make(map[string]wfv1.NodeStatus)
	onExitNodeName := wf.ObjectMeta.Name + ".onExit"
	for _, node := range wf.Status.Nodes {
		switch node.Phase {
		case wfv1.NodeSucceeded, wfv1.NodeSkipped:
			if strings.HasPrefix(node.Name, onExitNodeName) {
				continue
			}
			originalID := node.ID
			node.Name = replaceRegexp.ReplaceAllString(node.Name, newWF.ObjectMeta.Name)
			node.ID = newWF.NodeID(node.Name)
			node.BoundaryID = convertNodeID(&newWF, replaceRegexp, node.BoundaryID, wf.Status.Nodes)
			node.StartedAt = metav1.Time{Time: time.Now().UTC()}
			node.FinishedAt = node.StartedAt
			newChildren := make([]string, len(node.Children))
			for i, childID := range node.Children {
				newChildren[i] = convertNodeID(&newWF, replaceRegexp, childID, wf.Status.Nodes)
			}
			node.Children = newChildren
			newOutboundNodes := make([]string, len(node.OutboundNodes))
			for i, outboundID := range node.OutboundNodes {
				newOutboundNodes[i] = convertNodeID(&newWF, replaceRegexp, outboundID, wf.Status.Nodes)
			}
			node.OutboundNodes = newOutboundNodes
			if node.Type == wfv1.NodeTypePod {
				node.Phase = wfv1.NodeSkipped
				node.Type = wfv1.NodeTypeSkipped
				node.Message = fmt.Sprintf("original pod: %s", originalID)
			}
			newWF.Status.Nodes[node.ID] = node
		case wfv1.NodeError, wfv1.NodeFailed, wfv1.NodeRunning, wfv1.NodePending:
			// do not add this status to the node. pretend as if this node never existed.
			// NOTE: NodeRunning shouldn't really happen except in weird scenarios where controller
			// mismanages state (e.g. panic when operating on a workflow)
		default:
			return nil, errors.InternalErrorf("Workflow cannot be resubmitted with node %s in %s phase", node, node.Phase)
		}
	}
	return &newWF, nil
}

// convertNodeID converts an old nodeID to a new nodeID
func convertNodeID(newWf *wfv1.Workflow, regex *regexp.Regexp, oldNodeID string, oldNodes map[string]wfv1.NodeStatus) string {
	node := oldNodes[oldNodeID]
	newNodeName := regex.ReplaceAllString(node.Name, newWf.ObjectMeta.Name)
	return newWf.NodeID(newNodeName)
}

// RetryWorkflow updates a workflow, deleting all failed steps as well as the onExit node (and children)
func RetryWorkflow(kubeClient kubernetes.Interface, wfClient v1alpha1.WorkflowInterface, wf *wfv1.Workflow) (*wfv1.Workflow, error) {
	switch wf.Status.Phase {
	case wfv1.NodeFailed, wfv1.NodeError:
	default:
		return nil, errors.Errorf(errors.CodeBadRequest, "workflow must be Failed/Error to retry")
	}
	newWF := wf.DeepCopy()
	podIf := kubeClient.CoreV1().Pods(wf.ObjectMeta.Namespace)

	// Delete/reset fields which indicate workflow completed
	delete(newWF.Labels, common.LabelKeyCompleted)
	newWF.ObjectMeta.Labels[common.LabelKeyPhase] = string(wfv1.NodeRunning)
	newWF.Status.Phase = wfv1.NodeRunning
	newWF.Status.Message = ""
	newWF.Status.FinishedAt = metav1.Time{}
	if newWF.Spec.ActiveDeadlineSeconds != nil && *newWF.Spec.ActiveDeadlineSeconds == 0 {
		// if it was terminated, unset the deadline
		newWF.Spec.ActiveDeadlineSeconds = nil
	}

	// Iterate the previous nodes. If it was successful Pod carry it forward
	newWF.Status.Nodes = make(map[string]wfv1.NodeStatus)
	onExitNodeName := wf.ObjectMeta.Name + ".onExit"
	for _, node := range wf.Status.Nodes {
		switch node.Phase {
		case wfv1.NodeSucceeded, wfv1.NodeSkipped:
			if !strings.HasPrefix(node.Name, onExitNodeName) {
				newWF.Status.Nodes[node.ID] = node
				continue
			}
		case wfv1.NodeError, wfv1.NodeFailed:
			// do not add this status to the node. pretend as if this node never existed.
		default:
			// Do not allow retry of workflows with pods in Running/Pending phase
			return nil, errors.InternalErrorf("Workflow cannot be retried with node %s in %s phase", node, node.Phase)
		}
		if node.Type == wfv1.NodeTypePod {
			log.Infof("Deleting pod: %s", node.ID)
			err := podIf.Delete(node.ID, &metav1.DeleteOptions{})
			if err != nil && !apierr.IsNotFound(err) {
				return nil, errors.InternalWrapError(err)
			}
		}
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
		if node.Type == wfv1.NodeTypeSuspend && node.Phase == wfv1.NodeRunning {
			return true
		}
	}
	return false
}

func IsWorkflowTerminated(wf *wfv1.Workflow) bool {
	if wf.Spec.ActiveDeadlineSeconds != nil && *wf.Spec.ActiveDeadlineSeconds == 0 {
		return true
	}
	return false
}

// TerminateWorkflow terminates a workflow by setting its activeDeadlineSeconds to 0
func TerminateWorkflow(wfClient v1alpha1.WorkflowInterface, name string) error {
	patchObj := map[string]interface{}{
		"spec": map[string]interface{}{
			"activeDeadlineSeconds": 0,
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

// DecompressWorkflow decompresses the compressed status of a workflow (if compressed)
func DecompressWorkflow(wf *wfv1.Workflow) error {
	if wf.Status.CompressedNodes != "" {
		nodeContent, err := file.DecodeDecompressString(wf.Status.CompressedNodes)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		err = json.Unmarshal([]byte(nodeContent), &wf.Status.Nodes)
		if err != nil {
			return err
		}
		wf.Status.CompressedNodes = ""
	}
	return nil
}
