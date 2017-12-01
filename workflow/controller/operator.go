package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	workflowclient "github.com/argoproj/argo/workflow/client"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasttemplate"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// wfOperationCtx is the context for evaluation and operation of a single workflow
type wfOperationCtx struct {
	// wf is the workflow object
	wf *wfv1.Workflow
	// updated indicates whether or not the workflow object itself was updated
	// and needs to be persisted back to kubernetes
	updated bool
	// log is an logrus logging context to corrolate logs with a workflow
	log *log.Entry
	// controller reference to workflow controller
	controller *WorkflowController
	// NOTE: eventually we may need to store additional metadata state to
	// understand how to proceed in workflows with more complex control flows.
	// (e.g. workflow failed in step 1 of 3 but has finalizer steps)
}

// wfScope contains the current scope of variables available when iterating steps in a workflow
type wfScope struct {
	tmpl  *wfv1.Template
	scope map[string]interface{}
}

// operateWorkflow is the operator logic of a workflow
// It evaluates the current state of the workflow and decides how to proceed down the execution path
func (wfc *WorkflowController) operateWorkflow(wf *wfv1.Workflow) {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	woc := wfOperationCtx{
		wf:      wf.DeepCopyObject().(*wfv1.Workflow),
		updated: false,
		log: log.WithFields(log.Fields{
			"workflow":  wf.ObjectMeta.Name,
			"namespace": wf.ObjectMeta.Namespace,
		}),
		controller: wfc,
	}
	defer func() {
		if woc.updated {
			wfClient := workflowclient.NewWorkflowClient(wfc.restClient, wf.ObjectMeta.Namespace)
			_, err := wfClient.UpdateWorkflow(woc.wf)
			if err != nil {
				woc.log.Errorf("Error updating %s status: %v", woc.wf.ObjectMeta.SelfLink, err)
			} else {
				woc.log.Infof("Workflow %s updated", woc.wf.ObjectMeta.SelfLink)
			}
		}
	}()

	// Perform one-time workflow validation
	if woc.wf.Status.Phase == "" {
		woc.markWorkflowRunning()
		err := common.ValidateWorkflow(woc.wf)
		if err != nil {
			woc.markWorkflowFailed(fmt.Sprintf("invalid spec: %s", err.Error()))
			return
		}
	}

	err := woc.createPVCs()
	if err != nil {
		woc.log.Errorf("%s error: %+v", wf.ObjectMeta.Name, err)
		woc.markWorkflowError(err, true)
		return
	}

	err = woc.executeTemplate(wf.Spec.Entrypoint, wf.Spec.Arguments, wf.ObjectMeta.Name)
	if err != nil {
		woc.log.Errorf("%s error: %+v", wf.ObjectMeta.Name, err)
	}
	node := woc.wf.Status.Nodes[woc.wf.NodeID(wf.ObjectMeta.Name)]
	if !node.Completed() {
		return
	}

	err = woc.deletePVCs()
	if err != nil {
		woc.log.Errorf("%s error: %+v", wf.ObjectMeta.Name, err)
		// Mark the workflow with an error message and return, but intentionally do not
		// markCompletion so that we can retry PVC deletion (TODO: requires resync to be set on the informer)
		// This error phase may be cleared if a subsequent delete attempt is successful.
		woc.markWorkflowError(err, false)
		return
	}

	// TODO: workflow finalizer logic goes here

	// If we get here, the workflow completed, all PVCs were deleted successfully,
	// and finalizers were executed (finalizer feature yet to be implemented).
	// We now need to infer the workflow phase from the node phase.
	switch node.Phase {
	case wfv1.NodeSucceeded, wfv1.NodeSkipped:
		woc.markWorkflowSuccess()
	case wfv1.NodeFailed:
		woc.markWorkflowFailed(node.Message)
	case wfv1.NodeError:
		woc.markWorkflowPhase(wfv1.NodeError, true, node.Message)
	default:
		// NOTE: we should never make it here because if the the node was 'Running'
		// we should have returned earlier.
		err = errors.InternalErrorf("Unexpected node phase %s: %+v", wf.ObjectMeta.Name, err)
		woc.markWorkflowError(err, true)
	}
}

