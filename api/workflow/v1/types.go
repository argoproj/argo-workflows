package v1

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
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

// Workflow and node statuses
const (
	NodeStatusRunning   = "Running"
	NodeStatusSucceeded = "Succeeded"
	NodeStatusSkipped   = "Skipped"
	NodeStatusFailed    = "Failed"
	NodeStatusError     = "Error"
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
	Templates            []Template                    `json:"templates"`
	Entrypoint           string                        `json:"entrypoint"`
	Arguments            Arguments                     `json:"arguments,omitempty"`
	Volumes              []apiv1.Volume                `json:"volumes,omitempty"`
	VolumeClaimTemplates []apiv1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`
	Timeout              string                        `json:"timeout,omitempty"`
}

type Template struct {
	Name    string  `json:"name"`
	Inputs  Inputs  `json:"inputs,omitempty"`
	Outputs Outputs `json:"outputs,omitempty"`

	// Deamon indicates will allow a workflow to proceed to the next step if the container reaches readiness
	Daemon *bool `json:"daemon,omitempty"`

	// Workflow fields
	Steps [][]WorkflowStep `json:"steps,omitempty"`

	// Container
	Container *apiv1.Container `json:"container,omitempty"`

	// Script
	Script *Script `json:"script,omitempty"`

	// Sidecar containers
	Sidecars []Sidecar `json:"sidecars,omitempty"`

	// Location in which all files related to the step will be stored (logs, artifacts, etc...).
	// Can be overridden by individual items in Outputs. If omitted, will use the default
	// artifact repository location configured in the controller, appended with the
	// <workflowname>/<nodename> in the key.
	ArchiveLocation *ArtifactLocation `json:"archiveLocation,omitempty"`
}

// Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another
type Inputs struct {
	Parameters []Parameter `json:"parameters,omitempty"`
	Artifacts  []Artifact  `json:"artifacts,omitempty"`
}

// Parameter indicate a passed string parameter to a service template with an optional default value
type Parameter struct {
	Name    string  `json:"name"`
	Value   *string `json:"value,omitempty"`
	Default *string `json:"default,omitempty"`
}

// Artifact indicates an artifact to place at a specified path
type Artifact struct {
	// name of the artifact. must be unique within a template's inputs/outputs.
	Name string `json:"name"`

	// Path is the container path to the artifact
	Path string `json:"path,omitempty"`

	// mode bits to use on this file, must be a value between 0 and 0777
	// set when loading input artifacts.
	Mode *int32 `json:"mode,omitempty"`

	// From allows an artifact to reference an artifact from a previous step
	From string `json:"from,omitempty"`

	ArtifactLocation `json:",inline"`
}

// ArtifactLocation describes a location for a single or multiple artifacts.
// It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname).
// It is also used to describe the location of multiple artifacts such as the archive location
// of a single workflow step, which the executor will use as a default location to store its files.
type ArtifactLocation struct {
	S3   *S3Artifact   `json:"s3,omitempty"`
	Git  *GitArtifact  `json:"git,omitempty"`
	HTTP *HTTPArtifact `json:"http,omitempty"`
}

type Outputs struct {
	Parameters []Parameter `json:"parameters,omitempty"`
	Artifacts  []Artifact  `json:"artifacts,omitempty"`
	Result     *string     `json:"result,omitempty"`
	// TODO:
	// - Logs (log artifact(s) from the container?)
}

// WorkflowStep is a template ref
type WorkflowStep struct {
	Name      string    `json:"name,omitempty"`
	Template  string    `json:"template,omitempty"`
	Arguments Arguments `json:"arguments,omitempty"`
	WithItems []Item    `json:"withItems,omitempty"`
	When      string    `json:"when,omitempty"`
}

// Item expands a single workflow step into multiple parallel steps
type Item interface{}

// Arguments to a template
type Arguments struct {
	Parameters []Parameter `json:"parameters,omitempty"`
	Artifacts  []Artifact  `json:"artifacts,omitempty"`
}

// Sidecar is a container which runs alongside the main container
type Sidecar struct {
	apiv1.Container `json:",inline"`

	SidecarOptions `json:",inline"`
}

// SidecarOptions provide a way to customize the behavior of a sidecar and how it
// affects the main container.
type SidecarOptions struct {

	// MirrorVolumeMounts will mount the same volumes specified in the main container
	// to the sidecar (including artifacts), at the same mountPaths. This enables
	// dind daemon to partially see the same filesystem as the main container in
	// order to use features such as docker volume binding
	MirrorVolumeMounts *bool `json:"mirrorVolumeMounts,omitempty"`

	// Other sidecar options to consider:
	// * Lifespan - allow a sidecar to live longer than the main container and run to completion.
	// * PropogateFailure - if a sidecar fails, also fail the step
}

type WorkflowStatus struct {
	Nodes                  map[string]NodeStatus `json:"nodes"`
	PersistentVolumeClaims []apiv1.Volume        `json:"persistentVolumeClaims,omitempty"`
}

type NodeStatus struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	// PodIP captures the IP of the pod for deamoned steps
	PodIP string `json:"podIP,omitempty"`
	// Daemoned tracks whether or not this node was daemoned and need to be terminated
	Daemoned *bool `json:"daemoned,omitempty"`
	// Outputs captures output parameter values and artifact locations
	Outputs *Outputs `json:"outputs,omitempty"`
	// Children is a list of child node IDs
	Children []string `json:"children,omitempty"`
}

func (n NodeStatus) String() string {
	return fmt.Sprintf("%s (%s)", n.Name, n.ID)
}

// Completed returns whether or not the node has completed execution
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
	Endpoint        string                  `json:"endpoint"`
	Bucket          string                  `json:"bucket"`
	Region          string                  `json:"region,omitempty"`
	Insecure        *bool                   `json:"insecure,omitempty"`
	AccessKeySecret apiv1.SecretKeySelector `json:"accessKeySecret"`
	SecretKeySecret apiv1.SecretKeySelector `json:"secretKeySecret"`
}

type S3Artifact struct {
	S3Bucket `json:",inline"`
	Key      string `json:"key"`
}

type GitArtifact struct {
	Repo           string                   `json:"repo"`
	Revision       string                   `json:"revision,omitempty"`
	UsernameSecret *apiv1.SecretKeySelector `json:"usernameSecret,omitempty"`
	PasswordSecret *apiv1.SecretKeySelector `json:"passwordSecret,omitempty"`
}

type HTTPArtifact struct {
	URL string `json:"url"`
}

// Script is a template subtype to enable scripting through code steps
type Script struct {
	Image   string   `json:"image"`
	Command []string `json:"command"`
	Source  string   `json:"source"`
}

func (in *Inputs) GetArtifactByName(name string) *Artifact {
	for _, art := range in.Artifacts {
		if art.Name == name {
			return &art
		}
	}
	return nil
}

func (in *Inputs) GetParameterByName(name string) *Parameter {
	for _, param := range in.Parameters {
		if param.Name == name {
			return &param
		}
	}
	return nil
}

func (args *Arguments) GetArtifactByName(name string) *Artifact {
	for _, art := range args.Artifacts {
		if art.Name == name {
			return &art
		}
	}
	return nil
}

func (args *Arguments) GetParameterByName(name string) *Parameter {
	for _, param := range args.Parameters {
		if param.Name == name {
			return &param
		}
	}
	return nil
}

// HasLocation whether or not an artifact has a location defined
func (a *Artifact) HasLocation() bool {
	return a.S3 != nil || a.Git != nil || a.HTTP != nil
}
