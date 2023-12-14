import * as React from 'react';
import {useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {ArtifactRepositoryRefStatus, NodeStatus} from '../../models';
import {uiUrl} from '../shared/base';
import {ErrorNotice} from '../shared/components/error-notice';
import {historyUrl} from '../shared/history';
import {RetryWatch} from '../shared/retry-watch';
import {services} from '../shared/services';
import {WorkflowDag} from '../workflows/components/workflow-dag/workflow-dag';

export function WorkflowGraph({history, match}: RouteComponentProps<any>) {
    const queryParams = new URLSearchParams(location.search);
    const namespace = match.params.namespace;
    const name = queryParams.get('name');
    const label = queryParams.get('label');
    const showOptions = queryParams.get('showOptions') === 'true';
    const nodeSize = parseInt(queryParams.get('nodeSize'), 10) || 32;
    const target = queryParams.get('target') || '_top';

    useEffect(() => {
        history.push(
            historyUrl('widgets/workflow-graphs/{namespace}', {
                namespace,
                name,
                label,
                showOptions,
                nodeSize,
                target
            })
        );
    }, [namespace, name, label]);

    const [displayName, setDisplayName] = useState<string>();
    const [creationTimestamp, setCreationTimestamp] = useState<Date>(); // used to make sure we only display the most recent one
    const [nodes, setNodes] = useState<{[nodeId: string]: NodeStatus}>();
    const [artifactRepositoryRef, setArtifactRepositoryRef] = useState<ArtifactRepositoryRefStatus>();
    const [error, setError] = useState<Error>();

    useEffect(() => {
        const w = new RetryWatch(
            () => services.workflows.watch({namespace, name, labels: [label]}),
            () => setError(null),
            e => {
                const wf = e.object;
                const t = new Date(wf.metadata.creationTimestamp);
                if (t < creationTimestamp) {
                    return;
                }
                setDisplayName(wf.metadata.name);
                setNodes(wf.status.nodes);
                setCreationTimestamp(t);
                setArtifactRepositoryRef(wf.status.artifactRepositoryRef);
            },
            setError
        );
        w.start();
        return () => w.stop();
    }, [namespace, name, label]);

    return (
        <>
            <ErrorNotice error={error} />
            <WorkflowDag
                nodeClicked={nodeId => window.open(uiUrl(`workflows/${namespace}/${displayName}?nodeId=${nodeId}`), target)}
                workflowName={displayName}
                artifactRepositoryRef={artifactRepositoryRef}
                nodes={nodes || {}}
                hideOptions={!showOptions}
                nodeSize={nodeSize}
            />
        </>
    );
}
