import * as kubernetes from 'argo-ui/src/models/kubernetes';
import {WorkflowSpec} from './workflows';

export interface CronWorkflow {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ObjectMeta;
    spec: CronWorkflowSpec;
    status?: CronWorkflowStatus;
}

export interface CronWorkflowSpec {
    workflowSpec: WorkflowSpec;
    schedule: string;
    concurrencyPolicy?: string;
    suspend?: boolean;
    startingDeadlineSeconds?: number;
    successfulJobsHistoryLimit?: number;
    failedJobsHistoryLimit?: number;
    timezone?: string;
}

export interface CronWorkflowStatus {
    active: kubernetes.ObjectReference;
    lastScheduledTime: kubernetes.Time;
}

export interface CronWorkflowList {
    apiVersion?: string;
    kind?: string;
    metadata: kubernetes.ListMeta;
    items: CronWorkflow[];
}
