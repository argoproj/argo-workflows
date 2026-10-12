import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import React from 'react';

import {request} from './api';
import {WaitingInput} from './node-inputs';

jest.mock('./api', () => ({request: jest.fn()}));
const workflow: any = {metadata: {name: 'waiting', namespace: 'lab', uid: 'wf-uid'}};
const item = {
    nodeId: 'input-node',
    name: 'Authorize',
    contractDigest: 'frozen',
    contract: {
        model: {schema: {type: 'object', properties: {credential: {type: 'object'}}}, sensitivePaths: ['/credential']},
        form: {fields: [{path: '/credential', widget: 'secret'}]}
    }
};
beforeEach(() => {
    (request as jest.Mock).mockReset();
    Object.defineProperty(global, 'crypto', {value: {randomUUID: () => 'random-test-request-id'}, configurable: true});
});
it('uses frozen node identity for the secret endpoint and submits only a reference', async () => {
    const ref = {secretRef: {name: 'immutable', key: 'value'}};
    let complete: (v: any) => void;
    (request as jest.Mock).mockImplementation((_ns, path, method) => {
        if (path === 'secrets')
            return new Promise(resolve => {
                complete = resolve;
            });
        if (method !== 'post') return Promise.resolve({uid: 'wf-uid', resourceVersion: '7', items: [item]});
        return Promise.resolve({});
    });
    render(<WaitingInput workflow={workflow} item={item} encryptedSecretInput />);
    fireEvent.change(screen.getByLabelText('New secret /credential'), {target: {value: 'synthetic-secret'}});
    fireEvent.click(screen.getByText('Store new encrypted secret version'));
    expect(screen.getByText('Validate input and continue this node')).toBeDisabled();
    expect(request).toHaveBeenCalledWith('lab', 'secrets', 'post', {
        input: {workflow: 'waiting', uid: 'wf-uid', nodeId: 'input-node', contractDigest: 'frozen'},
        path: '/credential',
        requestId: 'random-test-request-id',
        value: 'synthetic-secret'
    });
    complete(ref);
    await waitFor(() => expect(screen.getByText('Validate input and continue this node')).toBeEnabled());
    expect(screen.getByLabelText('New secret /credential')).toHaveValue('');
    fireEvent.click(screen.getByText('Validate input and continue this node'));
    await screen.findByText('Input accepted for Authorize.');
    const submitted = (request as jest.Mock).mock.calls.find(c => c[1] === 'inputs/waiting' && c[2] === 'post')[3];
    expect(submitted.values).toEqual({credential: ref});
    expect(JSON.stringify(submitted)).not.toContain('synthetic-secret');
});

it('does not submit stale UI values after the Workflow identity changes', async () => {
    (request as jest.Mock).mockResolvedValue({uid: 'replacement-uid', resourceVersion: '8', items: [item]});
    render(<WaitingInput workflow={workflow} item={item} />);
    fireEvent.click(screen.getByText('Validate input and continue this node'));
    await screen.findByText('This workflow or input contract changed; reopen the pending input.');
    expect((request as jest.Mock).mock.calls.filter(call => call[2] === 'post')).toHaveLength(0);
});