func (woc *wfOperationCtx) createPVCs() error {
	if woc.wf.Status.Phase != wfv1.NodeRunning {
		// Only attempt to create PVCs if workflow transitioned to Running state
		// (e.g. passed validation, or didn't already complete)
		return nil
	}
	if len(woc.wf.Spec.VolumeClaimTemplates) == len(woc.wf.Status.PersistentVolumeClaims) {
		// If we have already created the PVCs, then there is nothing to do.
		// This will also handle the case where workflow has no volumeClaimTemplates.
		return nil
	}
	if len(woc.wf.Status.PersistentVolumeClaims) == 0 {
		woc.wf.Status.PersistentVolumeClaims = make([]apiv1.Volume, len(woc.wf.Spec.VolumeClaimTemplates))
	}
	pvcClient := woc.controller.clientset.CoreV1().PersistentVolumeClaims(woc.wf.ObjectMeta.Namespace)
	t := true
	for i, pvcTmpl := range woc.wf.Spec.VolumeClaimTemplates {
		if pvcTmpl.ObjectMeta.Name == "" {
			return errors.Errorf(errors.CodeBadRequest, "volumeClaimTemplates[%d].metadata.name is required", i)
		}
		pvcTmpl = *pvcTmpl.DeepCopy()
		// PVC name will be <workflowname>-<volumeclaimtemplatename>
		refName := pvcTmpl.ObjectMeta.Name
		pvcName := fmt.Sprintf("%s-%s", woc.wf.ObjectMeta.Name, pvcTmpl.ObjectMeta.Name)
		woc.log.Infof("Creating pvc %s", pvcName)
		pvcTmpl.ObjectMeta.Name = pvcName
		pvcTmpl.OwnerReferences = []metav1.OwnerReference{
			metav1.OwnerReference{
				APIVersion:         wfv1.CRDFullName,
				Kind:               wfv1.CRDKind,
				Name:               woc.wf.ObjectMeta.Name,
				UID:                woc.wf.ObjectMeta.UID,
				BlockOwnerDeletion: &t,
			},
		}
		pvc, err := pvcClient.Create(&pvcTmpl)
		if err != nil {
			woc.markNodeError(woc.wf.ObjectMeta.Name, err)
			return err
		}
		vol := apiv1.Volume{
			Name: refName,
			VolumeSource: apiv1.VolumeSource{
				PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.ObjectMeta.Name,
				},
			},
		}
		woc.wf.Status.PersistentVolumeClaims[i] = vol
		woc.updated = true
	}
	return nil
}

func (woc *wfOperationCtx) deletePVCs() error {
	totalPVCs := len(woc.wf.Status.PersistentVolumeClaims)
	if totalPVCs == 0 {
		// PVC list already empty. nothing to do
		return nil
	}
	pvcClient := woc.controller.clientset.CoreV1().PersistentVolumeClaims(woc.wf.ObjectMeta.Namespace)
	newPVClist := make([]apiv1.Volume, 0)
	// Attempt to delete all PVCs. Record first error encountered
	var firstErr error
	for _, pvc := range woc.wf.Status.PersistentVolumeClaims {
		woc.log.Infof("Deleting PVC %s", pvc.PersistentVolumeClaim.ClaimName)
		err := pvcClient.Delete(pvc.PersistentVolumeClaim.ClaimName, nil)
		if err != nil {
			if !apierr.IsNotFound(err) {
				woc.log.Errorf("Failed to delete pvc %s: %v", pvc.PersistentVolumeClaim.ClaimName, err)
				newPVClist = append(newPVClist, pvc)
				if firstErr == nil {
					firstErr = err
				}
			}
		}
	}
	if len(newPVClist) != totalPVCs {
		// we were successful in deleting one ore more PVCs
		woc.log.Infof("Deleted %d/%d PVCs", totalPVCs-len(newPVClist), totalPVCs)
		woc.wf.Status.PersistentVolumeClaims = newPVClist
		woc.updated = true
	}
	return firstErr
}

