import {uiUrl} from './base';
import {ContextApis} from './context';
import {services} from './services';

export interface WorkflowActionParams {
    ctx: ContextApis;
    name: string;
    namespace: string;
    handleError?: () => void;
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
