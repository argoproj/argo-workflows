import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import * as React from 'react';

import {request} from './api';
import {ExecutionParameters} from './execution-parameters';

jest.mock('./api', () => ({request: jest.fn()}));
jest.mock('./node-inputs', () => ({NodeInputs: (): null => null}));
const mocked = request as jest.Mock;
const workflow = {metadata: {name: 'saved', namespace: 'lab', uid: 'wf-uid', annotations: {'parameters.workflows.example.io/revision': 'r1'}}} as any;
afterEach(() => mocked.mockReset());
it('compares actual execution identity against the draft and renders layered changes without submitting', async () => {
    mocked.mockImplementation((_ns: string, path: string) => {
        if (path === 'revisions/r1') return Promise.resolve({spec: {set: 'set', values: {count: 2}, sources: {}}});
        if (path === 'sets/set') return Promise.resolve({spec: {revision: 'r2'}});
        return Promise.resolve({
            notices: ['Comparison is read-only.'],
            parameters: {left: 'r1', right: 'r2'},
            layers: [{name: 'Action binding', available: true, changes: [{op: 'replace', path: '/implementation', before: 'old', after: 'new'}]}]
        });
    });
    render(<ExecutionParameters workflow={workflow} />);
    fireEvent.click(await screen.findByText('Compare execution with current draft and definitions'));
    await waitFor(() => expect(mocked).toHaveBeenCalledWith('lab', 'execution-diff', 'post', {source: {workflow: 'saved', uid: 'wf-uid'}, revision: 'r2'}));
    expect(await screen.findByText('Action binding (1/1)')).toBeInTheDocument();
    expect(screen.getByText('Comparison is read-only.')).toBeInTheDocument();
    expect(mocked.mock.calls.map(call => call[1])).toEqual(['revisions/r1', 'sets/set', 'execution-diff']);
});
