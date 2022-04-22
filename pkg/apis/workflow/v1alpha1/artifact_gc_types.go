package v1alpha1

type ArtifactGCStrategies []ArtifactGCStrategy

func (s ArtifactGCStrategies) Contains(v ArtifactGCStrategy) bool {
	for _, k := range s {
		if k == v {
			return true
		}
	}
	return false
}

type ArtifactGCStrategy string

const (
	ArtifactGCOnAWorkflowCompletion ArtifactGCStrategy = "WorkflowCompletion"
	ArtifactGCOnWorkflowDeletion    ArtifactGCStrategy = "WorkflowDeletion"
)

type ArtifactGC struct {
	// Strategy is the strategy to use.
	// "" - do nothing
	// WorkflowCompletion - delete the artifact on completion of the workflow
	// WorkflowDeletion - delete the artifact when the workflow is deleted
	// +kubebuilder:validation:Enum=WorkflowCompletion;WorkflowDeletion
	Strategy ArtifactGCStrategy `json:"strategy,omitempty" protobuf:"bytes,1,opt,name=strategy,casttype=ArtifactGCStrategy"`
}

func (in *ArtifactGC) GetStrategy() ArtifactGCStrategy {
	if in == nil {
		return ""
	}
	return in.Strategy
}
