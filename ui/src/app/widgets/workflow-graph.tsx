import * as React from 'react';
import {useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {Workflow} from '../../models';
import {uiUrl} from '../shared/base';
import {ErrorNotice} from '../shared/components/error-notice';
import {Loading} from '../shared/components/loading';
import {historyUrl} from '../shared/history';
import {services} from '../shared/services';
import {WorkflowDag} from '../workflows/components/workflow-dag/workflow-dag';

export const WorkflowGraph = ({history, match}: RouteComponentProps<any>) => {
    const [namespace] = useState(match.params.namespace);
    const [name] = useState(match.params.name);

    const queryParams = new URLSearchParams(location.search);

    const [showOptions] = useState(!!queryParams.get('showOptions'));
    const [nodeSize] = useState(parseInt(queryParams.get('nodeSize') || '20'));
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

    const [workflow, setWorkflow] = useState<Workflow>();
    const [error, setError] = useState<Error>();

    useEffect(() => {
        services.workflows
            .get(namespace, name)
            .then(setWorkflow)
            .catch(setError);
    }, [namespace, name]);

    return (
        <>
            <ErrorNotice error={error} />
            {workflow ? (
                <WorkflowDag
                    nodeClicked={nodeId => window.open(uiUrl(`workflows/${namespace}/${name}?nodeId=${nodeId}`), target)}
                    workflowName={name}
                    nodes={workflow.status.nodes}
                    hideOptions={!showOptions}
                    nodeSize={nodeSize}
                />
            ) : (
                <Loading />
            )}
        </>
    );
};
