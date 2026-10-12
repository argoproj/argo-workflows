import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import React from 'react';

import {request} from './api';
import {NodeResults, ResultView} from './node-results';

jest.mock('./api', () => ({request: jest.fn()}));
const workflow: any = {metadata: {name: 'example', namespace: 'lab', uid: 'uid', annotations: {'parameters.workflows.example.io/submission': 'frozen'}}};
const node: any = {id: 'node', startedAt: '2026-10-14T00:00:00Z', phase: 'Succeeded'};
const view: any = {
    type: 'table',
    columns: [
        {title: 'State', path: '/state'},
        {title: 'Link', path: '/url', format: 'link'}
    ]
};
beforeEach(() => (request as jest.Mock).mockReset());

it('renders data-driven reports and rejects executable or credential-bearing links', () => {
    render(
        <ResultView
            view={view}
            value={[
                {state: 'Ready', url: 'https://example.org/report'},
                {state: 'Unknown', url: 'javascript:alert(1)'},
                {state: 'Failed', url: 'https://user:secret@example.org/'}
            ]}
        />
    );
    expect(screen.getByText('Ready')).toBeVisible();
    expect(screen.getAllByRole('link')).toHaveLength(1);
    expect(screen.getByRole('link')).toHaveAttribute('href', 'https://example.org/report');
});

it('refreshes only the bound read endpoint and retains results on a transport failure', async () => {
    (request as jest.Mock)
        .mockResolvedValueOnce({
            uid: 'uid',
            nodeId: 'node',
            items: [{name: 'report', title: 'Report', scope: 'LiveObservation', available: true, value: [{state: 'Ready'}], views: [view]}]
        })
        .mockRejectedValueOnce(new Error('Transport unavailable'));
    render(<NodeResults workflow={workflow} node={node} />);
    await screen.findByText('Ready');
    fireEvent.click(screen.getByText('Refresh results'));
    await screen.findByText(/Transport unavailable/);
    expect(screen.getByText('Ready')).toBeVisible();
    expect(request).toHaveBeenCalledTimes(2);
    for (const call of (request as jest.Mock).mock.calls) {
        expect(call[1]).toBe('node-results/example?uid=uid&nodeId=node');
        expect(call[2]).toBeUndefined();
    }
});

it('discards stale responses after switching invocation and never queries ordinary workflows', async () => {
    let finish: (v: any) => void;
    (request as jest.Mock)
        .mockImplementationOnce(
            () =>
                new Promise(resolve => {
                    finish = resolve;
                })
        )
        .mockResolvedValue({uid: 'uid', nodeId: 'other', items: []});
    const page = render(<NodeResults workflow={workflow} node={node} />);
    page.rerender(<NodeResults workflow={workflow} node={{...node, id: 'other'}} />);
    await waitFor(() => expect(request).toHaveBeenCalledTimes(2));
    finish({uid: 'uid', nodeId: 'node', items: [{name: 'old', title: 'Wrong result', available: true, views: [view], value: []}]});
    await waitFor(() => expect(screen.queryByText('Wrong result')).not.toBeInTheDocument());
    page.rerender(<NodeResults workflow={{...workflow, metadata: {...workflow.metadata, annotations: {}}}} node={node} />);
    expect(request).toHaveBeenCalledTimes(2);
});

it('paginates large tables without losing the result scope', () => {
    render(<ResultView view={view} value={Array.from({length: 70}, (_, i) => ({state: 'row-' + i}))} />);
    expect(screen.queryByText('row-69')).not.toBeInTheDocument();
    fireEvent.click(screen.getByText('Next rows'));
    expect(screen.getByText('row-69')).toBeVisible();
    expect(screen.queryByText('row-0')).not.toBeInTheDocument();
});
