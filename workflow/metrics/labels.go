package metrics

const (
	labelErrorCause string = "cause"

	labelLogLevel string = `level`

	labelNodePhase string = `node_phase`

	labelPodPhase  string = `phase`
	labelQueueName string = `queue_name`

	labelRecentlyStarted string = `recently_started`

	labelRequestKind = `kind`
	labelRequestVerb = `verb`
	labelRequestCode = `status_code`

	labelWorkerType string = `worker_type`

	labelWorkflowNamespace string = `namespace`
	labelWorkflowPhase     string = `phase`
	labelWorkflowStatus           = `status`
	labelWorkflowType             = `type`
)
