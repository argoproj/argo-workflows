import {act, fireEvent, render, screen, waitFor} from '@testing-library/react';
import {createMemoryHistory} from 'history';
import * as React from 'react';
import {Router} from 'react-router-dom';

import {Context} from '../shared/context';
import {exampleClusterWorkflowTemplate} from '../shared/examples';
import {services} from '../shared/services';
import {ClusterWorkflowTemplateList} from './cluster-workflow-template-list';

jest.mock('./cluster-workflow-template-creator', () => ({ClusterWorkflowTemplateCreator: (): null => null}));

function response(name: string, nextOffset?: string) {
    const template = exampleClusterWorkflowTemplate();
    template.metadata.name = name;
    return {metadata: {continue: nextOffset}, items: [template]};
}

function deferred<T>() {
    let resolve!: (value: T) => void;
    const promise = new Promise<T>(r => (resolve = r));
    return {promise, resolve};
}

function renderList(search = '') {
    const history = createMemoryHistory();
    history.push(`/cluster-workflow-templates${search}`);
    render(
        <Router history={history}>
            <Context.Provider value={{navigation: {goto: jest.fn()}} as any}>
                <ClusterWorkflowTemplateList history={history} location={history.location} match={{} as any} />
            </Context.Provider>
        </Router>
    );
}

describe('ClusterWorkflowTemplateList', () => {
    afterEach(() => {
        jest.restoreAllMocks();
        localStorage.clear();
    });

    it('loads pagination from the URL and requests the next page', async () => {
        const list = jest
            .spyOn(services.clusterWorkflowTemplate, 'list')
            .mockResolvedValueOnce(response('example-template', 'next-token') as any)
            .mockResolvedValueOnce(response('example-template') as any);
        renderList('?offset=current-token&limit=5');

        await waitFor(() => expect(list).toHaveBeenCalledWith({offset: 'current-token', limit: 5}));
        const nextPage = await screen.findByRole('button', {name: /Next page/});
        expect(nextPage).toBeEnabled();

        fireEvent.click(nextPage);

        await waitFor(() => expect(list).toHaveBeenNthCalledWith(2, {offset: 'next-token', limit: 5}));
    });

    it('preserves an explicit all-results limit from the URL', async () => {
        const list = jest.spyOn(services.clusterWorkflowTemplate, 'list').mockResolvedValue(response('example-template') as any);

        renderList('?limit=0');

        await waitFor(() => expect(list).toHaveBeenCalledWith({offset: null, limit: 0}));
    });

    it('loads and updates the stored page size', async () => {
        localStorage.setItem('ClusterWorkflowTemplateListOptions/paginationLimit', '10');
        const list = jest.spyOn(services.clusterWorkflowTemplate, 'list').mockResolvedValue(response('example-template') as any);
        renderList();

        await waitFor(() => expect(list).toHaveBeenCalledWith({offset: null, limit: 10}));
        fireEvent.change(await screen.findByRole('combobox'), {target: {value: '20'}});

        await waitFor(() => expect(list).toHaveBeenLastCalledWith({limit: 20, offset: undefined}));
        expect(localStorage.getItem('ClusterWorkflowTemplateListOptions/paginationLimit')).toBe('20');
    });

    it('ignores a stale response after the page size changes', async () => {
        const stale = deferred<any>();
        const latest = deferred<any>();
        const list = jest
            .spyOn(services.clusterWorkflowTemplate, 'list')
            .mockResolvedValueOnce(response('first-page', 'next-token') as any)
            .mockReturnValueOnce(stale.promise)
            .mockReturnValueOnce(latest.promise);
        renderList('?limit=5');

        fireEvent.click(await screen.findByRole('button', {name: /Next page/}));
        await waitFor(() => expect(list).toHaveBeenNthCalledWith(2, {offset: 'next-token', limit: 5}));
        fireEvent.change(screen.getByRole('combobox'), {target: {value: '20'}});
        await waitFor(() => expect(list).toHaveBeenNthCalledWith(3, {limit: 20, offset: undefined}));

        await act(async () => latest.resolve(response('latest-page')));
        expect(await screen.findByText('latest-page')).toBeInTheDocument();
        await act(async () => stale.resolve(response('stale-page')));

        expect(screen.getByText('latest-page')).toBeInTheDocument();
        expect(screen.queryByText('stale-page')).not.toBeInTheDocument();
    });
});
