import * as React from 'react';
import {useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {NodeStatus} from '../../models';
import {uiUrl} from '../shared/base';
import {ErrorNotice} from '../shared/components/error-notice';
import {Loading} from '../shared/components/loading';
import {historyUrl} from '../shared/history';
import {RetryWatch} from '../shared/retry-watch';
import {services} from '../shared/services';
import {WorkflowDag} from '../workflows/components/workflow-dag/workflow-dag';

export const WorkflowGraph = ({history, match}: RouteComponentProps<any>) => {
    const [namespace] = useState(match.params.namespace);
    const [name] = useState(match.params.name);

    const queryParams = new URLSearchParams(location.search);

    const [showOptions] = useState(queryParams.get('showOptions') === 'true');
    const [nodeSize] = useState(parseInt(queryParams.get('nodeSize'), 10) || 16);
    const [target] = useState(queryParams.get('target') || '_top');

    useEffect(() => {
        history.push(
            historyUrl('widgets/workflow-graphs/{namespace}/{name}', {
                namespace,
                name,
                showOptions,
                nodeSize,
                target
            })
        );
    }, [namespace, name]);

    const [nodes, setNodes] = useState<{[nodeId: string]: NodeStatus}>();
    const [error, setError] = useState<Error>();

    useEffect(() => {
        const w = new RetryWatch(
            () => services.workflows.watch({namespace, name}),
            () => setError(null),
            e => setNodes(e.object.status.nodes),
            setError
        );
        w.start();
        return () => w.stop();
    }, [namespace, name]);

    return (
        <>
            <ErrorNotice error={error} />
            {nodes ? (
                <WorkflowDag
                    nodeClicked={nodeId => window.open(uiUrl(`workflows/${namespace}/${name}?nodeId=${nodeId}`), target)}
                    workflowName={name}
                    nodes={nodes}
                    hideOptions={!showOptions}
                    nodeSize={nodeSize}
                />
            ) : (
                <Loading />
            )}
        </>
    );
};
