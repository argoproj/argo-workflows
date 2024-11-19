package telemetry

const (
	AttribBuildVersion      string = `version`
	AttribBuildPlatform     string = `platform`
	AttribBuildGoVersion    string = `go_version`
	AttribBuildDate         string = `build_date`
	AttribBuildCompiler     string = `compiler`
	AttribBuildGitCommit    string = `git_commit`
	AttribBuildGitTreeState string = `git_treestate`
	AttribBuildGitTag       string = `git_tag`

	AttribCronWFName        string = `name`
	AttribConcurrencyPolicy string = `concurrency_policy`

	AttribDeprecatedFeature string = "feature"

	AttribErrorCause string = "cause"

	AttribLogLevel string = `level`

	AttribNodePhase string = `node_phase`

	AttribPodPhase         string = `phase`
	AttribPodNamespace     string = `namespace`
	AttribPodPendingReason string = `reason`

	AttribQueueName string = `queue_name`

	AttribRecentlyStarted string = `recently_started`

	AttribRequestKind = `kind`
	AttribRequestVerb = `verb`
	AttribRequestCode = `status_code`

	AttribTemplateName      string = `name`
	AttribTemplateNamespace string = `namespace`
	AttribTemplateCluster   string = `cluster_scope`

	AttribWorkerType string = `worker_type`

	AttribWorkflowNamespace string = `namespace`
	AttribWorkflowPhase     string = `phase`
	AttribWorkflowStatus           = `status`
	AttribWorkflowType             = `type`
)
