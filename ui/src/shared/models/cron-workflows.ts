import * as kubernetes from 'argo-ui/src/models/kubernetes';

import {Condition, WorkflowSpec} from './workflows';

export interface CronWorkflow {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ObjectMeta;
    spec: CronWorkflowSpec;
    status?: CronWorkflowStatus;
}

export type ConcurrencyPolicy = 'Allow' | 'Forbid' | 'Replace';

export interface CronWorkflowSpec {
    workflowSpec: WorkflowSpec;
    workflowMetadata?: kubernetes.ObjectMeta;
    schedules?: string[];
    concurrencyPolicy?: ConcurrencyPolicy;
    suspend?: boolean;
    startingDeadlineSeconds?: number;
    successfulJobsHistoryLimit?: number;
    failedJobsHistoryLimit?: number;
    timezone?: string;
    when?: string;
}

export interface CronWorkflowStatus {
    active: kubernetes.ObjectReference[];
    lastScheduledTime: kubernetes.Time;
    conditions?: Condition[];
    // maps a schedule of the spec using a `H` (hash) token to what it resolved to, set by the controller
    resolvedSchedules?: {[schedule: string]: string};
}

export interface CronWorkflowList {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ListMeta;
    items: CronWorkflow[];
}