func (woc *wfOperationCtx) executeTemplate(templateName string, args wfv1.Arguments, nodeName string) error {
	woc.log.Infof("Evaluating node %s: template: %s", nodeName, templateName)
	nodeID := woc.wf.NodeID(nodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if ok && node.Completed() {
		woc.log.Infof("Node %s already completed", nodeName)
		return nil
	}
	tmpl := woc.wf.GetTemplate(templateName)
	if tmpl == nil {
		err := errors.Errorf(errors.CodeBadRequest, "Node %v error: template '%s' undefined", node, templateName)
		woc.markNodeError(nodeName, err)
		return err
	}

	tmpl, err := processArgs(tmpl, args)
	if err != nil {
		woc.markNodeError(nodeName, err)
		return err
	}

	if tmpl.Container != nil {
		if ok {
			// There's already a node entry for the container. This means the container was already
			// scheduled (or had a create pod error). Nothing to more to do with this node.
			return nil
		}
		// We have not yet created the pod
		return woc.executeContainer(nodeName, tmpl)

	} else if len(tmpl.Steps) > 0 {
		if !ok {
			node = *woc.markNodePhase(nodeName, wfv1.NodeRunning)
			woc.log.Infof("Initialized workflow node %v", node)
		}
		err = woc.executeSteps(nodeName, tmpl)
		if woc.wf.Status.Nodes[nodeID].Completed() {
			woc.killDeamonedChildren(nodeID)
		}
		return err

	} else if tmpl.Script != nil {
		return woc.executeScript(nodeName, tmpl)
	}
	err = errors.Errorf("Template '%s' missing specification", tmpl.Name)
	woc.markNodeError(nodeName, err)
	return err
}

// processArgs sets in the inputs, the values either passed via arguments, or the hardwired values
// It also substitutes parameters in the template from the arguments
func processArgs(tmpl *wfv1.Template, args wfv1.Arguments) (*wfv1.Template, error) {
	// For each input parameter:
	// 1) check if was supplied as argument. if so use the supplied value from arg
	// 2) if not, use default value.
	// 3) if no default value, it is an error
	tmpl = tmpl.DeepCopy()
	for i, inParam := range tmpl.Inputs.Parameters {
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
			return nil, errors.Errorf(errors.CodeBadRequest, "inputs.parameters.%s was not satisfied", inParam.Name)
		}
		tmpl.Inputs.Parameters[i] = inParam
	}
	tmpl, err := substituteParams(tmpl)
	if err != nil {
		return nil, err
	}

	newInputArtifacts := make([]wfv1.Artifact, len(tmpl.Inputs.Artifacts))
	for i, inArt := range tmpl.Inputs.Artifacts {
		// if artifact has hard-wired location, we prefer that
		if inArt.HasLocation() {
			newInputArtifacts[i] = inArt
			continue
		}
		// artifact must be supplied
		argArt := args.GetArtifactByName(inArt.Name)
		if argArt == nil {
			return nil, errors.Errorf(errors.CodeBadRequest, "arguments.artifacts.%s was not supplied", inArt.Name)
		}
		if !argArt.HasLocation() {
			return nil, errors.Errorf(errors.CodeBadRequest, "arguments.artifacts.%s missing location information", inArt.Name)
		}
		argArt.Path = inArt.Path
		argArt.Mode = inArt.Mode
		newInputArtifacts[i] = *argArt
	}
	tmpl.Inputs.Artifacts = newInputArtifacts
	return tmpl, nil
}

