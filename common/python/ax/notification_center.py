#!/usr/bin/env python
#
# Copyright 2015-2017 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#

# Channels
CHANNEL_CONFIGURATION = "configuration"
CHANNEL_DEPLOYMENT = "deployment"
CHANNEL_JOB = "job"
CHANNEL_SPENDING = "spending"
CHANNEL_SYSTEM = "system"

# Severities
SEVERITY_CRITICAL = "critical"
SEVERITY_WARNING = "warning"
SEVERITY_INFO = "info"

# Namespaces
NAMESPACE_AXSYS = "axsys"
NAMESPACE_AXUSER = "axuser"

# Facilities
FACILITY_AXAMM = "{}.axamm".format(NAMESPACE_AXSYS)
FACILITY_AXDB = "{}.axdb".format(NAMESPACE_AXSYS)
FACILITY_AXMON = "{}.axmon".format(NAMESPACE_AXSYS)
FACILITY_AXOPS = "{}.axops".format(NAMESPACE_AXSYS)
FACILITY_AXSTATS = "{}.axstats".format(NAMESPACE_AXSYS)
FACILITY_AX_ARTIFACT_MANAGER = "{}.axartifactmanager".format(NAMESPACE_AXSYS)
FACILITY_AX_CONSOLE = "{}.axconsole".format(NAMESPACE_AXSYS)
FACILITY_AX_EVENT_TRIGGER = "{}.axeventtrigger".format(NAMESPACE_AXSYS)
FACILITY_AX_NOTIFICATION_CENTER = "{}.ax-notification-center".format(NAMESPACE_AXSYS)
FACILITY_AX_SCHEDULER = "{}.axscheduler".format(NAMESPACE_AXSYS)
FACILITY_AX_WORKFLOW_ADC = "{}.axworkflowadc".format(NAMESPACE_AXSYS)
FACILITY_CRON = "{}.cron".format(NAMESPACE_AXSYS)
FACILITY_DEFAULT_HTTP_BACKEND = "{}.default-http-backend".format(NAMESPACE_AXSYS)
FACILITY_GATEWAY = "{}.gateway".format(NAMESPACE_AXSYS)
FACILITY_INGRESS_CONTROLLER = "{}.ingress-controller".format(NAMESPACE_AXSYS)
FACILITY_KAFKA_ZK = "{}.kafka-zk".format(NAMESPACE_AXSYS)
FACILITY_REDIS = "{}.redis".format(NAMESPACE_AXSYS)
FACILITY_PLATFORM = "{}.platform".format(NAMESPACE_AXSYS)
FACILITY_FIXTUREMANAGER = "{}.fixturemanager".format(NAMESPACE_AXSYS)

# Configuration codes
CODE_CONFIGURATION_NOTIFICATION_INVALID_SMTP = "configuration.notification.invalid_smtp"
CODE_CONFIGURATION_NOTIFICATION_INVALID_SLACK = "configuration.notification.invalid_slack"
CODE_CONFIGURATION_SCM_CONNECTION_ERROR = "configuration.scm.connection_error"

# Deployment codes

# Job codes
CODE_JOB_CI_INVALID_COMMAND = "job.ci.invalid_command"
CODE_JOB_CI_INVALID_EVENT_TYPE = "job.ci.invalid_event_type"
CODE_JOB_CI_INVALID_SCM_TYPE = "job.ci.invalid_scm_type"
CODE_JOB_CI_EVENT_CREATION_FAILURE = "job.ci.event_creation_failure"
CODE_JOB_CI_EVENT_TRANSLATE_FAILURE = "job.ci.event_translate_failure"
CODE_JOB_CI_TEMPLATE_NOT_FOUND = "job.ci.template_not_found"
CODE_JOB_CI_YAML_UPDATE_FAILURE = "job.ci.yaml_update_failure"
CODE_JOB_CI_STATUS_REPORTING_FAILURE = "job.ci.status_reporting_failure"
CODE_JOB_CI_ELB_CREATION_FAILURE = "job.ci.elb_creation_failure"
CODE_JOB_CI_ELB_VERIFICATION_TIMEOUT = "job.ci.elb_verification_timeout"
CODE_JOB_CI_WEBHOOK_CREATION_FAILURE = "job.ci.webhook_creation_failure"
CODE_JOB_CI_ELB_DELETION_FAILURE = "job.ci.elb_deletion_failure"
CODE_JOB_CI_WEBHOOK_DELETION_FAILURE = "job.ci.webhook_deletion_failure"
CODE_JOB_CI_REPO_NOT_FOUND = "job.ci.repo_not_found"

CODE_JOB_SCHEDULER_INVALID_POLICY_DEFINITION = "job.scheduler.invalid_policy_definition"
CODE_JOB_SCHEDULER_INVALID_CRON_EXPRESSION = "job.scheduler.invalid_cron_expression"
CODE_JOB_SCHEDULER_CANNOT_ADD_POLICY = "job.scheduler.cannot_add_policy_to_scheduler"

CODE_ADC_MISSING_HEARTBEAT_FROM_WFE = "system.adc.missing_heartbeat_from_wfe"
# Spending codes

# System codes
CODE_PLATFORM_ERROR = "system.platform.error"
CODE_PLATFORM_CRITICAL = "system.platform.critical"

# Fixturemanager codes
CODE_SYSTEM_FIXTURE_TEMPLATE_DISCONNECTED = "system.fixture.template_disconnected"
CODE_SYSTEM_FIXTURE_INVALID_ATTRIBUTES = "system.fixture.invalid_attributes"

