import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import {createMemoryHistory} from 'history';
import * as React from 'react';
import {Router} from 'react-router-dom';

import {Context} from '../shared/context';
import {exampleClusterWorkflowTemplate} from '../shared/examples';
import {services} from '../shared/services';
import {ClusterWorkflowTemplateList} from './cluster-workflow-template-list';

jest.mock('./cluster-workflow-template-creator', () => ({ClusterWorkflowTemplateCreator: (): null => null}));

describe('ClusterWorkflowTemplateList', () => {
    afterEach(() => {
        jest.restoreAllMocks();
        localStorage.clear();
    });

    it('loads pagination from the URL and requests the next page', async () => {
        const template = exampleClusterWorkflowTemplate();
        template.metadata.name = 'example-template';
        const list = jest
            .spyOn(services.clusterWorkflowTemplate, 'list')
            .mockResolvedValueOnce({metadata: {continue: 'next-token'}, items: [template]} as any)
            .mockResolvedValueOnce({metadata: {}, items: [template]} as any);
        const history = createMemoryHistory();
        history.push('/cluster-workflow-templates?offset=current-token&limit=5');

        render(
            <Router history={history}>
                <Context.Provider value={{navigation: {goto: jest.fn()}} as any}>
                    <ClusterWorkflowTemplateList history={history} location={history.location} match={{} as any} />
                </Context.Provider>
            </Router>
        );

        await waitFor(() => expect(list).toHaveBeenCalledWith({offset: 'current-token', limit: 5}));
        const nextPage = await screen.findByRole('button', {name: /Next page/});
        expect(nextPage).toBeEnabled();

        fireEvent.click(nextPage);

        await waitFor(() => expect(list).toHaveBeenNthCalledWith(2, {offset: 'next-token', limit: 5}));
    });
});
