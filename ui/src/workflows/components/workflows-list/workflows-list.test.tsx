import {render, waitFor} from '@testing-library/react';
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
});
