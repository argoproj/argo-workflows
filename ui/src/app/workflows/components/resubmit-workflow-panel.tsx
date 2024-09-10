import {Checkbox} from 'argo-ui/src/components/checkbox';
import React, {useContext, useState} from 'react';

import {Parameter, ResubmitOpts, Workflow} from '../../../models';
import {Context} from '../../shared/context';
import {uiUrl} from '../../shared/base';
import {ErrorNotice} from '../../shared/components/error-notice';
import {ParametersInput} from '../../shared/components/parameters-input';
import {services} from '../../shared/services';
import {Utils} from '../../shared/utils';

interface Props {
    workflow: Workflow;
    isArchived: boolean;
}

export function ResubmitWorkflowPanel(props: Props) {
    const {navigation} = useContext(Context);
    const [overrideParameters, setOverrideParameters] = useState(false);
    const [workflowParameters, setWorkflowParameters] = useState<Parameter[]>(JSON.parse(JSON.stringify(props.workflow.spec.arguments.parameters || [])));
    const [memoized, setMemoized] = useState(false);
    const [error, setError] = useState<Error>();
    const [isSubmitting, setIsSubmitting] = useState(false);

    async function submit() {
        setIsSubmitting(true);
        const parameters: ResubmitOpts['parameters'] = overrideParameters
            ? [...workflowParameters.filter(p => Utils.getValueFromParameter(p) !== undefined).map(p => p.name + '=' + Utils.getValueFromParameter(p))]
            : [];
        const opts: ResubmitOpts = {
            parameters,
            memoized
        };

        try {
            const submitted = props.isArchived
                ? await services.workflows.resubmitArchived(props.workflow.metadata.uid, props.workflow.metadata.namespace, opts)
                : await services.workflows.resubmit(props.workflow.metadata.name, props.workflow.metadata.namespace, opts);
            navigation.goto(uiUrl(`workflows/${submitted.metadata.namespace}/${submitted.metadata.name}`));
        } catch (err) {
            setError(err);
            setIsSubmitting(false);
        }
    }

    return (
        <>
            <h4>Resubmit Workflow</h4>
            <h5>
                {props.workflow.metadata.namespace}/{props.workflow.metadata.name}
            </h5>
            {error && <ErrorNotice error={error} />}
            <div className='white-box'>
                <div key='override-parameters' style={{marginBottom: 25}}>
                    <label>Override Parameters</label>
                    <div className='columns small-9'>
                        <Checkbox checked={overrideParameters} onChange={setOverrideParameters} />
                    </div>
                </div>

                {overrideParameters && (
                    <div key='parameters' style={{marginBottom: 25}}>
                        <label>Parameters</label>
                        {workflowParameters.length > 0 && <ParametersInput parameters={workflowParameters} onChange={setWorkflowParameters} />}
                        {workflowParameters.length === 0 && (
                            <>
                                <br />
                                <label>No parameters</label>
                            </>
                        )}
                    </div>
                )}

                <div key='memoized' style={{marginBottom: 25}}>
                    <label>Memoized</label>
                    <div className='columns small-9'>
                        <Checkbox checked={memoized} onChange={setMemoized} />
                    </div>
                </div>

                {overrideParameters && memoized && (
                    <div key='warning-override-with-memoized'>
                        <i className='fa fa-exclamation-triangle' style={{color: '#f4c030'}} />
                        Overriding parameters on memoized submitted workflows may have unexpected results.
                    </div>
                )}

                <div key='resubmit'>
                    <button onClick={submit} className='argo-button argo-button--base' disabled={isSubmitting}>
                        <i className='fa fa-plus' /> {isSubmitting ? 'Loading...' : 'Resubmit'}
                    </button>
                </div>
            </div>
        </>
    );
}
