import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import * as React from 'react';

import {request} from './api';
import {Parameterization} from './parameterization';

jest.mock('./api', () => ({request: jest.fn()}));
jest.mock('argo-ui/src/components/page/page', () => ({Page: ({children}: any) => <div>{children}</div>}));
jest.mock('./schema-form', () => ({
    SchemaForm: ({value}: any) => (
        <div>
            Structured form<output aria-label='Form values'>{JSON.stringify(value)}</output>
        </div>
    )
}));
jest.mock('../shared/components/suspense-monaco-editor', () => ({SuspenseMonacoEditor: () => <div />}));
const binding = {metadata: {name: 'selected'}, spec: {parameterTemplate: 'model', form: 'display'}};
const props = (namespace: string, search = '?binding=selected') => ({match: {params: {namespace}}, location: {search}}) as any;
beforeEach(() => (request as jest.Mock).mockReset());

it('opens the exact linked binding even when it is beyond the first listing page', async () => {
    (request as jest.Mock).mockImplementation(async (_ns, path) => {
        if (path === 'bindings') return {items: []};
        if (path === 'bindings/selected') return binding;
        if (path === 'models/model') return {spec: {schema: {type: 'object'}}};
        return {spec: {}};
    });
    render(<Parameterization {...props('lab')} />);
    await waitFor(() => expect(screen.getByLabelText('Workflow binding')).toHaveValue('selected'));
    expect(await screen.findByText('Structured form')).toBeVisible();
    expect(request).toHaveBeenCalledWith('lab', 'forms/display');
    expect((request as jest.Mock).mock.calls.some(([, path]) => path === 'submit')).toBe(false);
});

it('does not load a former namespace binding after the route changes', async () => {
    let resolveOld: (value: any) => void;
    (request as jest.Mock).mockImplementation(async (ns, path) => {
        if (ns === 'old' && path === 'bindings')
            return new Promise(resolve => {
                resolveOld = resolve;
            });
        if (path === 'bindings') return {items: []};
        if (path === 'bindings/selected') return binding;
        return {spec: {schema: {type: 'object'}}};
    });
    const view = render(<Parameterization {...props('old')} />);
    view.rerender(<Parameterization {...props('new', '')} />);
    await waitFor(() => expect(screen.getByLabelText('Workflow binding')).not.toBeDisabled());
    resolveOld({items: [binding]});
    await waitFor(() => expect(screen.getByLabelText('Workflow binding')).toHaveValue(''));
    expect(screen.queryByText('Structured form')).toBeNull();
});

it('loads execution values and the current CAS version, then saves without submitting', async () => {
    const revision = {metadata: {name: 'executed'}, spec: {template: 'model', values: {replicas: 2, enabled: false}}};
    const set = {metadata: {name: 'original-set', resourceVersion: 'current-cas'}, spec: {template: 'model', revision: 'newer', values: {replicas: 8}}};
    (request as jest.Mock).mockImplementation(async (_ns, path, method, body) => {
        if (path === 'bindings') return {items: []};
        if (path === 'bindings/selected') return binding;
        if (path.startsWith('edit-context/')) return {source: {workflow: 'original', uid: 'uid-one'}, binding: 'selected', revision, set};
        if (path === 'history/original-set?view=summary') return {items: [revision]};
        if (path === 'models/model') return {spec: {schema: {type: 'object'}}};
        if (path === 'sets/original-set' && method === 'put') return {...set, spec: {...set.spec, values: body.values, revision: 'edited'}};
        return {spec: {}};
    });
    render(<Parameterization {...props('lab', '?sourceWorkflow=original&sourceUID=uid-one')} />);
    await waitFor(() => expect(screen.getByLabelText('Parameter set')).toHaveValue('original-set'));
    expect(screen.getByLabelText('Form values')).toHaveTextContent('"replicas":2');
    expect(screen.getByLabelText('Form values')).toHaveTextContent('"enabled":false');
    expect(screen.getByLabelText('Workflow binding')).toBeDisabled();
    expect(screen.getByText(/Preview partial retry to keep this Workflow/)).toBeVisible();
    expect(request).toHaveBeenCalledWith('lab', 'diff', 'post', {left: 'executed', right: 'newer'});
    fireEvent.click(screen.getByRole('button', {name: 'Save revision'}));
    await waitFor(() =>
        expect(request).toHaveBeenCalledWith(
            'lab',
            'sets/original-set',
            'put',
            expect.objectContaining({resourceVersion: 'current-cas', values: {replicas: 2, enabled: false}, restoredFrom: 'executed'})
        )
    );
    expect((request as jest.Mock).mock.calls.some(([, path]) => path === 'submit' || path === 'retry')).toBe(false);
});

it('fails closed when an execution link references a different binding', async () => {
    (request as jest.Mock).mockImplementation(async (_ns, path) => {
        if (path === 'bindings') return {items: []};
        if (path.startsWith('edit-context/')) return {binding: 'different'};
        return {};
    });
    render(<Parameterization {...props('lab', '?binding=selected&sourceWorkflow=old&sourceUID=uid')} />);
    expect(await screen.findByRole('alert')).toHaveTextContent('linked binding differs');
    expect(screen.queryByText('Structured form')).toBeNull();
    expect(screen.queryByRole('button', {name: 'Save revision'})).toBeNull();
});

