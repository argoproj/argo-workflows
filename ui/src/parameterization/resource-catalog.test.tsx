import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import React from 'react';

import {request} from './api';
import {ResourceCatalog} from './resource-catalog';

jest.mock('./api', () => ({request: jest.fn()}));
jest.mock('../shared/components/suspense-monaco-editor', () => ({SuspenseMonacoEditor: ({value}: {value: string}) => <pre data-testid='resource-yaml'>{value}</pre>}));
const types = [
    {resource: 'parametertemplates', kind: 'ParameterTemplate'},
    {resource: 'workflowparameterbindings', kind: 'WorkflowParameterBinding'}
];
beforeEach(() => (request as jest.Mock).mockReset());
it('shows the actual binding references, paginates, and clears a stale resource on type changes', async () => {
    (request as jest.Mock).mockImplementation(async (_ns, path) => {
        if (path === 'catalog') return types;
        if (path === 'catalog/parametertemplates?view=metadata') return {items: [{kind: 'ParameterTemplate', metadata: {name: 'typed', namespace: 'lab', resourceVersion: '2'}}]};
        if (path === 'catalog/workflowparameterbindings?view=metadata')
            return {
                metadata: {continue: 'cursor'},
                items: [
                    {kind: 'WorkflowParameterBinding', metadata: {name: 'bind', namespace: 'lab'}, spec: {parameterTemplate: 'typed', form: 'display', workflowTemplate: 'native'}}
                ]
            };
        if (path === 'catalog/parametertemplates?name=typed') return {kind: 'ParameterTemplate', metadata: {name: 'typed'}, spec: {schema: {type: 'object'}}};
        if (path === 'catalog/workflowparameterbindings?name=bind')
            return {kind: 'WorkflowParameterBinding', metadata: {name: 'bind'}, spec: {parameterTemplate: 'typed', form: 'display', workflowTemplate: 'native'}};
        return {items: [{kind: 'WorkflowParameterBinding', metadata: {name: 'second', namespace: 'lab'}}]};
    });
    render(<ResourceCatalog namespace='lab' />);
    fireEvent.click(await screen.findByRole('button', {name: 'typed'}));
    expect(await screen.findByTestId('resource-yaml')).toHaveTextContent('ParameterTemplate');
    fireEvent.change(screen.getByLabelText('Parameter resource type'), {target: {value: 'workflowparameterbindings'}});
    await waitFor(() => expect(screen.queryByTestId('resource-yaml')).toBeNull());
    fireEvent.click(await screen.findByRole('button', {name: 'bind'}));
    expect(await screen.findByTestId('resource-yaml')).toHaveTextContent('workflowTemplate: native');
    expect(screen.getByTestId('resource-yaml')).toHaveTextContent('parameterTemplate: typed');
    fireEvent.click(screen.getByText('Load more resources'));
    await screen.findByRole('button', {name: 'second'});
    expect(request).toHaveBeenCalledWith('lab', 'catalog/workflowparameterbindings?view=metadata&continue=cursor');
});
it('does not display an old namespace response after navigation', async () => {
    let oldResult: (value: any) => void;
    (request as jest.Mock).mockImplementation(async (ns, path) => {
        if (path === 'catalog') return types;
        if (ns === 'old')
            return new Promise(resolve => {
                oldResult = resolve;
            });
        return {items: [{kind: 'ParameterTemplate', metadata: {name: 'new-scope', namespace: 'new'}}]};
    });
    const view = render(<ResourceCatalog namespace='old' />);
    view.rerender(<ResourceCatalog namespace='new' />);
    await screen.findByRole('button', {name: 'new-scope'});
    oldResult({items: [{metadata: {name: 'old-scope'}}]});
    await waitFor(() => expect(screen.queryByRole('button', {name: 'old-scope'})).toBeNull());
});

it('reads only the selected document and rejects replaced resources', async () => {
    (request as jest.Mock).mockImplementation(async (_ns, path) => {
        if (path === 'catalog') return types;
        if (path.includes('?name=')) return {kind: 'ParameterTemplate', metadata: {name: 'typed', uid: 'replacement'}, spec: {schema: {type: 'object'}}};
        return {items: [{metadata: {name: 'typed', uid: 'original'}}]};
    });
    render(<ResourceCatalog namespace='lab' />);
    const select = await screen.findByRole('button', {name: 'typed'});
    expect((request as jest.Mock).mock.calls.some(([, path]) => path.includes('?name='))).toBe(false);
    fireEvent.click(select);
    expect(await screen.findByRole('alert')).toHaveTextContent('Resource was replaced');
    expect(screen.queryByTestId('resource-yaml')).toBeNull();
});
it('ignores a slow document response after selecting another resource', async () => {
    let first: (value: any) => void;
    (request as jest.Mock).mockImplementation(async (_ns, path) => {
        if (path === 'catalog') return types;
        if (path.endsWith('?name=first'))
            return new Promise(resolve => {
                first = resolve;
            });
        if (path.endsWith('?name=second')) return {kind: 'ParameterTemplate', metadata: {name: 'second'}, spec: {title: 'latest selection'}};
        return {items: [{metadata: {name: 'first'}}, {metadata: {name: 'second'}}]};
    });
    render(<ResourceCatalog namespace='lab' />);
    fireEvent.click(await screen.findByRole('button', {name: 'first'}));
    fireEvent.click(screen.getByRole('button', {name: 'second'}));
    expect(await screen.findByTestId('resource-yaml')).toHaveTextContent('latest selection');
    first({metadata: {name: 'first'}, spec: {title: 'stale selection'}});
    await waitFor(() => expect(screen.getByTestId('resource-yaml')).not.toHaveTextContent('stale selection'));
});
