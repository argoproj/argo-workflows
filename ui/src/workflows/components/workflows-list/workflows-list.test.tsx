import {act, fireEvent, render, waitFor} from '@testing-library/react';
import {createMemoryHistory} from 'history';
import React from 'react';
import {EMPTY} from 'rxjs';

import {App} from '../../../app';
import {exampleWorkflow, exampleWorkflowTemplate} from '../../../shared/examples';
import {WorkflowTemplate} from '../../../shared/models';
import requests from '../../../shared/services/requests';
import {WorkflowTemplateService} from '../../../shared/services/workflow-template-service';
import {WorkflowsService} from '../../../shared/services/workflows-service';

jest.mock('../../../shared/services/workflow-template-service');
jest.mock('../../../shared/services/requests');

describe('WorkflowsList', () => {
    let workflowTemplate: WorkflowTemplate;

    beforeEach(() => {
        jest.clearAllMocks();
        workflowTemplate = exampleWorkflowTemplate('argo');
        // Mock out API calls
        jest.spyOn(WorkflowTemplateService, 'list').mockResolvedValue({items: [workflowTemplate]} as any);
        jest.spyOn(requests, 'get').mockResolvedValue({body: {}} as any);
        jest.spyOn(requests, 'post').mockReturnValue({send: jest.fn().mockResolvedValue('')} as any);
        jest.spyOn(WorkflowsService, 'list').mockResolvedValue({metadata: {resourceVersion: '1'}, items: []} as any);
        jest.spyOn(WorkflowsService, 'watchFields').mockReturnValue(EMPTY);
    });

    it('renders with "Submit New Workflow" and updates URL when clicked', async () => {
        const history = createMemoryHistory();
        history.push('/workflows?namespace=argo');

        const {getByRole, container} = render(<App history={history} />);
        expect(history.location.search).toBe('?namespace=argo&limit=50');

        // Click "Submit New Workflow" button and verify URL updates
        const submitWorkflowButton = getByRole('button', {name: 'Submit New Workflow'});
        expect(submitWorkflowButton).toBeInTheDocument();
        submitWorkflowButton.click();
        await waitFor(() => {
            expect(history.location.search).toBe('?namespace=argo&sidePanel=submit-new-workflow&limit=50');
        });

        // Wait for close button, then press it and verify URL updates
        // TODO: use findByRole once the close button has an aria-label
        const closeButton = await waitFor<HTMLElement>(() => {
            const closeButton = container.querySelector<HTMLElement>('button.sliding-panel__close');
            expect(closeButton).toBeInTheDocument();
            return closeButton;
        });
        closeButton.click();

        // Check sidePanel was removed from URL.
        await waitFor(() => {
            expect(history.location.search).toBe('?namespace=argo&limit=50');
        });
    });

    it('opens workflow creator with pre-filled URL parameters', async () => {
        const history = createMemoryHistory();
        history.push(`/workflows?sidePanel=submit-new-workflow&template=${workflowTemplate.metadata.name}&parameters[message]=test-hello`);

        const {findByRole, findByDisplayValue} = render(<App history={history} />);

        expect(await findByRole('heading', {name: 'Submit Workflow'})).toBeInTheDocument();
        expect(await findByRole('heading', {name: `argo/${workflowTemplate.metadata.name}`})).toBeInTheDocument();

        const parameterInput = await findByDisplayValue('test-hello');
        expect(parameterInput).toBeInTheDocument();
    });

    it('retains query parameters when side panel is closed', async () => {
        const history = createMemoryHistory();
        history.push('/workflows?namespace=argo&phase=Pending&label=test');

        const {getByRole, container} = render(<App history={history} />);
        expect(history.location.search).toBe('?namespace=argo&phase=Pending&label=test&limit=50');

        // Click "Submit New Workflow" button and verify URL updates
        const submitWorkflowButton = getByRole('button', {name: 'Submit New Workflow'});
        expect(submitWorkflowButton).toBeInTheDocument();
        submitWorkflowButton.click();
        await waitFor(() => {
            expect(history.location.search).toBe('?namespace=argo&sidePanel=submit-new-workflow&phase=Pending&label=test&limit=50');
        });

        // Wait for close button, then press it and verify URL updates
        // TODO: use findByRole once the close button has an aria-label
        const closeButton = await waitFor<HTMLElement>(() => {
            const closeButton = container.querySelector<HTMLElement>('button.sliding-panel__close');
            expect(closeButton).toBeInTheDocument();
            return closeButton;
        });
        closeButton.click();

        // Check sidePanel was removed from URL but phase remains
        await waitFor(() => {
            expect(history.location.search).toBe('?namespace=argo&phase=Pending&label=test&limit=50');
        });
    });

    it('clears workflows and avoids an empty-state message after a namespace list is denied', async () => {
        const workflow = exampleWorkflowWithStatus('namespace-a', 'workflow-from-namespace-a');
        const list = jest
            .spyOn(WorkflowsService, 'list')
            .mockResolvedValueOnce({metadata: {resourceVersion: '1'}, items: [workflow]} as any)
            .mockRejectedValueOnce(
                Object.assign(new Error('Request failed with status code 403'), {
                    response: {body: {message: 'Permission denied'}}
                })
            );
        const history = createMemoryHistory();
        history.push('/workflows/namespace-a');

        const view = render(<App history={history} />);
        expect(await view.findByText('workflow-from-namespace-a')).toBeInTheDocument();

        const namespaceInput = view.container.querySelector('.input-filter input') as HTMLInputElement;
        fireEvent.change(namespaceInput, {target: {value: 'namespace-b'}});
        fireEvent.keyUp(namespaceInput, {key: 'Enter', keyCode: 13});

        expect(await view.findByText(/Permission denied/)).toBeInTheDocument();
        expect(view.queryByText('workflow-from-namespace-a')).not.toBeInTheDocument();
        expect(view.queryByText('No workflows')).not.toBeInTheDocument();
        expect(list.mock.calls[1][0]).toBe('namespace-b');
        view.unmount();
    });

    it('ignores an older namespace list that resolves after the newer request fails', async () => {
        const pendingList = deferred<any>();
        const list = jest.spyOn(WorkflowsService, 'list').mockReturnValueOnce(pendingList.promise).mockRejectedValueOnce(new Error('Request failed with status code 403'));
        const history = createMemoryHistory();
        history.push('/workflows/namespace-a');

        const view = render(<App history={history} />);
        const namespaceInput = view.container.querySelector('.input-filter input') as HTMLInputElement;
        fireEvent.change(namespaceInput, {target: {value: 'namespace-b'}});
        fireEvent.keyUp(namespaceInput, {key: 'Enter', keyCode: 13});

        expect(await view.findByText(/403/)).toBeInTheDocument();
        const oldWorkflow = exampleWorkflowWithStatus('namespace-a', 'late-workflow-from-namespace-a');
        await act(async () => {
            pendingList.resolve({metadata: {resourceVersion: '1'}, items: [oldWorkflow]});
            await pendingList.promise;
        });

        expect(view.queryByText('late-workflow-from-namespace-a')).not.toBeInTheDocument();
        expect(WorkflowsService.watchFields).not.toHaveBeenCalled();
        expect(list.mock.calls[1][0]).toBe('namespace-b');
        view.unmount();
    });
});

function deferred<T>() {
    let resolve: (value: T) => void;
    const promise = new Promise<T>(res => {
        resolve = res;
    });
    return {promise, resolve: resolve!};
}

function exampleWorkflowWithStatus(namespace: string, name: string) {
    const workflow = exampleWorkflow(namespace);
    workflow.metadata.name = name;
    workflow.metadata.uid = `${namespace}-${name}`;
    workflow.metadata.creationTimestamp = '2026-01-01T00:00:00Z';
    workflow.status = {phase: 'Running', nodes: {}, startedAt: '2026-01-01T00:00:00Z'} as any;
    return workflow;
}
