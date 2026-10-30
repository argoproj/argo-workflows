import {fireEvent, render, screen} from '@testing-library/react';
import React from 'react';

import {Info, NodeStatus, Workflow} from '../../../shared/models';
import {services} from '../../../shared/services';
import {WorkflowNodeInfo} from './workflow-node-info';

jest.mock('../../../shared/services/requests');

const workflow = (node: NodeStatus): Workflow =>
    ({
        metadata: {name: 'my-wf', namespace: 'argo'},
        spec: {templates: [{name: 'work', container: {image: 'alpine:3.23'}}]},
        status: {nodes: {[node.id]: node}}
    }) as unknown as Workflow;

const podNode = (extra: Partial<NodeStatus> = {}): NodeStatus =>
    ({
        id: 'my-wf-123',
        name: 'my-wf[0].a',
        displayName: 'a',
        type: 'Pod',
        templateName: 'work',
        phase: 'Running',
        startedAt: '2026-10-15T00:00:00Z',
        ...extra
    }) as NodeStatus;

const claimed = (status: 'True' | 'False', reason: string) => ({type: 'Claimed' as const, status, reason, lastTransitionTime: '2026-10-15T00:00:00Z'});

const archivedLogs = {outputs: {artifacts: [{name: 'main-logs', s3: {key: 'my-wf/my-wf-work-123/main.log'}}]}};

function renderNode(node: NodeStatus) {
    const onShowContainerLogs = jest.fn();
    const onShowEvents = jest.fn();
    render(<WorkflowNodeInfo node={node} workflow={workflow(node)} links={[]} archived={false} onShowContainerLogs={onShowContainerLogs} onShowEvents={onShowEvents} />);
    return {onShowContainerLogs};
}

describe('WorkflowNodeInfo', () => {
    beforeEach(() => {
        jest.spyOn(services.info, 'getInfo').mockResolvedValue({links: [{name: 'Pod Dashboard', scope: 'pod', url: 'http://dashboard/${metadata.name}'}]} as Info);
    });

    afterEach(() => jest.restoreAllMocks());

    it('offers live logs, events and pod links for a pod', async () => {
        renderNode(podNode());

        expect(screen.getByText('my-wf-work-123')).toBeInTheDocument();
        expect(screen.getByRole('button', {name: /^LOGS$/})).toBeInTheDocument();
        expect(screen.getByRole('button', {name: /EVENTS/})).toBeInTheDocument();
        expect(await screen.findByRole('button', {name: /Pod Dashboard/})).toBeInTheDocument();
    });

    it('shows a queued pooled node by its task name, with no logs, events or pod links', () => {
        renderNode(
            podNode({
                phase: 'Pending',
                pooledTask: {poolName: 'work', poolHash: 'hash', taskName: 'my-wf-work-123', taskSetUID: 'uid', enqueuedAt: '', conditions: [claimed('False', 'Enqueued')]}
            })
        );

        expect(screen.getByText('TASK NAME')).toBeInTheDocument();
        expect(screen.getByText('my-wf-work-123')).toBeInTheDocument();
        expect(screen.getByText('WORKER POD')).toBeInTheDocument();
        expect(screen.getByText('none (queued)')).toBeInTheDocument();
        expect(screen.queryByText('POD NAME')).not.toBeInTheDocument();
        expect(screen.getByText('POOL')).toBeInTheDocument();
        expect(services.info.getInfo).not.toHaveBeenCalled();
        expect(screen.queryByRole('button', {name: /LOGS/})).not.toBeInTheDocument();
        expect(screen.queryByRole('button', {name: /EVENTS/})).not.toBeInTheDocument();
        expect(screen.queryByRole('button', {name: /Pod Dashboard/})).not.toBeInTheDocument();
    });

    it('says a pooled node that ended after a release was released', () => {
        renderNode(
            podNode({
                phase: 'Failed',
                pooledTask: {
                    poolName: 'work',
                    poolHash: 'hash',
                    taskName: 'my-wf-work-123',
                    taskSetUID: 'uid',
                    enqueuedAt: '',
                    conditions: [claimed('False', 'Released')],
                    releases: 1
                }
            })
        );

        expect(screen.getByText('none (released)')).toBeInTheDocument();
    });

    it('shows a claimed pooled node by its worker pod, with no live logs', () => {
        renderNode(
            podNode({
                pooledTask: {
                    poolName: 'work',
                    poolHash: 'hash',
                    taskName: 'my-wf-work-123',
                    conditions: [claimed('True', 'Claimed')],
                    taskSetUID: 'uid',
                    enqueuedAt: '',
                    workerPodName: 'work-0123456789-abcde'
                }
            })
        );

        expect(screen.getByText('my-wf-work-123')).toBeInTheDocument();
        expect(screen.getByText('work-0123456789-abcde')).toBeInTheDocument();
        expect(screen.queryByText(/\(queued\)/)).not.toBeInTheDocument();
        expect(screen.queryByRole('button', {name: /LOGS/})).not.toBeInTheDocument();
        expect(screen.queryByRole('button', {name: /EVENTS/})).not.toBeInTheDocument();
    });

    it('offers the archived logs of a finished pooled node', () => {
        const {onShowContainerLogs} = renderNode(
            podNode({
                phase: 'Succeeded',
                ...archivedLogs,
                pooledTask: {
                    poolName: 'work',
                    poolHash: 'hash',
                    taskName: 'my-wf-work-123',
                    conditions: [claimed('True', 'Claimed')],
                    taskSetUID: 'uid',
                    enqueuedAt: '',
                    workerPodName: 'work-0123456789-abcde'
                }
            } as Partial<NodeStatus>)
        );

        fireEvent.click(screen.getByRole('button', {name: /ARCHIVED LOGS/}));
        expect(onShowContainerLogs).toHaveBeenCalledWith('my-wf-123', 'main');
        expect(screen.queryByRole('button', {name: /EVENTS/})).not.toBeInTheDocument();
        expect(services.info.getInfo).not.toHaveBeenCalled();
    });
});
