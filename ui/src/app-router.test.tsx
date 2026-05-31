import {render} from '@testing-library/react';
import * as React from 'react';
import {MemoryRouter, Navigate, Route, Routes, useLocation, useParams} from 'react-router-dom';

import {ClusterWorkflowTemplateContainer} from './cluster-workflow-templates/cluster-workflow-template-container';
import {CronWorkflowContainer} from './cron-workflows/cron-workflow-container';
import {WorkflowsContainer} from './workflows/components/workflows-container';

// A stand-in leaf that surfaces the resolved route params + query string so the
// test can assert that the route tree extracts them correctly post react-router v7 migration.
function Probe({name}: {name: string}) {
    const params = useParams();
    const location = useLocation();
    return (
        <div data-testid={name}>
            <span data-testid={`${name}-namespace`}>{params.namespace ?? ''}</span>
            <span data-testid={`${name}-name`}>{params.name ?? ''}</span>
            <span data-testid={`${name}-search`}>{location.search}</span>
        </div>
    );
}

// Mock the leaf components used by each container so that mounting a container only
// exercises routing/param-resolution, not data fetching or websockets.
jest.mock('./workflows/components/workflows-list/workflows-list', () => ({WorkflowsList: () => <Probe name='workflows-list' />}));
jest.mock('./workflows/components/workflow-details/workflow-details', () => ({WorkflowDetails: () => <Probe name='workflow-details' />}));
jest.mock('./cluster-workflow-templates/cluster-workflow-template-list', () => ({ClusterWorkflowTemplateList: () => <Probe name='cwft-list' />}));
jest.mock('./cluster-workflow-templates/cluster-workflow-template-details', () => ({ClusterWorkflowTemplateDetails: () => <Probe name='cwft-details' />}));
jest.mock('./cron-workflows/cron-workflow-list', () => ({CronWorkflowList: () => <Probe name='cron-list' />}));
jest.mock('./cron-workflows/cron-workflow-details', () => ({CronWorkflowDetails: () => <Probe name='cron-details' />}));

// Mirror how app-router.tsx nests each container under a `path="section/*"` parent route.
function renderAt(url: string, section: string, Container: React.ComponentType) {
    return render(
        <MemoryRouter initialEntries={[url]}>
            <Routes>
                <Route path={`${section}/*`} element={<Container />} />
            </Routes>
        </MemoryRouter>
    );
}

describe('app route tree (react-router v7)', () => {
    it('list route with namespace resolves the namespace param', () => {
        const {getByTestId} = renderAt('/workflows/argo', 'workflows', WorkflowsContainer);
        expect(getByTestId('workflows-list')).toBeInTheDocument();
        expect(getByTestId('workflows-list-namespace')).toHaveTextContent('argo');
    });

    it('list route preserves the query string', () => {
        const {getByTestId} = renderAt('/workflows/argo?phase=Running', 'workflows', WorkflowsContainer);
        expect(getByTestId('workflows-list')).toBeInTheDocument();
        expect(getByTestId('workflows-list-search')).toHaveTextContent('?phase=Running');
    });

    it('detail route with namespace + name resolves both params', () => {
        const {getByTestId} = renderAt('/workflows/argo/my-wf', 'workflows', WorkflowsContainer);
        expect(getByTestId('workflow-details')).toBeInTheDocument();
        expect(getByTestId('workflow-details-namespace')).toHaveTextContent('argo');
        expect(getByTestId('workflow-details-name')).toHaveTextContent('my-wf');
    });

    it('cluster-scoped detail route resolves the name param (no namespace)', () => {
        const {getByTestId} = renderAt('/cluster-workflow-templates/foo', 'cluster-workflow-templates', ClusterWorkflowTemplateContainer);
        expect(getByTestId('cwft-details')).toBeInTheDocument();
        expect(getByTestId('cwft-details-name')).toHaveTextContent('foo');
    });

    it('cluster-scoped index route renders the list', () => {
        const {getByTestId} = renderAt('/cluster-workflow-templates', 'cluster-workflow-templates', ClusterWorkflowTemplateContainer);
        expect(getByTestId('cwft-list')).toBeInTheDocument();
    });

    it('optional-namespace route with no namespace renders the list', () => {
        const {getByTestId} = renderAt('/cron-workflows', 'cron-workflows', CronWorkflowContainer);
        expect(getByTestId('cron-list')).toBeInTheDocument();
        expect(getByTestId('cron-list-namespace')).toHaveTextContent('');
    });

    it('optional-namespace route with a namespace resolves it', () => {
        const {getByTestId} = renderAt('/cron-workflows/argo', 'cron-workflows', CronWorkflowContainer);
        expect(getByTestId('cron-list')).toBeInTheDocument();
        expect(getByTestId('cron-list-namespace')).toHaveTextContent('argo');
    });

    it('unknown path falls through to the default redirect', () => {
        // mirrors app-router.tsx: a catch-all `path="*"` route redirects to the workflows list
        function Tree() {
            return (
                <Routes>
                    <Route path='workflows/*' element={<WorkflowsContainer />} />
                    <Route path='*' element={<Navigate to='/workflows/argo' replace />} />
                </Routes>
            );
        }
        const {getByTestId} = render(
            <MemoryRouter initialEntries={['/does-not-exist']}>
                <Tree />
            </MemoryRouter>
        );
        // redirected into the workflows list with the fallback namespace
        expect(getByTestId('workflows-list')).toBeInTheDocument();
        expect(getByTestId('workflows-list-namespace')).toHaveTextContent('argo');
    });
});
