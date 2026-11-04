package indexes

// Indexers (i.e. IndexFunc) should be fast and should not return errors
// If an indexers returns an error, the cache will panic, and crash the Go VM.
// Indexers should be fast, if they are not, then the informer will get out of date
// and start returning old workflows, resulting in the operator reconciling out of date
// information, causing conflict errors and risking corrupted data.

const (
	ClusterWorkflowTemplateIndex = "clusterworkflowtemplate"
	CronWorkflowIndex            = "cronworkflow"
	NodeIDIndex                  = "nodeID"
	WorkflowIndex                = "workflow"
	WorkflowTemplateIndex        = "workflowtemplate"
	WorkflowPhaseIndex           = "workflow.phase"
	PodPhaseIndex                = "pod.phase"
	ConfigMapLabelsIndex         = "configmap.labels"
	ConditionsIndex              = "status.conditions"
	SemaphoreConfigIndexName     = "bySemaphoreConfigMap"
	UIDIndex                     = "uid"
	// PoolIndex indexes pooled WorkflowTaskSets, pool pods and pool Deployments by namespace and pool hash.
	PoolIndex = "pool"
	// WorkerPodUIDIndex indexes pooled WorkflowTaskResults by namespace and the UID of the worker pod that claimed them.
	WorkerPodUIDIndex = "workerPodUID"
)
