import {fireEvent, render, waitFor} from '@testing-library/react';
import {createMemoryHistory} from 'history';
import React from 'react';

import {App} from '../../../app';
import {exampleWorkflowTemplate} from '../../../shared/examples';
import {WorkflowTemplate} from '../../../shared/models';
import requests from '../../../shared/services/requests';
import {WorkflowTemplateService} from '../../../shared/services/workflow-template-service';

jest.mock('../../../shared/services/workflow-template-service');
jest.mock('../../../shared/services/requests');

describe('WorkflowsList', () => {
    let workflowTemplate: WorkflowTemplate;

    beforeEach(() => {
        jest.clearAllMocks();
        localStorage.clear();
        workflowTemplate = exampleWorkflowTemplate('argo');
        // Mock out API calls
        jest.spyOn(WorkflowTemplateService, 'list').mockResolvedValue({items: [workflowTemplate]} as any);
        jest.spyOn(requests, 'get').mockResolvedValue({body: {}} as any);
        jest.spyOn(requests, 'post').mockReturnValue({send: jest.fn().mockResolvedValue('')} as any);
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

    it('adds repeated phase and label query parameters when filters are selected', async () => {
        const history = createMemoryHistory();
        history.push('/workflows?namespace=argo');

        const {findByLabelText, container} = render(<App history={history} />);
        expect(history.location.search).toBe('?namespace=argo&limit=50');

        // Select multiple phases.
        const runningFilter = await findByLabelText('Running');
        const failedFilter = await findByLabelText('Failed');
        runningFilter.click();
        failedFilter.click();

        // Add multiple labels.
        const labelInput = container.querySelector<HTMLInputElement>('.tags-input input');
        if (!labelInput) {
            throw new Error('Labels input not found');
        }
        expect(labelInput).toBeInTheDocument();

        fireEvent.change(labelInput, {target: {value: 'team=ml'}});
        fireEvent.keyUp(labelInput, {key: 'Enter', keyCode: 13});
        fireEvent.change(labelInput, {target: {value: 'env=prod'}});
        fireEvent.keyUp(labelInput, {key: 'Enter', keyCode: 13});

        // Verify selected filters are added to the URL.
        await waitFor(() => {
            expect(history.location.search).toBe('?namespace=argo&phase=Running&phase=Failed&label=team%3Dml&label=env%3Dprod&limit=50');
        });

        // Unselect one phase and one label.
        runningFilter.click();
        const teamLabelRemoveButton = await waitFor<HTMLElement>(() => {
            const teamLabel = Array.from(container.querySelectorAll<HTMLElement>('.tags-input__tag')).find(tag => tag.textContent?.includes('team=ml'));
            expect(teamLabel).toBeInTheDocument();
            const removeButton = teamLabel?.querySelector<HTMLElement>('.fa-times');
            expect(removeButton).toBeInTheDocument();
            return removeButton;
        });
        teamLabelRemoveButton.click();

        // Verify unselected filters are removed from the URL.
        await waitFor(() => {
            expect(history.location.search).toBe('?namespace=argo&phase=Failed&label=env%3Dprod&limit=50');
        });
    });

    // Regression test: when the sidePanel is open and the URL already contains
    // duplicate parameters[*] keys, historyUrl's append() preserves all duplicates instead of
    // collapsing them. After any filter change the count must not exceed 1 per parameter key.
    it('does not accumulate duplicate parameters[*] in URL when sidePanel is open', async () => {
        const history = createMemoryHistory();
        history.push('/workflows/argo?sidePanel=submit-new-workflow&parameters[message]=hello&parameters[message]=world');

        const {findByLabelText} = render(<App history={history} />);

        // Trigger a state change so the save-history effect re-runs while sidePanel is still open.
        const runningFilter = await findByLabelText('Running');
        runningFilter.click();

        await waitFor(() => {
            const params = new URLSearchParams(history.location.search);
            expect(params.getAll('parameters[message]')).toHaveLength(1);
        });
    });

    it('selects repeated phase and label filters from query parameters', async () => {
        const history = createMemoryHistory();
        history.push('/workflows?namespace=argo&phase=Running&phase=Failed&label=team%3Dml&label=env%3Dprod');

        const {findByLabelText, findByText} = render(<App history={history} />);

        expect(await findByLabelText('Running')).toBeChecked();
        expect(await findByLabelText('Failed')).toBeChecked();
        expect(await findByText('team=ml')).toBeInTheDocument();
        expect(await findByText('env=prod')).toBeInTheDocument();

        await waitFor(() => {
            expect(history.location.search).toBe('?namespace=argo&phase=Running&phase=Failed&label=team%3Dml&label=env%3Dprod&limit=50');
        });
    });
});
