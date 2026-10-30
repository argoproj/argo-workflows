import {act, fireEvent, render, screen, waitFor} from '@testing-library/react';
import * as React from 'react';

import {request} from './api';
import {PagedRevisionDiff} from './paged-revision-diff';

jest.mock('./api', () => ({request: jest.fn()}));
const first = {
    left: 'r1',
    right: 'r2',
    changes: [{op: 'replace', path: '/a', before: 1, after: 2}],
    pagination: {total: 2, offset: 0, limit: 1, continue: 'next'},
    totals: {changes: 2, modelChanges: 0, sourceChanges: 0}
};
const second = {...first, changes: [{op: 'replace', path: '/b', before: 1, after: 2}], pagination: {total: 2, offset: 1, limit: 1, continue: ''}};
afterEach(() => jest.resetAllMocks());
it('reads exact comparison pages, limits rendered rows and allows return without mutations', async () => {
    (request as jest.Mock).mockResolvedValueOnce(second).mockResolvedValueOnce(first);
    render(<PagedRevisionDiff namespace='lab' comparison={first} />);
    fireEvent.click(screen.getByText('Next diff page'));
    await waitFor(() => expect(screen.getByText('/b')).toBeVisible());
    expect(screen.queryByText('/a')).toBeNull();
    expect(screen.getByText('Next diff page')).toBeDisabled();
    expect(request).toHaveBeenCalledWith('lab', 'diff', 'post', {left: 'r1', right: 'r2', limit: 1, continue: 'next'});
    fireEvent.click(screen.getByText('Previous diff page'));
    await waitFor(() => expect(screen.getByText('/a')).toBeVisible());
    expect(screen.queryByText('/b')).toBeNull();
});
it('retains the visible page after authorization errors and ignores an obsolete response', async () => {
    (request as jest.Mock).mockRejectedValueOnce(new Error('Forbidden'));
    const view = render(<PagedRevisionDiff namespace='lab' comparison={first} />);
    fireEvent.click(screen.getByText('Next diff page'));
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('Forbidden'));
    expect(screen.getByText('/a')).toBeVisible();
    let resolve: any;
    (request as jest.Mock).mockImplementationOnce(() => new Promise(r => (resolve = r)));
    fireEvent.click(screen.getByText('Next diff page'));
    view.rerender(<PagedRevisionDiff namespace='other' comparison={{...first, left: 'r3'}} />);
    await act(async () => resolve(second));
    expect(screen.queryByText('/b')).toBeNull();
    expect(screen.getByText('r3')).toBeVisible();
});
it('pages across execution layers while preserving the exact source, revision and definition snapshot', async () => {
    const comparison: any = {
        source: {workflow: 'saved', uid: 'uid'},
        parameters: first,
        layers: [{name: 'Form', available: true, total: 1, changes: []}],
        pagination: {...first.pagination, total: 3, snapshot: 'frozen-comparison'}
    };
    const later: any = {
        ...comparison,
        parameters: {...first, changes: []},
        layers: [{name: 'Form', available: true, total: 1, changes: [{op: 'replace', path: '/labels/count', before: 'old', after: 'new'}]}],
        pagination: {...comparison.pagination, offset: 2, continue: ''}
    };
    (request as jest.Mock).mockResolvedValueOnce(later).mockResolvedValueOnce(comparison);
    render(<PagedRevisionDiff namespace='lab' comparison={comparison} execution={true} />);
    expect(screen.getByText('Changes are on another page.')).toBeVisible();
    fireEvent.click(screen.getByText('Next diff page'));
    await waitFor(() => expect(screen.getByText('/labels/count')).toBeVisible());
    expect(screen.queryByText('/a')).toBeNull();
    expect(request).toHaveBeenCalledWith('lab', 'execution-diff', 'post', {source: comparison.source, revision: 'r2', snapshot: 'frozen-comparison', limit: 1, continue: 'next'});
    fireEvent.click(screen.getByText('Previous diff page'));
    await waitFor(() => expect(screen.getByText('/a')).toBeVisible());
    expect(request).toHaveBeenLastCalledWith('lab', 'execution-diff', 'post', {source: comparison.source, revision: 'r2', snapshot: 'frozen-comparison', limit: 1, continue: ''});
});
it('shows stale execution comparison errors without silently adopting new definitions', async () => {
    const comparison: any = {source: {workflow: 'saved', uid: 'uid'}, parameters: first, layers: [], pagination: {...first.pagination, snapshot: 'old'}};
    (request as jest.Mock).mockRejectedValueOnce(new Error('execution comparison changed; restart comparison'));
    render(<PagedRevisionDiff namespace='lab' comparison={comparison} execution={true} />);
    fireEvent.click(screen.getByText('Next diff page'));
    expect(await screen.findByRole('alert')).toHaveTextContent('restart comparison');
    expect(screen.getByText('/a')).toBeVisible();
    expect(request).toHaveBeenCalledTimes(1);
});
