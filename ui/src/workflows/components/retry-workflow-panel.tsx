import {Checkbox} from 'argo-ui/src/components/checkbox';
import {Tooltip} from 'argo-ui/src/components/tooltip/tooltip';
import React, {useState} from 'react';

import {ErrorNotice} from '../../shared/components/error-notice';
import {getValueFromParameter, ParametersInput} from '../../shared/components/parameters-input';
import {Parameter, RetryOpts, Workflow} from '../../shared/models';
import {services} from '../../shared/services';

interface Props {
    nodeId?: string;
    workflow: Workflow;
    isArchived: boolean;
    isWorkflowInCluster: boolean;
    onRetrySuccess: () => void;
}

export function RetryWorkflowPanel(props: Props) {
    const [overrideParameters, setOverrideParameters] = useState(false);
    const [restartSuccessful, setRestartSuccessful] = useState(false);
    const [workflowParameters, setWorkflowParameters] = useState<Parameter[]>(JSON.parse(JSON.stringify(props.workflow.spec.arguments.parameters || [])));
    const [nodeFieldSelector, setNodeFieldSelector] = useState('');
    const [error, setError] = useState<Error>();
    const [isSubmitting, setIsSubmitting] = useState(false);

    async function submit() {
        setIsSubmitting(true);
        const parameters: RetryOpts['parameters'] = overrideParameters
            ? [...workflowParameters.filter(p => getValueFromParameter(p) !== undefined).map(p => p.name + '=' + getValueFromParameter(p))]
            : [];
        const opts: RetryOpts = {
            parameters,
            restartSuccessful,
            nodeFieldSelector: props.nodeId ? `id=${props.nodeId}` : nodeFieldSelector
        };

        try {
            props.isArchived && !props.isWorkflowInCluster
                ? await services.workflows.retryArchived(props.workflow.metadata.uid, props.workflow.metadata.namespace, opts)
                : await services.workflows.retry(props.workflow.metadata.name, props.workflow.metadata.namespace, opts);
            props.onRetrySuccess();
        } catch (err) {
            setError(err);
            setIsSubmitting(false);
        }
    }

    return (
        <>
            <h4>Retry Workflow</h4>
            <h5>
                {props.workflow.metadata.namespace}/{props.workflow.metadata.name}
                {props.nodeId ? `/${props.nodeId}` : ''}
            </h5>

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
                    <label>Restart Successful</label>
                    <div className='columns small-9'>
                        <Checkbox checked={restartSuccessful} onChange={setRestartSuccessful} />
                        <Tooltip content='Checking this box will re-run previously successful nodes as well'>
                            <i className='fa fa-question-circle' style={{marginLeft: 4}} />
                        </Tooltip>
                    </div>
                </div>

                {restartSuccessful && !props.nodeId && (
                    <div key='node-field-selector' style={{marginBottom: 25}}>
                        <label>
                            <a href='https://argo-workflows.readthedocs.io/en/latest/node-field-selector/'>Node Field Selector</a> to restart nodes..
                        </label>

                        <div className='columns small-9'>
                            <textarea className='argo-field' value={nodeFieldSelector} onChange={e => setNodeFieldSelector(e.target.value)} />
                        </div>
                    </div>
                )}

                {/* Retry button */}
                <div key='retry'>
                    <button onClick={submit} className='argo-button argo-button--base' disabled={isSubmitting}>
                        <i className='fa fa-plus' /> {isSubmitting ? 'Loading...' : 'Retry'}
                    </button>
                </div>
            </div>
        </>
    );
}