// markWorkflowPhase is a convenience method to set the phase of the workflow with optional message
// optionally marks the workflow completed, which sets the finishedAt timestamp and completed label
func (woc *wfOperationCtx) markWorkflowPhase(phase wfv1.NodePhase, markCompleted bool, message ...string) {
	if woc.wf.Status.Phase != phase {
		woc.log.Infof("Updated phase %s -> %s", woc.wf.Status.Phase, phase)
		woc.updated = true
		woc.wf.Status.Phase = phase
		if woc.wf.ObjectMeta.Labels == nil {
			woc.wf.ObjectMeta.Labels = make(map[string]string)
		}
		woc.wf.ObjectMeta.Labels[common.LabelKeyPhase] = string(phase)
	}
	if woc.wf.Status.StartedAt.IsZero() {
		woc.updated = true
		woc.wf.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	}
	if len(message) > 0 && woc.wf.Status.Message != message[0] {
		woc.log.Infof("Updated message %s -> %s", woc.wf.Status.Message, message[0])
		woc.updated = true
		woc.wf.Status.Message = message[0]
	}

	switch phase {
	case wfv1.NodeSucceeded, wfv1.NodeFailed, wfv1.NodeError:
		if markCompleted {
			woc.log.Infof("Marking workflow completed")
			woc.wf.Status.FinishedAt = metav1.Time{Time: time.Now().UTC()}
			if woc.wf.ObjectMeta.Labels == nil {
				woc.wf.ObjectMeta.Labels = make(map[string]string)
			}
			woc.wf.ObjectMeta.Labels[common.LabelKeyCompleted] = "true"
			woc.updated = true
		}
	}
}

func (woc *wfOperationCtx) markWorkflowRunning() {
	woc.markWorkflowPhase(wfv1.NodeRunning, false)
}

func (woc *wfOperationCtx) markWorkflowSuccess() {
	woc.markWorkflowPhase(wfv1.NodeSucceeded, true)
}

func (woc *wfOperationCtx) markWorkflowFailed(message string) {
	woc.markWorkflowPhase(wfv1.NodeFailed, true, message)
}

func (woc *wfOperationCtx) markWorkflowError(err error, markCompleted bool) {
	woc.markWorkflowPhase(wfv1.NodeError, markCompleted, err.Error())
}

