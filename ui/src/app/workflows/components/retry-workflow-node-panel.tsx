import {Checkbox} from 'argo-ui/src/components/checkbox';
import {Tooltip} from 'argo-ui/src/components/tooltip/tooltip';
import React, {useState} from 'react';

import {Parameter, RetryOpts, Workflow} from '../../../models';
import {getValueFromParameter, ParametersInput} from '../../shared/components/parameters-input';
import {services} from '../../shared/services';
import {ErrorNotice} from '../../shared/components/error-notice';

interface Props {
    nodeId: string;
    workflow: Workflow;
    isArchived: boolean;
    isWorkflowInCluster: boolean;
    onRetrySuccess: () => void;
}

export function RetryWorkflowNode(props: Props) {
    const [overrideParameters, setOverrideParameters] = useState(false);
    const [restartSuccessful, setRestartSuccessful] = useState(false);
    const [workflowParameters, setWorkflowParameters] = useState<Parameter[]>(JSON.parse(JSON.stringify(props.workflow.spec.arguments.parameters || [])));
    const [error, setError] = useState<Error>();

    async function submit() {
        const parameters: RetryOpts['parameters'] = overrideParameters
            ? [...workflowParameters.filter(p => getValueFromParameter(p) !== undefined).map(p => p.name + '=' + getValueFromParameter(p))]
            : [];
        const opts: RetryOpts = {
            parameters,
            restartSuccessful,
            nodeFieldSelector: `id=${props.nodeId}`
        };

        try {
            props.isArchived && !props.isWorkflowInCluster
                ? await services.workflows.retryArchived(props.workflow.metadata.uid, props.workflow.metadata.namespace, opts)
                : await services.workflows.retry(props.workflow.metadata.name, props.workflow.metadata.namespace, opts);
            props.onRetrySuccess();
        } catch (err) {
            setError(err);
        }
    }

    return (
        <div style={{padding: 16}}>
            <h4>Retry Node</h4>
            <h5>
                {props.workflow.metadata.namespace}/{props.nodeId}
            </h5>

            <p>Note: Retrying this node will re-execute this node and all downstream nodes.</p>

            {error && <ErrorNotice error={error} />}
            <div className='white-box'>
                {/* Override Parameters */}
                <div key='override-parameters' style={{marginBottom: 25}}>
                    <label>Override Parameters</label>
                    <div className='columns small-9'>
                        <Checkbox checked={overrideParameters} onChange={setOverrideParameters} />
                    </div>
                </div>

                {overrideParameters && (
                    <div key='parameters' style={{marginBottom: 25}}>
                        <label>Parameters</label>
                        {workflowParameters.length > 0 ? (
                            <ParametersInput parameters={workflowParameters} onChange={setWorkflowParameters} />
                        ) : (
                            <>
                                <br />
                                <label>No parameters</label>
                            </>
                        )}
                    </div>
                )}

                {/* Restart Successful */}
                <div key='restart-successful' style={{marginBottom: 25}}>
                    <label>
                        Restart Successful{' '}
                        <Tooltip content='Leaving this box unchecked avoids re-running nodes that have run successfully before'>
                            <i className='fa fa-question-circle' style={{marginLeft: 4}} />
                        </Tooltip>
                    </label>
                    <div className='columns small-9'>
                        <Checkbox checked={restartSuccessful} onChange={setRestartSuccessful} />
                    </div>
                </div>

                {/* Retry button */}
                <div key='retry'>
                    <button onClick={submit} className='argo-button argo-button--base'>
                        <i className='fa fa-undo-alt' /> Retry
                    </button>
                </div>
            </div>
        </div>
    );
}
