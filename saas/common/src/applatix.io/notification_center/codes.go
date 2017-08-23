package notification_center

const (
	CodeConfigurationNotificationInvalidSmtp  = "configuration.notification.invalid_smtp"
	CodeConfigurationNotificationInvalidSlack = "configuration.notification.invalid_slack"

	CodeJobCiInvalidCommand         = "job.ci.invalid_command"
	CodeJobCiInvalidEventType       = "job.ci.invalid_event_type"
	CodeJobCiInvalidScmType         = "job.ci.invalid_scm_type"
	CodeJobCiEventCreationFailure   = "job.ci.event_creation_failure"
	CodeJobCiTemplateNotFound       = "job.ci.template_not_found"
	CodeJobCiYamlUpdateFailure      = "job.ci.yaml_update_failure"
	CodeJobCiStatusReportingFailure = "job.ci.status_reporting_failure"
	CodeJobCiElbCreationFailure     = "job.ci.elb_creation_failure"
	CodeJobCiElbVerificationTimeout = "job.ci.elb_verification_timeout"
	CodeJobCiWebhookCreationFailure = "job.ci.webhook_creation_failure"
	CodeJobCiElbDeletionFailure     = "job.ci.elb_deletion_failure"
	CodeJobCiWebhookDeletionFailure = "job.ci.webhook_deletion_failure"
	CodeJobCiRepoNotFound           = "job.ci.repo_not_found"

	CodeJobStatusStarted = "job.status.started"
	CodeJobStatusSuccess = "job.status.success"
	CodeJobStatusFailed  = "job.status.failed"

	CodeDeploymentStatusChanged = "deployment.status.changed"

	CodeEnabledPolicyInvalid      = "job.policy.invalid_enabled_policy"
	CodeInvalidPolicyBecomesValid = "job.policy.invalid_policy_becomes_valid"
	CodeEnabledPolicy             = "job.policy.enabled_policy"
	CodeDisabledPolicy            = "job.policy.disabled_policy"
)
