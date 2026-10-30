import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import React from 'react';
import {of} from 'rxjs';

import {Context, ContextApis} from '../../../shared/context';
import {Info, NodeStatus, Workflow} from '../../../shared/models';
import {services} from '../../../shared/services';
import {WorkflowLogsViewer} from './workflow-logs-viewer';

jest.mock('../../../shared/services/requests');

const podNode = {
    id: 'my-wf-123',
    name: 'my-wf[0].a',
    displayName: 'a',
    type: 'Pod',
    templateName: 'work',
    phase: 'Running',
    startedAt: '2026-10-15T00:00:00Z'
} as NodeStatus;

const pooledNode = {
    id: 'my-wf-456',
    name: 'my-wf[0].b',
    displayName: 'b',
    type: 'Pod',
    templateName: 'work',
    phase: 'Succeeded',
    startedAt: '2026-10-15T00:00:00Z',
    outputs: {artifacts: [{name: 'main-logs', s3: {key: 'my-wf/my-wf-work-456/main.log'}}]},
    pooledTask: {
        poolName: 'work',
        poolHash: 'hash',
        taskName: 'my-wf-work-456',
        taskSetUID: 'uid',
        enqueuedAt: '',
        workerPodName: 'work-abc',
        conditions: [{type: 'Claimed', status: 'True', reason: 'Claimed', lastTransitionTime: '2026-10-15T00:00:00Z'}]
    }
} as NodeStatus;

const workflow = {
    metadata: {name: 'my-wf', namespace: 'argo'},
    spec: {templates: [{name: 'work', container: {image: 'alpine:3.23'}}]},
    status: {phase: 'Running', nodes: {[podNode.id]: podNode, [pooledNode.id]: pooledNode}}
} as unknown as Workflow;

function renderViewer(node: NodeStatus, podName: string) {
    render(
        <Context.Provider value={{popup: {}} as ContextApis}>
            <WorkflowLogsViewer workflow={workflow} initialNodeId={node.id} initialPodName={podName} container='main' archived={false} />
        </Context.Provider>
    );
}

function selectPod(from: string, to: string) {
    const input = screen.getByDisplayValue(from);
    fireEvent.focus(input);
    fireEvent.click(input);
    fireEvent.click(screen.getByText(to));
}

describe('WorkflowLogsViewer', () => {
    let fromCluster: jest.SpyInstance;
    let fromArtifact: jest.SpyInstance;

    beforeEach(() => {
        jest.spyOn(services.info, 'getInfo').mockResolvedValue({links: []} as Info);
        jest.spyOn(services.workflows, 'get').mockResolvedValue(workflow);
        fromCluster = jest.spyOn(services.workflows, 'getContainerLogsFromCluster').mockReturnValue(of({content: 'live', podName: 'live'}));
        fromArtifact = jest.spyOn(services.workflows, 'getContainerLogsFromArtifact').mockReturnValue(of({content: 'archived', podName: 'archived'}));
    });

    afterEach(() => jest.restoreAllMocks());

    it('shows the archived logs of a pooled pod picked after opening on a pod', async () => {
        renderViewer(podNode, 'my-wf-work-123');
        await waitFor(() => expect(fromCluster).toHaveBeenCalledWith(workflow, 'my-wf-work-123', 'main', ''));

        selectPod('a (my-wf-work-123)', 'b (work-abc)');

        await waitFor(() => expect(fromArtifact).toHaveBeenCalledWith(workflow, 'my-wf-456', 'main', '', false));
        expect(fromCluster).not.toHaveBeenCalledWith(workflow, 'my-wf-work-456', 'main', '');
    });

    it('shows the live logs of a pod picked after opening on a pooled pod', async () => {
        renderViewer(pooledNode, 'my-wf-work-456');
        await waitFor(() => expect(fromArtifact).toHaveBeenCalledWith(workflow, 'my-wf-456', 'main', '', false));
        expect(fromCluster).not.toHaveBeenCalled();

        selectPod('b (work-abc)', 'a (my-wf-work-123)');

        await waitFor(() => expect(fromCluster).toHaveBeenCalledWith(workflow, 'my-wf-work-123', 'main', ''));
        expect(fromArtifact).toHaveBeenCalledTimes(1);
    });

    it.each([
        ['a pod', podNode, 'my-wf-work-123', 'a (my-wf-work-123)'],
        ['a pooled pod', pooledNode, 'my-wf-work-456', 'b (work-abc)']
    ])('keeps the initial node for All after opening on %s', async (_, node, podName, label) => {
        const getContainerLogs = jest.spyOn(services.workflows, 'getContainerLogs');
        renderViewer(node, podName);
        await waitFor(() => expect(getContainerLogs).toHaveBeenCalledWith(workflow, podName, node.id, 'main', '', false));

        selectPod(label, 'All');

        await waitFor(() => expect(getContainerLogs).toHaveBeenCalledWith(workflow, '', node.id, 'main', '', false));
    });
});
