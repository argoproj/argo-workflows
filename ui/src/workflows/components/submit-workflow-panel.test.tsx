import {render, screen} from '@testing-library/react';
import {createMemoryHistory} from 'history';
import React from 'react';

import {Context} from '../../shared/context';
import {Parameter, Template} from '../../shared/models';
import {SubmitWorkflowPanel} from './submit-workflow-panel';

const argosay: Template = {
    name: 'argosay',
    inputs: {
        parameters: [
            {name: 'message', value: 'test'},
            {name: 'count', default: '1'}
        ]
    }
};

const whalesay: Template = {
    name: 'whalesay',
    inputs: {
        parameters: [{name: 'other', value: 'hidden'}]
    }
};

function renderPanel(entrypoint: string, templates: Template[], workflowParameters: Parameter[] = []) {
    const history = createMemoryHistory();
    return render(
        <Context.Provider
            value={{
                navigation: {goto: jest.fn()} as any,
                notifications: {} as any,
                popup: {} as any,
                history
            }}
        >
            <SubmitWorkflowPanel
                kind='WorkflowTemplate'
                namespace='argo'
                name='hello-world'
                entrypoint={entrypoint}
                templates={templates}
                workflowParameters={workflowParameters}
                history={history}
            />
        </Context.Provider>
    );
}

describe('SubmitWorkflowPanel', () => {
    it('shows the default entrypoint input parameters before the entrypoint is changed', () => {
        renderPanel('argosay', [argosay, whalesay]);

        expect(screen.getByText('message')).toBeInTheDocument();
        expect(screen.getByDisplayValue('test')).toBeInTheDocument();
        expect(screen.getByText('count')).toBeInTheDocument();
        expect(screen.getByDisplayValue('1')).toBeInTheDocument();
        expect(screen.queryByDisplayValue('hidden')).not.toBeInTheDocument();
        expect(screen.queryByText('No parameters')).not.toBeInTheDocument();
    });

    it('shows workflow arguments together with the default entrypoint parameters', () => {
        renderPanel('argosay', [argosay], [{name: 'wf-param', value: 'from-workflow'}]);

        expect(screen.getByDisplayValue('from-workflow')).toBeInTheDocument();
        expect(screen.getByDisplayValue('test')).toBeInTheDocument();
    });

    it('shows no template parameters when the entrypoint is unset', () => {
        renderPanel('', [argosay]);

        expect(screen.getByText('No parameters')).toBeInTheDocument();
        expect(screen.queryByDisplayValue('test')).not.toBeInTheDocument();
    });

    it('refreshes parameters when the entrypoint or templates change', () => {
        const view = renderPanel('argosay', [argosay, whalesay]);
        expect(screen.getByDisplayValue('test')).toBeInTheDocument();

        view.rerender(
            <Context.Provider
                value={{
                    navigation: {goto: jest.fn()} as any,
                    notifications: {} as any,
                    popup: {} as any,
                    history: createMemoryHistory()
                }}
            >
                <SubmitWorkflowPanel
                    kind='WorkflowTemplate'
                    namespace='argo'
                    name='hello-world'
                    entrypoint='whalesay'
                    templates={[argosay, whalesay]}
                    workflowParameters={[]}
                    history={createMemoryHistory()}
                />
            </Context.Provider>
        );

        expect(screen.getByDisplayValue('hidden')).toBeInTheDocument();
        expect(screen.queryByDisplayValue('test')).not.toBeInTheDocument();
    });

    it('shows one field per name when an entrypoint input matches a workflow argument', () => {
        renderPanel('argosay', [argosay], [{name: 'message', value: 'from-args'}]);

        expect(screen.getAllByText('message')).toHaveLength(1);
        expect(screen.getByDisplayValue('from-args')).toBeInTheDocument();
        expect(screen.queryByDisplayValue('test')).not.toBeInTheDocument();
        expect(screen.getByText('count')).toBeInTheDocument();
    });
});
