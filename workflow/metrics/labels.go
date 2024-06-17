package metrics

const (
	labelBuildVersion      string = `version`
	labelBuildPlatform     string = `platform`
	labelBuildGoVer        string = `goversion`
	labelBuildDate         string = `build`
	labelBuildCompiler     string = `compiler`
	labelBuildGitCommit    string = `commit`
	labelBuildGitTreeState string = `treestate`
	labelBuildGitTag       string = `tag`

	labelCronWFName string = `name`

	labelErrorCause string = "cause"

	labelLogLevel string = `level`

	labelNodePhase string = `node_phase`

	labelPodPhase         string = `phase`
	labelPodNamespace     string = `namespace`
	labelPodPendingReason string = `reason`

	labelQueueName string = `queue_name`

	labelRecentlyStarted string = `recently_started`

	labelRequestKind = `kind`
	labelRequestVerb = `verb`
	labelRequestCode = `status_code`

	labelTemplateName      string = `name`
	labelTemplateNamespace string = `namespace`
	labelTemplateCluster   string = `cluster_scope`

	labelWorkerType string = `worker_type`

	labelWorkflowNamespace string = `namespace`
	labelWorkflowPhase     string = `phase`
	labelWorkflowStatus           = `status`
	labelWorkflowType             = `type`
)
