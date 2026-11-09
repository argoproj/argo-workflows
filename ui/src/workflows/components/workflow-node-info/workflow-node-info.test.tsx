import {render, screen} from '@testing-library/react';
import * as React from 'react';

import {archivalStatus, isArchivedWorkflow, NodeStatus, Workflow} from '../../../shared/models';
import {WorkflowNodeInfo} from './workflow-node-info';

jest.mock('../../../shared/components/links', () => ({Links: (): null => null}));

const node = {
    id: 'demo-1234567890',
    name: 'process-a',
    type: 'Pod',
    phase: 'Succeeded',
    startedAt: '2026-10-07T00:00:00Z',
    finishedAt: '2026-10-07T00:00:01Z',
    templateName: 'main'
} as NodeStatus;

describe('WorkflowNodeInfo POD button', () => {
    test('shows the POD button for an archived workflow still in the cluster', () => {
        const workflow = {
            metadata: {name: 'demo', namespace: 'argo', labels: {[archivalStatus]: 'Archived'}},
            status: {nodes: {}}
        } as unknown as Workflow;

        render(
            <WorkflowNodeInfo
                node={node}
                workflow={workflow}
                links={[]}
                archived={isArchivedWorkflow(workflow)}
                onShowContainerLogs={() => undefined}
                onShowPod={() => undefined}
            />
        );

        expect(screen.getByRole('button', {name: 'POD'})).toBeInTheDocument();
    });

    test('hides the POD button for a persisted workflow', () => {
        const workflow = {
            metadata: {name: 'demo', namespace: 'argo', labels: {[archivalStatus]: 'Persisted'}},
            status: {nodes: {}}
        } as unknown as Workflow;

        render(
            <WorkflowNodeInfo
                node={node}
                workflow={workflow}
                links={[]}
                archived={isArchivedWorkflow(workflow)}
                onShowContainerLogs={() => undefined}
                onShowPod={() => undefined}
            />
        );

        expect(screen.queryByRole('button', {name: 'POD'})).not.toBeInTheDocument();
    });
});
