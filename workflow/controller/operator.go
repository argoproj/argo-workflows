package controller

import (
	"fmt"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// operateWorkflow is the operator logic of a workflow
// It evaluates the current state of the workflow and decides how to proceed down the execution path
func (wfc *WorkflowController) operateWorkflow(wf *wfv1.Workflow) {
	if wf.Completed() {
		return
	}
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	wfCopy := wf.DeepCopyObject().(*wfv1.Workflow)
	updated := false

	defer func() {
		if updated {
			_, err := wfc.WorkflowClient.UpdateWorkflow(wfCopy)
			if err != nil {
				fmt.Printf("ERROR updating status: %v\n", err)
			} else {
				fmt.Printf("UPDATED %s: %#v\n", wfCopy.ObjectMeta.Name, wfCopy.Status)
			}
		}
	}()
	if wfCopy.Status.Nodes == nil {
		wfCopy.Status.Nodes = make(map[string]wfv1.NodeStatus)
		updated = true
	}

	tmplUpdates, err := wfc.executeTemplate(wfCopy, wfCopy.Spec.Entrypoint, nil, wfCopy.ObjectMeta.Name)
	updated = updated || tmplUpdates
	if err != nil {
		fmt.Printf("%s error: %+v\n", wf.ObjectMeta.Name, err)
	}
}

func (wfc *WorkflowController) createWorkflowContainer(wf *wfv1.Workflow, nodeName string, tmpl *wfv1.Template, args *wfv1.Arguments) error {
	fmt.Printf("Creating Pod: %s\n", nodeName)
	initCtr := wfc.newExecContainer("init", false)
	sidekickCtr := wfc.newExecContainer("wait", false)
	mainCtr := tmpl.Container.DeepCopy()
	mainCtr.Name = "main"
	t := true
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: wf.NodeID(nodeName),
			Labels: map[string]string{
				"workflow":    wf.ObjectMeta.Name,
				"argo-wf-pod": "true",
			},
			Annotations: map[string]string{
				"nodeName": nodeName,
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion:         "argoproj.io/v1",
					Kind:               "Workflow",
					Name:               wf.ObjectMeta.Name,
					UID:                wf.ObjectMeta.UID,
					BlockOwnerDeletion: &t,
				},
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			InitContainers: []corev1.Container{
				*initCtr,
			},
			Containers: []corev1.Container{
				*sidekickCtr,
				*mainCtr,
			},
		},
	}
	created, err := wfc.podCl.Create(&pod)
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			// workflow pod names are deterministic. We can get here if
			// the controller crashes after creating the pod, but fails
			// to store the update to etc, and controller retries creation
			fmt.Printf("pod %s already exists\n", nodeName)
			return nil
		}
		fmt.Printf("Failed to create pod %s: %v\n", nodeName, err)
		return errors.InternalWrapError(err)
	}
	fmt.Printf("Created pod: %v\n", created)
	return nil
}

