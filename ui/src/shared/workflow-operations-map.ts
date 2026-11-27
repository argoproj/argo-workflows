import type {NodePhase, Workflow} from './models';
import {services} from './services';
import {WorkflowDeleteResponse} from './services/responses';

export type OperationDisabled = {
    [action in WorkflowOperationName]: boolean;
};

export type WorkflowOperationName = 'RETRY' | 'RESUBMIT' | 'SUSPEND' | 'RESUME' | 'STOP' | 'TERMINATE' | 'DELETE';

export interface WorkflowOperation {
    title: WorkflowOperationName;
    action: WorkflowOperationAction;
    iconClassName: string;
    disabled: (wf: Workflow) => boolean;
    /** Hover text shown on the action button in the UI. */
    description?: string;
}

export type WorkflowOperationAction = (wf: Workflow) => Promise<Workflow | WorkflowDeleteResponse>;

export interface WorkflowOperations {
    [name: string]: WorkflowOperation;
}

export const WorkflowOperationsMap: WorkflowOperations = {
    RETRY: {
        title: 'RETRY',
        iconClassName: 'fa fa-undo',
        description: 'Rerun failed or errored steps on this workflow',
        disabled: (wf: Workflow) => {
            const workflowPhase: NodePhase = wf && wf.status ? wf.status.phase : undefined;
            return workflowPhase === undefined || !(workflowPhase === 'Failed' || workflowPhase === 'Error');
        },
        action: (wf: Workflow) => services.workflows.retry(wf.metadata.name, wf.metadata.namespace, null)
    },
    RESUBMIT: {
        title: 'RESUBMIT',
        iconClassName: 'fa fa-plus-circle',
        description: 'Create a new workflow from the same spec',
        disabled: () => false,
        action: (wf: Workflow) => services.workflows.resubmit(wf.metadata.name, wf.metadata.namespace, null)
    },
    SUSPEND: {
        title: 'SUSPEND',
        iconClassName: 'fa fa-pause',
        disabled: (wf: Workflow) => !isWorkflowRunning(wf) || isWorkflowSuspended(wf),
        action: (wf: Workflow) => services.workflows.suspend(wf.metadata.name, wf.metadata.namespace)
    },
    RESUME: {
        title: 'RESUME',
        iconClassName: 'fa fa-play',
        disabled: (wf: Workflow) => !isWorkflowRunning(wf) || !isWorkflowSuspended(wf),
        action: (wf: Workflow) => services.workflows.resume(wf.metadata.name, wf.metadata.namespace, null)
    },
    STOP: {
        title: 'STOP',
        iconClassName: 'fa fa-stop-circle',
        description: 'Stop the workflow and run exit handlers',
        disabled: (wf: Workflow) => !isWorkflowRunning(wf),
        action: (wf: Workflow) => services.workflows.stop(wf.metadata.name, wf.metadata.namespace)
    },
    TERMINATE: {
        title: 'TERMINATE',
        iconClassName: 'fa fa-times-circle',
        description: 'Stop the workflow immediately without exit handlers',
        disabled: (wf: Workflow) => !isWorkflowRunning(wf),
        action: (wf: Workflow) => services.workflows.terminate(wf.metadata.name, wf.metadata.namespace)
    },
    DELETE: {
        title: 'DELETE',
        iconClassName: 'fa fa-trash',
        disabled: () => false,
        action: (wf: Workflow) => services.workflows.delete(wf.metadata.name, wf.metadata.namespace)
    }
};

function isWorkflowSuspended(wf: Workflow): boolean {
    if (!wf?.spec) {
        return false;
    }
    if (wf.spec.suspend) {
        return true;
    }
    if (wf.status?.nodes) {
        for (const node of Object.values(wf.status.nodes)) {
            if (node.type === 'Suspend' && node.phase === 'Running') {
                return true;
            }
        }
    }
    return false;
}

function isWorkflowRunning(wf: Workflow): boolean {
    if (!wf?.spec) {
        return false;
    }
    return wf.status.phase === 'Running';
}
