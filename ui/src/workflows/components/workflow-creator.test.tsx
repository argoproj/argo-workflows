import {fireEvent, render, screen, waitFor, within} from '@testing-library/react';
import {createMemoryHistory} from 'history';
import * as React from 'react';

import {Workflow} from '../../shared/models';
import {services} from '../../shared/services';
import {WorkflowCreator} from './workflow-creator';

jest.mock('../../shared/services', () => ({
    services: {
        workflowTemplate: {list: jest.fn()},
        workflows: {create: jest.fn()}
    }
}));

// Keep the editor's parsing and synchronization behavior, replacing only Monaco's browser surface.
jest.mock('../../shared/components/suspense-monaco-editor', () => ({
    SuspenseMonacoEditor: React.forwardRef(function MockMonacoEditor(props: {defaultValue: string; onChange: (value: string) => void}, ref: React.ForwardedRef<unknown>) {
        const [text, setText] = React.useState(props.defaultValue || '');
        const currentText = React.useRef(text);
        currentText.current = text;
        React.useImperativeHandle(
            ref,
            () => ({
                editor: {
                    getValue: () => currentText.current,
                    setValue: (value: string) => setText(value)
                }
            }),
            []
        );
        return (
            <textarea
                aria-label='Workflow manifest'
                value={text}
                onChange={event => {
                    setText(event.target.value);
                    props.onChange(event.target.value);
                }}
            />
        );
    })
}));

jest.mock('../../shared/components/graph/graph-panel', () => ({
    GraphPanel: () => <div>Workflow graph</div>
}));

const restoredManifest = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: restored-workflow
  namespace: restored-namespace
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        name: main
        image: busybox
        command: [echo, restored]