it('requires an explicit impact preview before applying a new revision to the same workflow', async () => {
    (request as jest.Mock).mockImplementation(async (_ns, path) => {
        if (path === 'bindings') return {items: [binding]};
        if (path === 'bindings/selected') return binding;
        if (path.startsWith('edit-context/'))
            return {
                source: {workflow: 'original', uid: 'uid'},
                binding: 'selected',
                revision: {metadata: {name: 'old'}, spec: {values: {count: 1}, template: 'model'}},
                set: {metadata: {name: 'set', resourceVersion: '3'}, spec: {revision: 'new', template: 'model'}}
            };
        if (path === 'models/model') return {spec: {schema: {type: 'object'}}};
        if (path === 'retry-preview') return {revision: 'new', resourceVersion: '9', digest: 'plan', affectedTasks: ['changed'], retainedNodes: {approved: 'hash'}};
        if (path === 'retry') return {metadata: {name: 'original', uid: 'uid'}};
        if (path.startsWith('history/')) return {items: []};
        return {spec: {}};
    });
    render(<Parameterization {...props('lab', '?sourceWorkflow=original&sourceUID=uid')} />);
    await waitFor(() => expect(screen.getByRole('button', {name: 'Preview partial retry'})).not.toBeDisabled());
    fireEvent.click(screen.getByRole('button', {name: 'Preview partial retry'}));
    expect(await screen.findByText(/Affected tasks: changed/)).toBeVisible();
    expect((request as jest.Mock).mock.calls.some(([, path]) => path === 'retry')).toBe(false);
    fireEvent.click(screen.getByRole('button', {name: 'Apply revision and retry this Workflow'}));
    await waitFor(() =>
        expect(request).toHaveBeenCalledWith(
            'lab',
            'retry',
            'post',
            expect.objectContaining({workflow: 'original', uid: 'uid', revision: 'new', resourceVersion: '9', planDigest: 'plan'})
        )
    );
    expect((request as jest.Mock).mock.calls.some(([, path]) => path === 'submit')).toBe(false);
});

it('loads a historical body only on restore and preserves falsy input values', async () => {
    const revision = {metadata: {name: 'executed'}, spec: {template: 'model', values: {replicas: 2}}};
    const set = {metadata: {name: 'original-set', resourceVersion: 'cas'}, spec: {template: 'model', revision: 'executed'}};
    (request as jest.Mock).mockImplementation(async (_ns, path) => {
        if (path === 'bindings') return {items: [binding]};
        if (path === 'bindings/selected') return binding;
        if (path.startsWith('edit-context/')) return {source: {workflow: 'original', uid: 'uid'}, binding: 'selected', revision, set};
        if (path === 'models/model') return {spec: {schema: {type: 'object'}}};
        if (path === 'history/original-set?view=summary') return {items: [{metadata: {name: 'older', uid: 'old-uid'}, spec: {actor: 'tester', message: 'old revision'}}]};
        if (path === 'revisions/older') return {metadata: {name: 'older', uid: 'old-uid'}, spec: {set: 'original-set', values: {enabled: false, count: 0, list: []}}};
        return {spec: {}};
    });
    render(<Parameterization {...props('lab', '?sourceWorkflow=original&sourceUID=uid')} />);
    const restore = await screen.findByRole('button', {name: 'Restore into editor'});
    expect((request as jest.Mock).mock.calls.some(([, path]) => path === 'revisions/older')).toBe(false);
    fireEvent.click(restore);
    await waitFor(() => expect(screen.getByLabelText('Form values')).toHaveTextContent('"enabled":false'));
    expect(screen.getByLabelText('Form values')).toHaveTextContent('"count":0');
    expect(screen.getByLabelText('Form values')).toHaveTextContent('"list":[]');
    expect((request as jest.Mock).mock.calls.some(([, path]) => path === 'submit' || path === 'retry' || path.startsWith('sets/'))).toBe(false);
});

it('reloads expired history without saving, retrying or mixing old pages', async () => {
    let loads = 0;
    (request as jest.Mock).mockImplementation(async (_ns, path) => {
        if (path === 'bindings') return {items: [binding]};
        if (path === 'bindings/selected') return binding;
        if (path.startsWith('edit-context/'))
            return {
                source: {workflow: 'original', uid: 'uid'},
                binding: 'selected',
                revision: {metadata: {name: 'old'}, spec: {template: 'model', values: {count: 1}}},
                set: {metadata: {name: 'set'}, spec: {template: 'model', revision: 'old'}}
            };
        if (path === 'models/model') return {spec: {schema: {type: 'object'}}};
        if (path === 'history/set?view=summary') {
            loads++;
            return {metadata: {continue: loads === 1 ? 'expired' : ''}, items: [{metadata: {name: loads === 1 ? 'previous-page' : 'new-head'}, spec: {actor: 'tester'}}]};
        }
        if (path.includes('continue=expired')) throw new Error('history cursor expired or changed; reload history from the first page');
        return {spec: {}};
    });
    render(<Parameterization {...props('lab', '?sourceWorkflow=original&sourceUID=uid')} />);
    fireEvent.click(await screen.findByRole('button', {name: 'Load older revisions'}));
    expect(await screen.findByRole('alert')).toHaveTextContent('reload history');
    fireEvent.click(screen.getByRole('button', {name: 'Reload history'}));
    expect(await screen.findByText('new-head')).toBeVisible();
    expect(screen.queryByText('previous-page')).toBeNull();
    expect((request as jest.Mock).mock.calls.some(([, path]) => path === 'submit' || path === 'retry' || path.startsWith('sets/'))).toBe(false);
});
