import {fireEvent, render, screen} from '@testing-library/react';
import * as React from 'react';

import {Workflow} from '../../../shared/models';
import {workflowHasOutputs, WorkflowOutputsModal} from './workflow-outputs-modal';

function workflow(status: Partial<Workflow['status']>): Workflow {
    return {
        metadata: {name: 'my-wf', namespace: 'default'},
        spec: {},
        status: status as Workflow['status']
    };
}

describe('workflowHasOutputs', () => {
    it('is false without outputs', () => {
        expect(workflowHasOutputs(workflow({}))).toBe(false);
        expect(workflowHasOutputs(workflow({outputs: {}}))).toBe(false);
    });

    it('is true with global outputs', () => {
        expect(workflowHasOutputs(workflow({outputs: {parameters: [{name: 'p', value: 'v'}]}}))).toBe(true);
    });

    it('is true with entrypoint node outputs', () => {
        expect(workflowHasOutputs(workflow({nodes: {'my-wf': {outputs: {result: 'hello'}} as Workflow['status']['nodes'][string]}}))).toBe(true);
    });
});

describe('WorkflowOutputsModal', () => {
    it('renders global and entrypoint outputs and can be dismissed', () => {
        const dismiss = jest.fn();
        render(
            <WorkflowOutputsModal
                workflow={workflow({
                    outputs: {parameters: [{name: 'global-param', value: 'global-value'}], artifacts: [{name: 'global-art'}]},
                    nodes: {'my-wf': {outputs: {result: 'hello world', exitCode: 0}} as Workflow['status']['nodes'][string]}
                })}
                dismiss={dismiss}
            />
        );

        expect(screen.getByText('Global Outputs')).toBeInTheDocument();
        expect(screen.getByText('global-param')).toBeInTheDocument();
        expect(screen.getByText('global-value')).toBeInTheDocument();
        expect(screen.getByText('global-art')).toBeInTheDocument();
        expect(screen.getByText('Entrypoint Outputs')).toBeInTheDocument();
        expect(screen.getByText('hello world')).toBeInTheDocument();

        fireEvent.click(screen.getByText('×'));
        expect(dismiss).toHaveBeenCalled();
    });

    it('omits empty sections', () => {
        render(<WorkflowOutputsModal workflow={workflow({outputs: {parameters: [{name: 'p', value: 'v'}]}})} dismiss={() => {}} />);
        expect(screen.queryByText('Entrypoint Outputs')).not.toBeInTheDocument();
    });
});
