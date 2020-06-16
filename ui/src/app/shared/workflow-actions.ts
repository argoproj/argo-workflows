import {NodePhase, Workflow, WorkflowAction} from '../../models';
import {uiUrl} from './base';
import {ContextApis} from './context';
import {services} from './services';
import {Utils} from './utils';

export interface WorkflowActionParams {
    ctx: ContextApis;
    name: string;
    namespace: string;
    handleError?: () => void;
}

export type ActionDisabled = {
    [action in WorkflowAction]: boolean;
};

export function isDisabled(action: WorkflowAction, wf: Workflow) {
    const workflowPhase: NodePhase = wf && wf.status ? wf.status.phase : undefined;
    switch (action) {
        case 'retry':
            return workflowPhase === undefined || !(workflowPhase === 'Failed' || workflowPhase === 'Error');
        case 'resubmit':
            return false;
        case 'suspend':
            return !Utils.isWorkflowRunning(wf) || Utils.isWorkflowSuspended(wf);
        case 'resume':
            return !Utils.isWorkflowSuspended(wf);
        case 'stop':
            return !Utils.isWorkflowRunning(wf);
        case 'terminate':
            return !Utils.isWorkflowRunning(wf);
        case 'delete':
            return false;
        default:
            return false;
    }
}

export function deleteWorkflow(action: WorkflowActionParams) {
    return services.workflows
        .delete(action.name, action.namespace)
        .then(() => action.ctx.navigation.goto(uiUrl(`workflows/`)))
        .catch(() => {
            if (action.handleError) {
                action.handleError();
            }
            return;
        });
}

export function stopWorkflow(action: WorkflowActionParams) {
    return services.workflows
        .stop(action.name, action.namespace)
        .then(wf => action.ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
        .catch(() => {
            if (action.handleError) {
                action.handleError();
            }
            return;
        });
}

export function terminateWorkflow(action: WorkflowActionParams) {
    return services.workflows
        .terminate(action.name, action.namespace)
        .then(wf => action.ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
        .catch(() => {
            if (action.handleError) {
                action.handleError();
            }
        });
}

export function resumeWorkflow(action: WorkflowActionParams) {
    return services.workflows
        .resume(action.name, action.namespace)
        .then(wf => action.ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
        .catch(() => {
            if (action.handleError) {
                action.handleError();
            }
        });
}

export function suspendWorkflow(action: WorkflowActionParams) {
    return services.workflows
        .suspend(action.name, action.namespace)
        .then(wf => action.ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
        .catch(() => {
            if (action.handleError) {
                action.handleError();
            }
        });
}

export function resubmitWorkflow(action: WorkflowActionParams) {
    return services.workflows
        .resubmit(action.name, action.namespace)
        .then(wf => action.ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
        .catch(() => {
            if (action.handleError) {
                action.handleError();
            }
        });
}

export function retryWorkflow(action: WorkflowActionParams) {
    return services.workflows
        .retry(action.name, action.namespace)
        .then(wf => action.ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
        .catch(() => {
            if (action.handleError) {
                action.handleError();
            }
        });
}
