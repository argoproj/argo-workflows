package v1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CRD constants
const (
	CRDKind      string = "Workflow"
	CRDSingular  string = "workflow"
	CRDPlural    string = "workflows"
	CRDShortName string = "wf"
	CRDGroup     string = "argoproj.io"
	CRDVersion   string = "v1"
	CRDFullName  string = CRDPlural + "." + CRDGroup
)

// Workflow types
const (
	TypeWorkflow  = "workflow"
	TypeContainer = "container"
)

// Workflow Status
const (
	WorkflowStatusCreated  = "Created"
	WorkflowStatusRunning  = "Running"
	WorkflowStatusSuccess  = "Success"
	WorkflowStatusFailed   = "Failed"
	WorkflowStatusCanceled = "Canceled"
)

// Create a Rest client with the new CRD Schema
var SchemeGroupVersion = schema.GroupVersion{Group: CRDGroup, Version: CRDVersion}

// Workflow is the definition of our CRD Workflow class
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              WorkflowSpec   `json:"spec"`
	Status            WorkflowStatus `json:"status"`
}

type WorkflowSpec struct {
	Templates  []Template `json:"templates"`
	Entrypoint string     `json:"entrypoint"`
}

type Template struct {
	Type    string  `json:"type,omitempty"`
	Inputs  Inputs  `json:"inputs,omitempty"`
	Outputs Outputs `json:"outputs,omitempty"`

	// Workflow fields
	Steps []map[string]WorkflowStep `json:"steps,omitempty"`

	// Container fields
	*corev1.Container
}

type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Workflow `json:"items"`
}

// Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another
type Inputs struct {
	Parameters map[string]*InputParameter `json:"parameters,omitempty"`
	Artifacts  map[string]*InputArtifact  `json:"artifacts,omitempty"`
}

// InputParameter indicate a passed string parameter to a service template with an optional default value
type InputParameter struct {
	Default *string       `json:"default,omitempty"`
	Options []interface{} `json:"options,omitempty"` // TODO: implement validation
	Regex   string        `json:"regex,omitempty"`   // TODO: implement validation
}

// InputArtifact indicates a passed string parameter to a service template with an optional default value
type InputArtifact struct {
	From string `json:"from,omitempty"`
	Path string `json:"path,omitempty"`
}

type Outputs struct {
	Artifacts OutputArtifacts `json:"artifacts,omitempty"`
}

type OutputArtifacts map[string]OutputArtifact

type OutputArtifact struct {
	Path string `json:"path,omitempty"`
	From string `json:"from,omitempty"`
}

// WorkflowStep is either a template ref, an inlined container, with added flags
type WorkflowStep struct {
	Template  string    `json:"template,omitempty"`
	Arguments Arguments `json:"arguments,omitempty"`
	Flags     []string  `json:"flags,omitempty"`
}

// Arguments to a template
type Arguments map[string]*string

type WorkflowStatus struct {
	Phase string                `json:"phase"`
	Tree  NodeTree              `json:"tree"`
	Nodes map[string]NodeStatus `json:"nodes"`
}

type NodeTree struct {
	Name     string     `json:"name"`
	Children []NodeTree `json:"children"`
}

type NodeStatus struct {
	ID      string                 `json:"id"`
	Name    string                 `json:"name"`
	Status  string                 `json:"status"`
	Outputs map[string]interface{} `json:"outputs"`
	//ReturnCode *int                    `json:"returnCode"`
}

const (
	NodeStatusPending   = "Pending"
	NodeStatusRunning   = "Running"
	NodeStatusSucceeded = "Succeeded"
	NodeStatusFailed    = "Failed"
	NodeStatusError     = "Error"
)

func (n NodeStatus) String() string {
	return fmt.Sprintf("%s (%s)", n.Name, n.ID)
}

func (n NodeStatus) Completed() bool {
	return n.Status == NodeStatusSucceeded || n.Status == NodeStatusFailed || n.Status == NodeStatusError
}

func (n NodeStatus) Successful() bool {
	return n.Status == NodeStatusSucceeded
}