`;

const restoredWorkflow: Workflow = {
    apiVersion: 'argoproj.io/v1alpha1',
    kind: 'Workflow',
    metadata: {name: 'restored-workflow', namespace: 'restored-namespace'},
    spec: {
        entrypoint: 'main',
        templates: [{name: 'main', container: {name: 'main', image: 'busybox', command: ['echo', 'restored']}}]
    }
};

async function openFullEditor() {
    const onCreate = jest.fn();
    render(<WorkflowCreator namespace='argo' onCreate={onCreate} history={createMemoryHistory()} />);
    await waitFor(() => expect(services.workflowTemplate.list).toHaveBeenCalledWith('argo', []));
    fireEvent.click(screen.getByText('Edit using full workflow options'));
    const editor = await screen.findByRole('textbox', {name: 'Workflow manifest'});
    return {editor, onCreate};
}

describe('WorkflowCreator manifest editing', () => {
    beforeEach(() => {
        localStorage.clear();
        jest.clearAllMocks();
        (services.workflowTemplate.list as jest.Mock).mockResolvedValue({items: []});
        (services.workflows.create as jest.Mock).mockResolvedValue(restoredWorkflow);
    });

    test.each([
        ['empty', ''],
        ['whitespace-only', ' \n  '],
        ['comment-only', '# Draft a new workflow here\n'],
        ['null', 'null\n'],
        ['empty YAML document', '---\n']
    ])('keeps the %s manifest editable without submitting the previous workflow', async (_, manifest) => {
        const {editor, onCreate} = await openFullEditor();
        fireEvent.change(editor, {target: {value: manifest}});

        expect(screen.getByRole('textbox', {name: 'Workflow manifest'})).toBe(editor);
        expect(editor).toHaveValue(manifest);
        const create = screen.getByRole('button', {name: 'Create'});
        expect(create).toBeDisabled();
        fireEvent.click(create);
        expect(services.workflows.create).not.toHaveBeenCalled();
        expect(onCreate).not.toHaveBeenCalled();
    });

    test('submits the newly entered workflow after clearing and restoring the manifest', async () => {
        const {editor, onCreate} = await openFullEditor();
        fireEvent.change(editor, {target: {value: ''}});
        expect(screen.getByRole('button', {name: 'Create'})).toBeDisabled();

        fireEvent.change(screen.getByRole('textbox', {name: 'Workflow manifest'}), {target: {value: restoredManifest}});
        expect(screen.getByRole('button', {name: 'Create'})).toBeEnabled();
        fireEvent.click(screen.getByRole('button', {name: 'Create'}));

        await waitFor(() => expect(onCreate).toHaveBeenCalledWith(restoredWorkflow));
        expect(services.workflows.create).toHaveBeenCalledTimes(1);
        expect(services.workflows.create).toHaveBeenCalledWith(restoredWorkflow, 'restored-namespace');
    });

    test.each([false, true])('submits a workflow template reference without an explicit entrypoint (clusterScope: %p)', async clusterScope => {
        const workflow: Workflow = {
            metadata: {generateName: 'from-template-', namespace: 'argo'},
            spec: {workflowTemplateRef: {name: 'example-template', clusterScope}}
        };
        (services.workflows.create as jest.Mock).mockResolvedValue(workflow);
        const {editor, onCreate} = await openFullEditor();
        fireEvent.change(editor, {target: {value: ''}});
        fireEvent.change(editor, {target: {value: JSON.stringify(workflow)}});

        expect(screen.getByRole('button', {name: 'Create'})).toBeEnabled();
        fireEvent.click(screen.getByRole('button', {name: 'Create'}));

        await waitFor(() => expect(onCreate).toHaveBeenCalledWith(workflow));
        expect(services.workflows.create).toHaveBeenCalledTimes(1);
        expect(services.workflows.create).toHaveBeenCalledWith(workflow, 'argo');
    });

    test('keeps Create disabled after adding parameters to a cleared manifest', async () => {
        const {editor, onCreate} = await openFullEditor();
        fireEvent.change(editor, {target: {value: ''}});
        fireEvent.click(screen.getByText('Parameters', {selector: 'a'}));
        fireEvent.change(screen.getByPlaceholderText('Name...'), {target: {value: 'message'}});
        fireEvent.change(screen.getByPlaceholderText('Value...'), {target: {value: 'hello'}});
        fireEvent.keyPress(screen.getByPlaceholderText('Value...'), {key: 'Enter', charCode: 13});

        expect(screen.getByText('message')).toBeInTheDocument();
        expect(screen.getByText('hello')).toBeInTheDocument();
        expect(screen.getByRole('button', {name: 'Create'})).toBeDisabled();
        fireEvent.click(screen.getByRole('button', {name: 'Create'}));
        expect(services.workflows.create).not.toHaveBeenCalled();
        expect(onCreate).not.toHaveBeenCalled();
    });

    test('keeps Create disabled after editing metadata in a cleared manifest', async () => {
        const {editor, onCreate} = await openFullEditor();
        fireEvent.change(editor, {target: {value: ''}});
        fireEvent.click(screen.getByText('MetaData', {selector: 'a'}));
        const nameInput = within(screen.getByText('Name', {exact: true}).closest('.row') as HTMLElement).getByRole('textbox');
        fireEvent.change(nameInput, {target: {value: 'draft-workflow'}});

        expect(nameInput).toHaveValue('draft-workflow');
        expect(screen.getByRole('button', {name: 'Create'})).toBeDisabled();
        fireEvent.click(screen.getByRole('button', {name: 'Create'}));
        expect(services.workflows.create).not.toHaveBeenCalled();
        expect(onCreate).not.toHaveBeenCalled();
    });

    test.each(['', '{}', 'spec: {}\n', 'metadata: {}\n', 'spec:\n  workflowTemplateRef: {}\n', 'spec:\n  workflowTemplateRef:\n    name: ""\n'])(
        'keeps an incomplete manifest editable without allowing submission: %p',
        async manifest => {
            const {editor} = await openFullEditor();
            fireEvent.change(editor, {target: {value: manifest}});

            fireEvent.click(screen.getByText('Parameters', {selector: 'a'}));
            expect(screen.getByRole('heading', {name: 'Parameters'})).toBeInTheDocument();
            fireEvent.click(screen.getByText('MetaData', {selector: 'a'}));
            expect(screen.getByText('Generate Name')).toBeInTheDocument();
            fireEvent.click(screen.getByText('Graph', {selector: 'a'}));
            fireEvent.click(screen.getByText('Manifest', {selector: 'a'}));

            expect(screen.getByRole('textbox', {name: 'Workflow manifest'})).toHaveValue(manifest);
            expect(screen.getByRole('button', {name: 'Create'})).toBeDisabled();
            fireEvent.click(screen.getByRole('button', {name: 'Create'}));
            expect(services.workflows.create).not.toHaveBeenCalled();
        }
    );
});