// markNodePhase marks a node with the given phase, creating the node if necessary and handles timestamps
func (woc *wfOperationCtx) markNodePhase(nodeName string, phase wfv1.NodePhase, message ...string) *wfv1.NodeStatus {
	if woc.wf.Status.Nodes == nil {
		woc.wf.Status.Nodes = make(map[string]wfv1.NodeStatus)
	}
	nodeID := woc.wf.NodeID(nodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if !ok {
		node = wfv1.NodeStatus{
			ID:        nodeID,
			Name:      nodeName,
			Phase:     phase,
			StartedAt: metav1.Time{Time: time.Now().UTC()},
		}
	} else {
		node.Phase = phase
	}
	if len(message) > 0 {
		node.Message = message[0]
	}
	if node.Completed() && node.FinishedAt.IsZero() {
		node.FinishedAt = metav1.Time{Time: time.Now().UTC()}
	}
	woc.wf.Status.Nodes[nodeID] = node
	woc.updated = true
	return &node
}

// markNodeError is a convenience method to mark a node with an error and set the message from the error
func (woc *wfOperationCtx) markNodeError(nodeName string, err error) *wfv1.NodeStatus {
	return woc.markNodePhase(nodeName, wfv1.NodeError, err.Error())
}

func (woc *wfOperationCtx) executeContainer(nodeName string, tmpl *wfv1.Template) error {
	err := woc.createWorkflowPod(nodeName, tmpl)
	if err != nil {
		// TODO: may need to query pod status if we hit already exists error
		woc.markNodeError(nodeName, err)
		return err
	}
	node := woc.markNodePhase(nodeName, wfv1.NodeRunning)
	woc.log.Infof("Initialized container node %v", node)
	return nil
}

func (woc *wfOperationCtx) executeSteps(nodeName string, tmpl *wfv1.Template) error {
	scope := wfScope{
		tmpl:  tmpl,
		scope: make(map[string]interface{}),
	}
	for i, stepGroup := range tmpl.Steps {
		sgNodeName := fmt.Sprintf("%s[%d]", nodeName, i)
		woc.addChildNode(nodeName, sgNodeName)
		err := woc.executeStepGroup(stepGroup, sgNodeName, &scope)
		if err != nil {
			woc.markNodeError(nodeName, err)
			return err
		}
		sgNodeID := woc.wf.NodeID(sgNodeName)
		if !woc.wf.Status.Nodes[sgNodeID].Completed() {
			woc.log.Infof("Workflow step group node %v not yet completed", woc.wf.Status.Nodes[sgNodeID])
			return nil
		}

		if !woc.wf.Status.Nodes[sgNodeID].Successful() {
			failMessage := fmt.Sprintf("step group %s was unsuccessful", sgNodeName)
			woc.log.Info(failMessage)
			woc.markNodePhase(nodeName, wfv1.NodeFailed, failMessage)
			return nil
		}

		// HACK: need better way to add children to scope
		for _, step := range stepGroup {
			childNodeName := fmt.Sprintf("%s.%s", sgNodeName, step.Name)
			childNodeID := woc.wf.NodeID(childNodeName)
			childNode, ok := woc.wf.Status.Nodes[childNodeID]
			if !ok {
				// This can happen if there was `withItem` expansion
				// it is okay to ignore this because these expanded steps
				// are not easily referenceable by user.
				continue
			}
			if childNode.PodIP != "" {
				key := fmt.Sprintf("steps.%s.ip", step.Name)
				scope.addParamToScope(key, childNode.PodIP)
			}
			if childNode.Outputs != nil {
				if childNode.Outputs.Result != nil {
					key := fmt.Sprintf("steps.%s.outputs.result", step.Name)
					scope.addParamToScope(key, *childNode.Outputs.Result)
				}
				for _, outParam := range childNode.Outputs.Parameters {
					key := fmt.Sprintf("steps.%s.outputs.parameters.%s", step.Name, outParam.Name)
					scope.addParamToScope(key, *outParam.Value)
				}
				for _, outArt := range childNode.Outputs.Artifacts {
					key := fmt.Sprintf("steps.%s.outputs.artifacts.%s", step.Name, outArt.Name)
					scope.addArtifactToScope(key, outArt)
				}
			}
		}
	}
	woc.markNodePhase(nodeName, wfv1.NodeSucceeded)
	return nil
}

// executeStepGroup examines a map of parallel steps and executes them in parallel.
// Handles referencing of variables in scope, expands `withItem` clauses, and evaluates `when` expressions
func (woc *wfOperationCtx) executeStepGroup(stepGroup []wfv1.WorkflowStep, sgNodeName string, scope *wfScope) error {
	nodeID := woc.wf.NodeID(sgNodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if ok && node.Completed() {
		woc.log.Infof("Step group node %v already marked completed", node)
		return nil
	}
	if !ok {
		node = *woc.markNodePhase(sgNodeName, wfv1.NodeRunning)
		woc.log.Infof("Initializing step group node %v", node)
	}

	// First, resolve any references to outputs from previous steps, and perform substitution
	stepGroup, err := woc.resolveReferences(stepGroup, scope)
	if err != nil {
		woc.markNodeError(sgNodeName, err)
		return err
	}

	// Next, expand the step's withItems (if any)
	stepGroup, err = woc.expandStepGroup(stepGroup)
	if err != nil {
		woc.markNodeError(sgNodeName, err)
		return err
	}

	// Kick off all parallel steps in the group
	for _, step := range stepGroup {
		childNodeName := fmt.Sprintf("%s.%s", sgNodeName, step.Name)
		woc.addChildNode(sgNodeName, childNodeName)

		// Check the step's when clause to decide if it should execute
		proceed, err := shouldExecute(step.When)
		if err != nil {
			woc.markNodeError(childNodeName, err)
			woc.markNodeError(sgNodeName, err)
			return err
		}
		if !proceed {
			skipReason := fmt.Sprintf("when '%s' evaluated false", step.When)
			woc.log.Infof("Skipping %s: %s", childNodeName, skipReason)
			woc.markNodePhase(childNodeName, wfv1.NodeSkipped, skipReason)
			continue
		}
		err = woc.executeTemplate(step.Template, step.Arguments, childNodeName)
		if err != nil {
			woc.markNodeError(childNodeName, err)
			woc.markNodeError(sgNodeName, err)
			return err
		}
	}

	node = woc.wf.Status.Nodes[nodeID]
	// Return if not all children completed
	for _, childNodeID := range node.Children {
		if !woc.wf.Status.Nodes[childNodeID].Completed() {
			return nil
		}
	}
	// All children completed. Determine step group status as a whole
	for _, childNodeID := range node.Children {
		childNode := woc.wf.Status.Nodes[childNodeID]
		if !childNode.Successful() {
			failMessage := fmt.Sprintf("child '%s' failed", childNodeID)
			woc.markNodePhase(sgNodeName, wfv1.NodeFailed, failMessage)
			woc.log.Infof("Step group node %s deemed failed: %s", childNode, failMessage)
			return nil
		}
	}
	woc.markNodePhase(node.Name, wfv1.NodeSucceeded)
	woc.log.Infof("Step group node %v successful", woc.wf.Status.Nodes[nodeID])
	return nil
}

var whenExpression = regexp.MustCompile("^(.*)(==|!=)(.*)$")

// shouldExecute evaluates a already substituted when expression to decide whether or not a step should execute
func shouldExecute(when string) (bool, error) {
	if when == "" {
		return true, nil
	}
	parts := whenExpression.FindStringSubmatch(when)
	if len(parts) == 0 {
		return false, errors.Errorf(errors.CodeBadRequest, "Invalid 'when' expression: %s", when)
	}
	var1 := strings.TrimSpace(parts[1])
	operator := parts[2]
	var2 := strings.TrimSpace(parts[3])
	switch operator {
	case "==":
		return var1 == var2, nil
	case "!=":
		return var1 != var2, nil
	default:
		return false, errors.Errorf(errors.CodeBadRequest, "Unknown operator: %s", operator)
	}
}

// resolveReferences replaces any references to outputs of previous steps, or artifacts in the inputs
// NOTE: by now, input parameters should have been substituted throughout the template, so we only
// are concerned with:
// 1) dereferencing output.parameters from previous steps
// 2) dereferencing output.result from previous steps
// 2) dereferencing artifacts from previous steps
// 3) dereferencing artifacts from inputs
func (woc *wfOperationCtx) resolveReferences(stepGroup []wfv1.WorkflowStep, scope *wfScope) ([]wfv1.WorkflowStep, error) {
	newStepGroup := make([]wfv1.WorkflowStep, len(stepGroup))

	for i, step := range stepGroup {
		// Step 1: replace all parameter scope references in the step
		// TODO: improve this
		stepBytes, err := json.Marshal(step)
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}
		replaceMap := make(map[string]string)
		for key, val := range scope.scope {
			valStr, ok := val.(string)
			if ok {
				replaceMap[key] = valStr
			}
		}
		fstTmpl := fasttemplate.New(string(stepBytes), "{{", "}}")
		newStepStr, err := replace(fstTmpl, replaceMap, true)
		if err != nil {
			return nil, err
		}
		var newStep wfv1.WorkflowStep
		err = json.Unmarshal([]byte(newStepStr), &newStep)
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}

		// Step 2: replace all artifact references
		for j, art := range newStep.Arguments.Artifacts {
			if art.From == "" {
				continue
			}
			resolvedArt, err := scope.resolveArtifact(art.From)
			if err != nil {
				return nil, err
			}
			resolvedArt.Name = art.Name
			newStep.Arguments.Artifacts[j] = *resolvedArt
		}

		newStepGroup[i] = newStep
	}
	return newStepGroup, nil
}

