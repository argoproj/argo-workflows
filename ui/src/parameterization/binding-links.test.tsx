import {render, screen, waitFor} from '@testing-library/react';
import * as React from 'react';

import {request} from './api';
import {BindingLinks} from './binding-links';

jest.mock('./api', () => ({request: jest.fn()}));
jest.mock('../shared/base', () => ({uiUrl: (path: string) => '/argo/' + path}));
beforeEach(() => (request as jest.Mock).mockReset());

it('separates same-name cluster and namespace templates and updates after scope changes', async () => {
    (request as jest.Mock).mockResolvedValue({
        items: [
            {metadata: {name: 'cluster-binding'}, spec: {workflowTemplate: 'native', clusterScope: true}},
            {metadata: {name: 'namespace-binding'}, spec: {workflowTemplate: 'native'}}
        ]
    });
    const view = render(<BindingLinks namespace='lab' workflowTemplate='native' clusterScope={true} />);
    expect(await screen.findByRole('link', {name: 'cluster-binding'})).toBeVisible();
    expect(screen.queryByRole('link', {name: 'namespace-binding'})).toBeNull();
    view.rerender(<BindingLinks namespace='lab' workflowTemplate='native' />);
    expect(await screen.findByRole('link', {name: 'namespace-binding'})).toBeVisible();
    expect(screen.queryByRole('link', {name: 'cluster-binding'})).toBeNull();
});

it('discovers all matching bindings across pages and links to an exact model choice', async () => {
    (request as jest.Mock).mockImplementation(async (_ns, path) =>
        path === 'bindings'
            ? {
                  items: [{metadata: {name: 'unrelated'}, spec: {workflowTemplate: 'other'}}],
                  metadata: {continue: 'next'}
              }
            : {
                  items: ['alpha', 'beta'].map(name => ({metadata: {name}, spec: {workflowTemplate: 'native', parameterTemplate: 'typed', form: 'form'}}))
              }
    );
    render(<BindingLinks namespace='lab' workflowTemplate='native' />);
    expect(await screen.findByRole('link', {name: 'alpha'})).toHaveAttribute('href', '/argo/parameters/lab?binding=alpha');
    expect(screen.getByRole('link', {name: 'beta'})).toBeVisible();
    expect(screen.queryByText('unrelated')).toBeNull();
    expect(request).toHaveBeenCalledWith('lab', 'bindings?continue=next');
});

it('preserves unextended native behavior and discards stale namespace responses', async () => {
    let oldResult: (value: any) => void;
    (request as jest.Mock).mockImplementation(async ns => {
        if (ns === 'old')
            return new Promise(resolve => {
                oldResult = resolve;
            });
        throw {status: 404};
    });
    const view = render(<BindingLinks namespace='old' workflowTemplate='native' />);
    view.rerender(<BindingLinks namespace='new' workflowTemplate='native' />);
    oldResult({items: [{metadata: {name: 'old-binding'}, spec: {workflowTemplate: 'native'}}]});
    await waitFor(() => expect(screen.queryByRole('region')).toBeNull());
    expect(screen.queryByRole('link')).toBeNull();
});

it('reports authorization failures without exposing or selecting an unrelated binding', async () => {
    (request as jest.Mock).mockRejectedValue({status: 403});
    render(<BindingLinks namespace='denied' workflowTemplate='native' />);
    expect(await screen.findByRole('status')).toHaveTextContent('could not be loaded');
    expect(screen.queryByRole('link')).toBeNull();
});
