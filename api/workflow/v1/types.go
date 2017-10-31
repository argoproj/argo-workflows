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

// Create a Rest client with the new CRD Schema
var SchemeGroupVersion = schema.GroupVersion{Group: CRDGroup, Version: CRDVersion}

// Workflow is the definition of our CRD Workflow class
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              WorkflowSpec   `json:"spec"`
	Status            WorkflowStatus `json:"status"`
}

type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Workflow `json:"items"`
}

type WorkflowSpec struct {
	Templates  []Template `json:"templates"`
	Entrypoint string     `json:"entrypoint"`
	Arguments  Arguments  `json:"arguments,omitempty"`
}

type Template struct {
	Name    string  `json:"name"`
	Inputs  Inputs  `json:"inputs,omitempty"`
	Outputs Outputs `json:"outputs,omitempty"`

	// Workflow fields
	Steps []map[string]WorkflowStep `json:"steps,omitempty"`

	// Container
	Container *corev1.Container `json:"container,omitempty"`

	// Script
	Script *Script `json:"script,omitempty"`
}

// Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another
type Inputs struct {
	Parameters map[string]*InputParameter `json:"parameters,omitempty"`
	Artifacts  map[string]*InputArtifact  `json:"artifacts,omitempty"`
}

// InputParameter indicate a passed string parameter to a service template with an optional default value
type InputParameter struct {
	Default *string `json:"default,omitempty"`
}

// InputArtifact indicates an artifact to place at a specified path
type InputArtifact struct {
	Path string              `json:"path,omitempty"`
	S3   *S3ArtifactSource   `json:"s3,omitempty"`
	Git  *GitArtifactSource  `json:"git,omitempty"`
	HTTP *HTTPArtifactSource `json:"http,omitempty"`
}

type Outputs struct {
	Artifacts  OutputArtifacts  `json:"artifacts,omitempty"`
	Parameters OutputParameters `json:"parameters,omitempty"`
}

type OutputArtifacts map[string]OutputArtifact

type OutputArtifact struct {
	Path        string               `json:"path,omitempty"`
	Destination *ArtifactDestination `json:"destination,omitempty"`
}

type OutputParameters map[string]OutputParameter

type OutputParameter struct {
	Path string `json:"path,omitempty"`
}

type Item interface{}

// WorkflowStep is a template ref
type WorkflowStep struct {
	Template  string    `json:"template,omitempty"`
	Arguments Arguments `json:"arguments,omitempty"`
	WithItems []Item    `json:"withItems,omitempty"`
	When      string    `json:"when,omitempty"`
}

// Arguments to a template
type Arguments map[string]interface{}

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
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Status  string  `json:"status"`
	Outputs Outputs `json:"outputs"`
	//ReturnCode *int                    `json:"returnCode"`
}

const (
	NodeStatusPending   = "Pending"
	NodeStatusRunning   = "Running"
	NodeStatusSucceeded = "Succeeded"
	NodeStatusSkipped   = "Skipped"
	NodeStatusFailed    = "Failed"
	NodeStatusError     = "Error"
)

func (n NodeStatus) String() string {
	return fmt.Sprintf("%s (%s)", n.Name, n.ID)
}

func (n NodeStatus) Completed() bool {
	return n.Status == NodeStatusSucceeded ||
		n.Status == NodeStatusFailed ||
		n.Status == NodeStatusError ||
		n.Status == NodeStatusSkipped
}

func (n NodeStatus) Successful() bool {
	return n.Status == NodeStatusSucceeded || n.Status == NodeStatusSkipped
}

type S3Bucket struct {
	Endpoint        string                   `json:"endpoint"`
	Bucket          string                   `json:"bucket"`
	AccessKeySecret corev1.SecretKeySelector `json:"accessKeySecret"`
	SecretKeySecret corev1.SecretKeySelector `json:"secretKeySecret"`
}

type S3ArtifactSource struct {
	S3Bucket `json:",inline"`
	Key      string `json:"key"`
}

type S3ArtifactDestination S3ArtifactSource

type GitArtifactSource struct {
	URL            string                    `json:"url"`
	UsernameSecret *corev1.SecretKeySelector `json:"usernameSecret"`
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret"`
}

type HTTPArtifactSource struct {
	URL string `json:"url"`
}

type ArtifactDestination struct {
	S3 *S3ArtifactDestination `json:"s3,omitempty"`
	// Future artifact destinations go here
	// * artifactory, nexus
}

type Script struct {
	Image   string   `json:"image"`
	Command []string `json:"command"`
	Source  string   `json:"source"`
}