// expandStepGroup looks at each step in a collection of parallel steps, and expands all steps using withItems/withParam
func (woc *wfOperationCtx) expandStepGroup(stepGroup []wfv1.WorkflowStep) ([]wfv1.WorkflowStep, error) {
	newStepGroup := make([]wfv1.WorkflowStep, 0)
	for _, step := range stepGroup {
		if len(step.WithItems) == 0 && step.WithParam == "" {
			newStepGroup = append(newStepGroup, step)
			continue
		}
		expandedStep, err := woc.expandStep(step)
		if err != nil {
			return nil, err
		}
		for _, newStep := range expandedStep {
			newStepGroup = append(newStepGroup, newStep)
		}
	}
	return newStepGroup, nil
}

// expandStep expands a step containing withItems or withParams into multiple parallel steps
func (woc *wfOperationCtx) expandStep(step wfv1.WorkflowStep) ([]wfv1.WorkflowStep, error) {
	stepBytes, err := json.Marshal(step)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	fstTmpl := fasttemplate.New(string(stepBytes), "{{", "}}")
	expandedStep := make([]wfv1.WorkflowStep, 0)
	var items []wfv1.Item
	if len(step.WithItems) > 0 {
		items = step.WithItems
	} else if step.WithParam != "" {
		err = json.Unmarshal([]byte(step.WithParam), &items)
		if err != nil {
			return nil, errors.Errorf(errors.CodeBadRequest, "withParam value not be parsed as a JSON list: %s", step.WithParam)
		}
	} else {
		// this should have been prevented in expandStepGroup()
		return nil, errors.InternalError("expandStep() was called with withItems and withParam empty")
	}

	for i, item := range items {
		replaceMap := make(map[string]string)
		var newStepName string
		switch val := item.(type) {
		case string, int32, int64, float32, float64:
			replaceMap["item"] = fmt.Sprintf("%v", val)
			newStepName = fmt.Sprintf("%s(%v)", step.Name, val)
		case map[string]interface{}:
			// Handle the case when withItems is a list of maps.
			// vals holds stringified versions of the map items which are incorporated as part of the step name.
			// For example if the item is: {"name": "jesse","group":"developer"}
			// the vals would be: ["name:jesse", "group:developer"]
			// This would eventually be part of the step name (group:developer,name:jesse)
			vals := make([]string, 0)
			for itemKey, itemValIf := range val {
				switch itemVal := itemValIf.(type) {
				case string, int32, int64, float32, float64:
					replaceMap[fmt.Sprintf("item.%s", itemKey)] = fmt.Sprintf("%v", itemVal)
					vals = append(vals, fmt.Sprintf("%s:%s", itemKey, itemVal))
				default:
					return nil, errors.Errorf(errors.CodeBadRequest, "withItems[%d][%s] expected string or number. received: %s", i, itemKey, itemVal)
				}
			}
			// sort the values so that the name is deterministic
			sort.Strings(vals)
			newStepName = fmt.Sprintf("%s(%v)", step.Name, strings.Join(vals, ","))
		default:
			return nil, errors.Errorf(errors.CodeBadRequest, "withItems[%d] expected string, number, or map. received: %s", i, val)
		}
		newStepStr, err := replace(fstTmpl, replaceMap, false)
		if err != nil {
			return nil, err
		}
		var newStep wfv1.WorkflowStep
		err = json.Unmarshal([]byte(newStepStr), &newStep)
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}
		newStep.Name = newStepName
		expandedStep = append(expandedStep, newStep)
	}
	return expandedStep, nil
}

