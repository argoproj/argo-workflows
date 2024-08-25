package metrics

const (
	labelBuildVersion      string = `version`
	labelBuildPlatform     string = `platform`
	labelBuildGoVersion    string = `go_version`
	labelBuildDate         string = `build_date`
	labelBuildCompiler     string = `compiler`
	labelBuildGitCommit    string = `git_commit`
	labelBuildGitTreeState string = `git_treestate`
	labelBuildGitTag       string = `git_tag`

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
