import {NodePhase, Workflow} from '../../models';
import {services} from './services';
import {Utils} from './utils';

export type ActionDisabled = {
    [action in WorkflowActionName]: boolean;
};

export type WorkflowActionName = 'RETRY' | 'RESUBMIT' | 'SUSPEND' | 'RESUME' | 'STOP' | 'TERMINATE' | 'DELETE';

export interface WorkflowAction {
    title: string;
    action: () => Promise<any>;
    iconClassName: string;
    disabled: (wf: Workflow) => boolean;
}

export const WorkflowActions = {
    retry: {
        title: 'retry',
        iconClassName: 'fa fa-undo',
        disabled: (wf: Workflow) => {
            const workflowPhase: NodePhase = wf && wf.status ? wf.status.phase : undefined;
            return workflowPhase === undefined || !(workflowPhase === 'Failed' || workflowPhase === 'Error');
        },
        action: services.workflows.retry
    },
    resubmit: {
        title: 'resubmit',
        iconClassName: 'fa fa-plus-circle',
        disabled: () => false,
        action: services.workflows.resubmit
    },
    suspend: {
        title: 'suspend',
        iconClassName: 'fa fa-pause',
        disabled: (wf: Workflow) => !Utils.isWorkflowRunning(wf) || Utils.isWorkflowSuspended(wf),
        action: services.workflows.suspend
    },
    resume: {
        title: 'resume',
        iconClassName: 'fa fa-play',
        disabled: (wf: Workflow) => !Utils.isWorkflowSuspended(wf),
        action: services.workflows.resume
    },
    stop: {
        title: 'stop',
        iconClassName: 'fa fa-stop-circle',
        disabled: (wf: Workflow) => !Utils.isWorkflowSuspended(wf),
        action: services.workflows.stop
    },
    terminate: {
        title: 'terminate',
        iconClassName: 'fa fa-times-circle',
        disabled: (wf: Workflow) => !Utils.isWorkflowSuspended(wf),
        action: services.workflows.terminate
    },
    delete: {
        title: 'delete',
        iconClassName: 'fa fa-trash',
        disabled: () => false,
        action: services.workflows.delete
    }
};