func (woc *wfOperationCtx) executeScript(nodeName string, tmpl *wfv1.Template) error {
	err := woc.createWorkflowPod(nodeName, tmpl)
	if err != nil {
		woc.markNodeError(nodeName, err)
		return err
	}
	node := woc.markNodePhase(nodeName, wfv1.NodeRunning)
	woc.log.Infof("Initialized container node %v", node)
	return nil
}

// substituteParams returns a new copy of the template with all input parameters substituted
func substituteParams(tmpl *wfv1.Template) (*wfv1.Template, error) {
	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	replaceMap := make(map[string]string)
	for _, inParam := range tmpl.Inputs.Parameters {
		if inParam.Value == nil {
			return nil, errors.InternalErrorf("inputs.parameters.%s had no value", inParam.Name)
		}
		replaceMap["inputs.parameters."+inParam.Name] = *inParam.Value
	}
	fstTmpl := fasttemplate.New(string(tmplBytes), "{{", "}}")
	s, err := replace(fstTmpl, replaceMap, true)
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

func (wfs *wfScope) addParamToScope(key, val string) {
	wfs.scope[key] = val
}

func (wfs *wfScope) addArtifactToScope(key string, artifact wfv1.Artifact) {
	wfs.scope[key] = artifact
}

func (wfs *wfScope) resolveVar(v string) (interface{}, error) {
	v = strings.TrimPrefix(v, "{{")
	v = strings.TrimSuffix(v, "}}")
	if strings.HasPrefix(v, "steps.") {
		val, ok := wfs.scope[v]
		if !ok {
			return nil, errors.Errorf(errors.CodeBadRequest, "Unable to resolve: {{%s}}", v)
		}
		return val, nil
	}
	parts := strings.Split(v, ".")
	// HACK (assuming it is an input artifact)
	art := wfs.tmpl.Inputs.GetArtifactByName(parts[2])
	if art != nil {
		return *art, nil
	}
	return nil, errors.Errorf(errors.CodeBadRequest, "Unable to resolve input artifact: {{%s}}", v)
}

func (wfs *wfScope) resolveParameter(v string) (string, error) {
	val, err := wfs.resolveVar(v)
	if err != nil {
		return "", err
	}
	valStr, ok := val.(string)
	if !ok {
		return "", errors.Errorf(errors.CodeBadRequest, "Variable {{%s}} is not a string", v)
	}
	return valStr, nil
}

func (wfs *wfScope) resolveArtifact(v string) (*wfv1.Artifact, error) {
	val, err := wfs.resolveVar(v)
	if err != nil {
		return nil, err
	}
	valArt, ok := val.(wfv1.Artifact)
	if !ok {
		return nil, errors.Errorf(errors.CodeBadRequest, "Variable {{%s}} is not an artifact", v)
	}
	return &valArt, nil
}

// replace executes basic string substitution of a template with replacement values.
// allowUnresolved indicates whether or not it is acceptable to have unresolved variables
// remaining in the substituted template.
func replace(fstTmpl *fasttemplate.Template, replaceMap map[string]string, allowUnresolved bool) (string, error) {
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

// addChildNode adds a nodeID as a child to a parent
func (woc *wfOperationCtx) addChildNode(parent string, child string) {
	parentID := woc.wf.NodeID(parent)
	childID := woc.wf.NodeID(child)
	node, ok := woc.wf.Status.Nodes[parentID]
	if !ok {
		panic(fmt.Sprintf("parent node %s not initialized", parent))
	}
	if node.Children == nil {
		node.Children = make([]string, 0)
	}
	for _, nodeID := range node.Children {
		if childID == nodeID {
			// already exists
			return
		}
	}
	node.Children = append(node.Children, childID)
	woc.wf.Status.Nodes[parentID] = node
	woc.updated = true
}

// killDeamonedChildren kill any granchildren of a step template node, which have been daemoned.
// We only need to check grandchildren instead of children becuase the direct children of a step
// template are actually stepGroups, which are nodes that cannot represent actual containers.
// Returns the first error that occurs (if any)
func (woc *wfOperationCtx) killDeamonedChildren(nodeID string) error {
	woc.log.Infof("Checking deamon children of %s", nodeID)
	var firstErr error
	for _, childNodeID := range woc.wf.Status.Nodes[nodeID].Children {
		for _, grandChildID := range woc.wf.Status.Nodes[childNodeID].Children {
			gcNode := woc.wf.Status.Nodes[grandChildID]
			if gcNode.Daemoned == nil || !*gcNode.Daemoned {
				continue
			}
			err := common.KillPodContainer(woc.controller.restConfig, woc.wf.ObjectMeta.Namespace, gcNode.ID, common.MainContainerName)
			if err != nil {
				woc.log.Errorf("Failed to kill %s: %+v", gcNode, err)
				if firstErr == nil {
					firstErr = err
				}
			}
		}
	}
	return firstErr
}
