import {NodePhase, Workflow} from '../../models';
import {services} from './services';
import {Utils} from './utils';

export type OperationDisabled = {
    [action in WorkflowOperationName]: boolean;
};

export type WorkflowOperationName = 'RETRY' | 'RESUBMIT' | 'SUSPEND' | 'RESUME' | 'STOP' | 'TERMINATE' | 'DELETE';

export interface WorkflowOperation {
    title: string;
    action: () => Promise<any>;
    iconClassName: string;
    disabled: (wf: Workflow) => boolean;
}

export const WorkflowOperations = {
    RETRY: {
        title: 'RETRY',
        iconClassName: 'fa fa-undo',
        disabled: (wf: Workflow) => {
            const workflowPhase: NodePhase = wf && wf.status ? wf.status.phase : undefined;
            return workflowPhase === undefined || !(workflowPhase === 'Failed' || workflowPhase === 'Error');
        },
        action: services.workflows.retry
    },
    RESUBMIT: {
        title: 'RESUBMIT',
        iconClassName: 'fa fa-plus-circle',
        disabled: () => false,
        action: services.workflows.resubmit
    },
    SUSPEND: {
        title: 'SUSPEND',
        iconClassName: 'fa fa-pause',
        disabled: (wf: Workflow) => !Utils.isWorkflowRunning(wf) || Utils.isWorkflowSuspended(wf),
        action: services.workflows.suspend
    },
    RESUME: {
        title: 'RESUME',
        iconClassName: 'fa fa-play',
        disabled: (wf: Workflow) => !Utils.isWorkflowSuspended(wf),
        action: services.workflows.resume
    },
    STOP: {
        title: 'STOP',
        iconClassName: 'fa fa-stop-circle',
        disabled: (wf: Workflow) => !Utils.isWorkflowSuspended(wf),
        action: services.workflows.stop
    },
    TERMINATE: {
        title: 'TERMINATE',
        iconClassName: 'fa fa-times-circle',
        disabled: (wf: Workflow) => !Utils.isWorkflowSuspended(wf),
        action: services.workflows.terminate
    },
    DELETE: {
        title: 'DELETE',
        iconClassName: 'fa fa-trash',
        disabled: () => false,
        action: services.workflows.delete
    }
};
