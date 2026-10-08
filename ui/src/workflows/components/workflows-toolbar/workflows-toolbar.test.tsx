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

function workflow(name: string): Workflow {
    return {metadata: {name, namespace: 'argo', uid: name}, spec: {}} as Workflow;
}

describe('WorkflowsToolbar', () => {
    const show = jest.fn();
    const clearSelection = jest.fn();
    const loadWorkflows = jest.fn();

    function renderToolbar(...names: string[]) {
        const apis = {popup: {confirm: jest.fn().mockResolvedValue(true)}, notifications: {show}} as unknown as ContextApis;
        render(
            <Context.Provider value={apis}>
                <WorkflowsToolbar
                    selectedWorkflows={new Map(names.map(name => [name, workflow(name)]))}
                    disabledActions={{} as OperationDisabled}
                    clearSelection={clearSelection}
                    loadWorkflows={loadWorkflows}
                />
            </Context.Provider>
        );
    }

    beforeEach(() => jest.clearAllMocks());

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
