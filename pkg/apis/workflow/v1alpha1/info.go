package v1alpha1

// A link to another app.
// +patchStrategy=merge
// +patchMergeKey=name
type Link struct {
	// The name of the link, E.g. "Workflow Logs" or "Pod Logs"
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// "workflow", "pod", "pod-logs", "event-source-logs", "sensor-logs", "workflow-list" or "chat"
	Scope string `json:"scope" protobuf:"bytes,2,opt,name=scope"`
	// The type of the link. E.g. "external", "internal". If the link type is external, it will open in a new tab.
	Type string `json:"type" protobuf:"bytes,4,opt,name=type"`
	// The URL. Can contain "${metadata.namespace}", "${metadata.name}", "${status.startedAt}", "${status.finishedAt}" or any other element in workflow yaml, e.g. "${workflow.metadata.annotations.userDefinedKey}"
	URL string `json:"url" protobuf:"bytes,3,opt,name=url"`
}

// Column is a custom column that will be exposed in the Workflow List View.
// +patchStrategy=merge
// +patchMergeKey=name
type Column struct {
	// The name of this column, e.g., "Workflow Completed".
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// The type of this column, "label" or "annotation".
	Type string `json:"type" protobuf:"bytes,2,opt,name=type"`
	// The key of the label or annotation, e.g., "workflows.argoproj.io/completed".
	Key string `json:"key" protobuf:"bytes,3,opt,name=key"`
}
