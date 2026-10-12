import {render, screen, waitFor} from '@testing-library/react';
import * as React from 'react';

import {request} from '../parameterization/api';
import {Context} from '../shared/context';
import {services} from '../shared/services';
import {ClusterWorkflowTemplateDetails} from './cluster-workflow-template-details';

jest.mock('argo-ui/src/components/page/page', () => ({Page: ({children}: any) => <div>{children}</div>}));
jest.mock('argo-ui/src/components/sliding-panel/sliding-panel', () => ({SlidingPanel: (): null => null}));
jest.mock('../shared/base', () => ({uiUrl: (path: string) => '/' + path}));
jest.mock('../shared/use-collect-event', () => ({useCollectEvent: (): void => undefined}));
jest.mock('../shared/use-query-params', () => ({useQueryParams: () => (): void => undefined}));
jest.mock('../workflow-templates/workflow-template-editor', () => ({WorkflowTemplateEditor: (): null => null}));
jest.mock('../workflows/components/submit-workflow-panel', () => ({SubmitWorkflowPanel: (): null => null}));
jest.mock('../workflows/components/workflow-details-list/workflow-details-list', () => ({WorkflowDetailsList: (): null => null}));
jest.mock('../parameterization/api', () => ({request: jest.fn()}));
jest.mock('../shared/services', () => ({
    services: {
        clusterWorkflowTemplate: {get: jest.fn()},
        info: {getInfo: jest.fn()},
        workflows: {list: jest.fn()}
    }
}));

beforeEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
    (services.clusterWorkflowTemplate.get as jest.Mock).mockResolvedValue({metadata: {name: 'native'}, spec: {arguments: {parameters: []}}});
    (services.info.getInfo as jest.Mock).mockResolvedValue({managedNamespace: 'lab', columns: []});
    (request as jest.Mock).mockResolvedValue({items: [{metadata: {name: 'cluster-form'}, spec: {workflowTemplate: 'native', clusterScope: true, parameterTemplate: 'typed'}}]});
});

function open() {
    return render(
        <Context.Provider value={{navigation: {}, notifications: {}, popup: {}} as any}>
            <ClusterWorkflowTemplateDetails history={{push: jest.fn()} as any} location={{search: ''} as any} match={{params: {name: 'native'}} as any} />
        </Context.Provider>
    );
}

it('loads cluster bindings with a namespaced server and limits execution history to that namespace', async () => {
    (services.workflows.list as jest.Mock).mockResolvedValue({items: []});
    open();
    expect(await screen.findByRole('link', {name: 'cluster-form'})).toHaveAttribute('href', '/parameters/lab?binding=cluster-form');
    expect(services.workflows.list).toHaveBeenCalledWith('lab', null, expect.any(Array), {limit: 50});
});

it('keeps the binding available when historical execution listing is forbidden', async () => {
    (services.workflows.list as jest.Mock).mockRejectedValue(new Error('history denied'));
    open();
    expect(await screen.findByRole('link', {name: 'cluster-form'})).toBeVisible();
    await waitFor(() => expect(screen.getAllByText(/history denied/).length).toBeGreaterThan(0));
});

it('preserves cluster-wide history for an unrestricted server', async () => {
    (services.info.getInfo as jest.Mock).mockResolvedValue({columns: []});
    (services.workflows.list as jest.Mock).mockResolvedValue({items: []});
    open();
    expect(await screen.findByRole('link', {name: 'cluster-form'})).toHaveAttribute('href', '/parameters/default?binding=cluster-form');
    expect(services.workflows.list).toHaveBeenCalledWith('', null, expect.any(Array), {limit: 50});
});
