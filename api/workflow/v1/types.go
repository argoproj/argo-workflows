package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CRD constants
const (
	CRDSingular  string = "workflow"
	CRDPlural    string = "workflows"
	CRDShortName string = "wf"
	CRDGroup     string = "argoproj.io"
	CRDVersion   string = "v1"
	FullCRDName  string = CRDPlural + "." + CRDGroup
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
	Templates         []Template             `json:"templates"`
	Target            string                 `json:"target"`
	Arguments         Arguments              `json:"arguments"`
	Status            string                 `json:"status"`
	Results           map[string]interface{} `json:"results"`
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
	From interface{} `json:"from,omitempty"`
	Path string      `json:"path,omitempty"`
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
