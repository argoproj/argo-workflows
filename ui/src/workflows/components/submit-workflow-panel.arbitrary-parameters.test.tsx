import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import {createMemoryHistory} from 'history';
import React from 'react';

import {Context} from '../../shared/context';
import {Parameter, Template} from '../../shared/models';
import {services} from '../../shared/services';
import {SubmitWorkflowPanel} from './submit-workflow-panel';

function renderPanel(workflowParameters: Parameter[] = [], templates: Template[] = [], kind: 'ClusterWorkflowTemplate' | 'WorkflowTemplate' = 'WorkflowTemplate') {
    const navigation = {goto: jest.fn()};

    render(
        <Context.Provider value={{navigation} as any}>
            <SubmitWorkflowPanel
                kind={kind}
                namespace='argo'
                name='example-template'
                entrypoint=''
                templates={templates}
                workflowParameters={workflowParameters}
                history={createMemoryHistory()}
            />
        </Context.Provider>
    );

    return navigation;
}

afterEach(() => {
    jest.restoreAllMocks();
});

describe('SubmitWorkflowPanel arbitrary parameters', () => {
    it('appends an arbitrary parameter after declared workflow and entrypoint parameters', async () => {
        const submit = jest.spyOn(services.workflows, 'submit').mockResolvedValue({metadata: {namespace: 'argo', name: 'submitted-workflow'}} as any);
        const navigation = renderPanel(
            [{name: 'workflow', default: 'workflow-value'}],
            [{name: 'main', inputs: {parameters: [{name: 'entrypoint', default: 'entrypoint-value'}]}}],
            'ClusterWorkflowTemplate'
        );

        fireEvent.click(screen.getByRole('button', {name: 'Add a parameter'}));
        fireEvent.change(screen.getByLabelText('Arbitrary parameter 1 name'), {target: {value: 'custom'}});
        fireEvent.change(screen.getByLabelText('Arbitrary parameter 1 value'), {target: {value: 'a=b'}});
        fireEvent.click(document.querySelector('.select__value') as HTMLElement);
        fireEvent.click(screen.getByText('main'));

        expect(screen.getByLabelText('Arbitrary parameter 1 name')).toHaveValue('custom');
        expect(screen.getByLabelText('Arbitrary parameter 1 value')).toHaveValue('a=b');
        fireEvent.click(screen.getByRole('button', {name: 'Submit'}));

        await waitFor(() => {
            expect(submit).toHaveBeenCalledWith(
                'ClusterWorkflowTemplate',
                'example-template',
                'argo',
                expect.objectContaining({parameters: ['workflow=workflow-value', 'entrypoint=entrypoint-value', 'custom=a=b']})
            );
        });
        expect(navigation.goto).toHaveBeenCalledWith('/workflows/argo/submitted-workflow');
    });

    it('requires a name but allows an empty value', async () => {
        const submit = jest.spyOn(services.workflows, 'submit').mockResolvedValue({metadata: {namespace: 'argo', name: 'submitted-workflow'}} as any);
        renderPanel();

        expect(screen.getByText('No parameters')).toBeInTheDocument();
        fireEvent.click(screen.getByRole('button', {name: 'Add a parameter'}));

        expect(screen.queryByText('No parameters')).not.toBeInTheDocument();
        expect(screen.getByText('Parameter name is required')).toBeInTheDocument();
        expect(screen.getByRole('button', {name: 'Submit'})).toBeDisabled();
        fireEvent.click(screen.getByRole('button', {name: 'Submit'}));
        expect(submit).not.toHaveBeenCalled();

        fireEvent.change(screen.getByLabelText('Arbitrary parameter 1 name'), {target: {value: 'empty-value'}});
        expect(screen.getByRole('button', {name: 'Submit'})).toBeEnabled();
        fireEvent.click(screen.getByRole('button', {name: 'Submit'}));

        await waitFor(() => {
            expect(submit).toHaveBeenCalledWith('WorkflowTemplate', 'example-template', 'argo', expect.objectContaining({parameters: ['empty-value=']}));
        });
    });

    it('removes only the selected arbitrary parameter', async () => {
        const submit = jest.spyOn(services.workflows, 'submit').mockResolvedValue({metadata: {namespace: 'argo', name: 'submitted-workflow'}} as any);
        renderPanel();

        fireEvent.click(screen.getByRole('button', {name: 'Add a parameter'}));
        fireEvent.change(screen.getByLabelText('Arbitrary parameter 1 name'), {target: {value: 'removed'}});
        fireEvent.change(screen.getByLabelText('Arbitrary parameter 1 value'), {target: {value: 'value'}});
        fireEvent.click(screen.getByRole('button', {name: 'Add a parameter'}));
        fireEvent.change(screen.getByLabelText('Arbitrary parameter 2 name'), {target: {value: 'kept'}});
        fireEvent.change(screen.getByLabelText('Arbitrary parameter 2 value'), {target: {value: 'other-value'}});
        fireEvent.click(screen.getByRole('button', {name: 'Remove arbitrary parameter 1'}));

        expect(screen.getByLabelText('Arbitrary parameter 1 name')).toHaveValue('kept');
        expect(screen.getByRole('button', {name: 'Submit'})).toBeEnabled();
        fireEvent.click(screen.getByRole('button', {name: 'Submit'}));

        await waitFor(() => {
            expect(submit).toHaveBeenCalledWith('WorkflowTemplate', 'example-template', 'argo', expect.objectContaining({parameters: ['kept=other-value']}));
        });
    });
});
