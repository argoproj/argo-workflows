import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import {NotificationType} from 'argo-ui/src/components/notifications/notifications';
import React from 'react';

import {Context, ContextApis} from '../../../shared/context';
import {Workflow} from '../../../shared/models';
import {services} from '../../../shared/services';
import {OperationDisabled} from '../../../shared/workflow-operations-map';
import {WorkflowsToolbar} from './workflows-toolbar';

jest.mock('../../../shared/services', () => ({
    services: {
        workflows: {
            delete: jest.fn(),
            stop: jest.fn()
        }
    }
}));

const workflows = services.workflows as jest.Mocked<typeof services.workflows>;

// what super-agent rejects with when the API server denies a request
function forbidden(message: string) {
    return Object.assign(new Error('Forbidden'), {status: 403, response: {body: {code: 7, message}}});
}

const deleted = {workflowName: 'a', status: 'Deleted'};

function workflow(name: string, labels?: {[key: string]: string}): Workflow {
    return {metadata: {name, namespace: 'argo', uid: name, labels}, spec: {}} as Workflow;
}

describe('WorkflowsToolbar', () => {
    const show = jest.fn();
    const clearSelection = jest.fn();
    const loadWorkflows = jest.fn();

    const confirm = jest.fn();

    function renderToolbar(...selected: (string | Workflow)[]) {
        const apis = {popup: {confirm}, notifications: {show}} as unknown as ContextApis;
        const wfs = selected.map(wf => (typeof wf === 'string' ? workflow(wf) : wf));
        render(
            <Context.Provider value={apis}>
                <WorkflowsToolbar
                    selectedWorkflows={new Map(wfs.map(wf => [wf.metadata.name, wf]))}
                    disabledActions={{} as OperationDisabled}
                    clearSelection={clearSelection}
                    loadWorkflows={loadWorkflows}
                />
            </Context.Provider>
        );
    }

    beforeEach(() => {
        jest.clearAllMocks();
        confirm.mockResolvedValue(true);
    });

    it('reports success when every delete succeeds', async () => {
        workflows.delete.mockResolvedValue(deleted);
        renderToolbar('a', 'b');

        fireEvent.click(screen.getByText('DELETE'));

        await waitFor(() => expect(loadWorkflows).toHaveBeenCalled());
        expect(show).toHaveBeenCalledTimes(1);
        expect(show).toHaveBeenCalledWith({content: "Performed 'DELETE' on selected workflows.", type: NotificationType.Success});
    });

    it('reports the error, not success, when a delete is denied', async () => {
        workflows.delete.mockRejectedValue(forbidden('Permission denied, you do not have access to delete workflows'));
        renderToolbar('a');

        fireEvent.click(screen.getByText('DELETE'));

        await waitFor(() => expect(loadWorkflows).toHaveBeenCalled());
        expect(show).toHaveBeenCalledTimes(1);
        expect(show).toHaveBeenCalledWith({
            content: 'Unable to delete workflow a in the cluster: Forbidden: Permission denied, you do not have access to delete workflows',
            type: NotificationType.Error
        });
    });

    it('does not report success when only some deletes succeed', async () => {
        workflows.delete.mockImplementation(name => (name === 'a' ? Promise.resolve(deleted) : Promise.reject(forbidden('Permission denied'))));
        renderToolbar('a', 'b');

        fireEvent.click(screen.getByText('DELETE'));

        await waitFor(() => expect(loadWorkflows).toHaveBeenCalled());
        expect(show).toHaveBeenCalledTimes(1);
        expect(show).toHaveBeenCalledWith({content: 'Unable to delete workflow b in the cluster: Forbidden: Permission denied', type: NotificationType.Error});
    });

    it('does not report success when nothing was deleted', async () => {
        // confirm the delete, then decline to delete from the archive
        confirm.mockResolvedValueOnce(true).mockResolvedValueOnce(false);
        renderToolbar(workflow('a', {'workflows.argoproj.io/workflow-archiving-status': 'Persisted'}));

        fireEvent.click(screen.getByText('DELETE'));

        await waitFor(() => expect(loadWorkflows).toHaveBeenCalled());
        expect(workflows.delete).not.toHaveBeenCalled();
        expect(show).not.toHaveBeenCalled();
    });

    it('reports the error when another action is denied', async () => {
        workflows.stop.mockRejectedValue(forbidden('Permission denied, you do not have access to update workflows'));
        renderToolbar('a');

        fireEvent.click(screen.getByText('STOP'));

        await waitFor(() => expect(loadWorkflows).toHaveBeenCalled());
        expect(show).toHaveBeenCalledTimes(1);
        expect(show).toHaveBeenCalledWith({
            content: 'Unable to stop workflow a: Forbidden: Permission denied, you do not have access to update workflows',
            type: NotificationType.Error
        });
    });
});