func (wfc *WorkflowController) newExecContainer(name string, privileged bool) *corev1.Container {
	exec := corev1.Container{
		Name:    name,
		Image:   wfc.ArgoExecImage,
		Command: []string{"sh", "-c"},
		Args:    []string{"echo sleeping; sleep 60"},
		//EnvFrom []EnvFromSource `json:"envFrom,omitempty" protobuf:"bytes,19,rep,name=envFrom"`
		//Env []EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,7,rep,name=env"`
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("0.5"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("0.1"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
		},
		// VolumeMounts: []corev1.VolumeMount{
		// 	corev1.VolumeMount{
		// 		// This must match the Name of a Volume.
		// 		Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
		// 		// Mounted read-only if true, read-write otherwise (false or unspecified).
		// 		// Defaults to false.
		// 		// +optional
		// 		ReadOnly bool `json:"readOnly,omitempty" protobuf:"varint,2,opt,name=readOnly"`
		// 		// Path within the container at which the volume should be mounted.  Must
		// 		// not contain ':'.
		// 		MountPath string `json:"mountPath" protobuf:"bytes,3,opt,name=mountPath"`
		// 		// Path within the volume from which the container's volume should be mounted.
		// 		// Defaults to "" (volume's root).
		// 		// +optional
		// 		SubPath string `json:"subPath,omitempty" protobuf:"bytes,4,opt,name=subPath"`
		// 	}
		// },
		// Security options the pod should run with.
		// More info: https://kubernetes.io/docs/concepts/policy/security-context/
		// More info: https://git.k8s.io/community/contributors/design-proposals/security_context.md
		// +optional
		SecurityContext: &corev1.SecurityContext{
			Privileged: &privileged,
		},
	}
	return &exec
}

// Returns tuple of: (workflow was updated, node has completed, error)
func (wfc *WorkflowController) executeTemplate(wf *wfv1.Workflow, templateName string, args *wfv1.Arguments, nodeName string) (bool, error) {
	fmt.Printf("Executing %s: %v, args: %#v\n", nodeName, templateName, args)
	nodeID := wf.NodeID(nodeName)
	node, ok := wf.Status.Nodes[nodeID]
	if ok && node.Completed() {
		fmt.Printf("Node %s already completed\n", nodeName)
		return false, nil
	}
	tmpl := wf.GetTemplate(templateName)
	if tmpl == nil {
		err := errors.Errorf(errors.CodeBadRequest, "Node %s error: template '%s' undefined", nodeName, templateName)
		wf.Status.Nodes[nodeID] = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: "Error"}
		return true, err
	}

	switch tmpl.Type {
	case wfv1.TypeContainer:
		if !ok {
			// We have not yet created the pod
			status := wfv1.NodeStatusRunning
			err := wfc.createWorkflowContainer(wf, nodeName, tmpl, args)
			if err != nil {
				// TODO: may need to query pod status if we hit already exists error
				status = wfv1.NodeStatusError
			}
			wf.Status.Nodes[nodeID] = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: status}
			fmt.Printf("Initializing container node %s\n", nodeName)
			return true, nil
		}
		return false, nil

	case wfv1.TypeWorkflow:
		updates := false
		if !ok {
			fmt.Printf("Initializing workflow node %s\n", nodeName)
			node = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: wfv1.NodeStatusRunning}
			wf.Status.Nodes[nodeID] = node
			updates = true
		}
		for i, stepGroup := range tmpl.Steps {
			sgNodeName := fmt.Sprintf("%s[%d]", nodeName, i)
			sgUpdates, err := wfc.executeStepGroup(wf, stepGroup, sgNodeName)
			if err != nil {
				node.Status = wfv1.NodeStatusError
				wf.Status.Nodes[nodeID] = node
				return true, err
			}
			updates = updates || sgUpdates
			sgNodeID := wf.NodeID(sgNodeName)
			if !wf.Status.Nodes[sgNodeID].Completed() {
				fmt.Printf("Workflow step group %s not yet completed\n", sgNodeName)
				return updates, nil
			}
			if !wf.Status.Nodes[sgNodeID].Successful() {
				fmt.Printf("Workflow step group %s not successful\n", sgNodeName)
				node.Status = wfv1.NodeStatusFailed
				wf.Status.Nodes[nodeID] = node
				return true, nil
			}
		}
		node.Status = wfv1.NodeStatusSucceeded
		wf.Status.Nodes[nodeID] = node
		return true, nil

	default:
		wf.Status.Nodes[nodeID] = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: "Error"}
		return true, fmt.Errorf("Unknown type: %s", tmpl.Type)
	}
}

func (wfc *WorkflowController) executeStepGroup(wf *wfv1.Workflow, stepGroup map[string]wfv1.WorkflowStep, nodeName string) (bool, error) {
	nodeID := wf.NodeID(nodeName)
	node, ok := wf.Status.Nodes[nodeID]
	if ok && node.Completed() {
		fmt.Printf("Step group node %s already marked completed\n", nodeName)
		return false, nil
	}
	updates := false
	if !ok {
		fmt.Printf("Initializing step group node %s\n", nodeName)
		node = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: "Running"}
		wf.Status.Nodes[nodeID] = node
		updates = true
	}
	childNodeIDs := make([]string, 0)
	// First kick off all parallel steps in the group
	for stepName, step := range stepGroup {
		childNodeName := fmt.Sprintf("%s.%s", nodeName, stepName)
		childNodeIDs = append(childNodeIDs, wf.NodeID(childNodeName))
		sUpdates, err := wfc.executeTemplate(wf, step.Template, &step.Arguments, childNodeName)
		updates = updates || sUpdates
		if err != nil {
			node.Status = wfv1.NodeStatusError
			wf.Status.Nodes[nodeID] = node
			return true, err
		}
	}
	// Return if not all children completed
	for _, childNodeID := range childNodeIDs {
		if !wf.Status.Nodes[childNodeID].Completed() {
			return updates, nil
		}
	}
	// All children completed. Determine status
	for _, childNodeID := range childNodeIDs {
		if !wf.Status.Nodes[childNodeID].Successful() {
			node.Status = wfv1.NodeStatusFailed
			wf.Status.Nodes[nodeID] = node
			updates = true
			fmt.Printf("Step group node %s deemed failed due to failure of %s\n", nodeID, childNodeID)
			return updates, nil
		}
	}
	node.Status = wfv1.NodeStatusSucceeded
	wf.Status.Nodes[nodeID] = node
	fmt.Printf("Step group node %s successful\n", nodeID)
	return true, nil
}
