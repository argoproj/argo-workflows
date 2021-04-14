package v1alpha1

// A link to another app.
// +patchStrategy=merge
// +patchMergeKey=name
type Link struct {
	// The name of the link, E.g. "Workflow Logs" or "Pod Logs"
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// "workflow", "pod", "pod-logs", "event-source-logs", "sensor-logs" or "chat"
	Scope string `json:"scope" protobuf:"bytes,2,opt,name=scope"`
	// The URL. Can contain "${metadata.namespace}", "${metadata.name}", "${status.startedAt}", "${status.finishedAt}" or any other element in workflow yaml, e.g. "${workflow.metadata.annotations.userDefinedKey}"
	URL string `json:"url" protobuf:"bytes,3,opt,name=url"`
}
