package v1alpha1

// A link to another app.
// +patchStrategy=merge
// +patchMergeKey=name
type Link struct {
	// The name of the link, E.g. "Workflow Logs" or "Pod Logs"
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Either "workflow" or "pod"
	Scope string `json:"scope" protobuf:"bytes,2,opt,name=scope"`
	// The URL. May contain "${metadata.namespace}", "${metadata.name}", "${status.startedAt}" and "${status.finishedAt}".
	URL string `json:"url" protobuf:"bytes,3,opt,name=url"`
}
