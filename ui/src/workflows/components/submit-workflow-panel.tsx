import {Select} from 'argo-ui/src/components/select/select';
import {History} from 'history';
import React, {useContext, useEffect, useMemo, useState} from 'react';

import {uiUrl} from '../../shared/base';
import {ErrorNotice} from '../../shared/components/error-notice';
import {getValueFromParameter, ParametersInput} from '../../shared/components/parameters-input';
import {TagsInput} from '../../shared/components/tags-input/tags-input';
import {Context} from '../../shared/context';
import {getWorkflowParametersFromQuery} from '../../shared/get_workflow_params';
import {Parameter, Template} from '../../shared/models';
import {services} from '../../shared/services';

interface Props {
    kind: string;
    namespace: string;
    name: string;
    entrypoint: string;
    templates: Template[];
    workflowParameters: Parameter[];
    history: History;
}

const workflowEntrypoint = '<default>';
const defaultTemplate: Template = {
    name: workflowEntrypoint,
    inputs: {
        parameters: []
    }
};

export function SubmitWorkflowPanel(props: Props) {
    const {navigation} = useContext(Context);
    const [entrypoint, setEntrypoint] = useState(props.entrypoint || workflowEntrypoint);
    const [parameters, setParameters] = useState<Parameter[]>([]);
    const [workflowParameters, setWorkflowParameters] = useState<Parameter[]>(JSON.parse(JSON.stringify(props.workflowParameters)));
    const [labels, setLabels] = useState(['submit-from-ui=true']);
    const [error, setError] = useState<Error>();
    const [isSubmitting, setIsSubmitting] = useState(false);

    useEffect(() => {
        const templatePropertiesInQuery = getWorkflowParametersFromQuery(props.history);
        // Get the user arguments from the query params
        const updatedParams = workflowParameters.map(param => ({
            name: param.name,
            value: templatePropertiesInQuery[param.name] || param.value
        }));
        setWorkflowParameters(updatedParams);
    }, [props.history, setWorkflowParameters]);

    const templates = useMemo(() => {
        return [defaultTemplate].concat(props.templates);
    }, [props.templates]);

    const templateOptions = useMemo(() => {
        return templates.map(t => ({
            value: t.name,
            title: t.name
        }));
    }, [templates]);

    async function submit() {
        setIsSubmitting(true);
        try {
            const submitted = await services.workflows.submit(props.kind, props.name, props.namespace, {
                entryPoint: entrypoint === workflowEntrypoint ? null : entrypoint,
                parameters: [
                    ...workflowParameters.filter(p => getValueFromParameter(p) !== undefined).map(p => p.name + '=' + getValueFromParameter(p)),
                    ...parameters.filter(p => getValueFromParameter(p) !== undefined).map(p => p.name + '=' + getValueFromParameter(p))
                ],
                labels: labels.join(',')
            });
            navigation.goto(uiUrl(`workflows/${submitted.metadata.namespace}/${submitted.metadata.name}`));
        } catch (err) {
            setError(err);
            setIsSubmitting(false);
        }
    }

    return (
        <>
            <h4>Submit Workflow</h4>
            <h5>
                {props.namespace}/{props.name}
            </h5>
            {error && <ErrorNotice error={error} />}
            <div className='white-box'>
                <div key='entrypoint' title='Entrypoint' style={{marginBottom: 25}}>
                    <label>Entrypoint</label>
                    <Select
                        value={entrypoint}
                        options={templateOptions}
                        onChange={selected => {
                            const selectedTemp = templates.find(t => t.name === selected.value);
                            setEntrypoint(selected.value);
                            setParameters(selectedTemp?.inputs?.parameters || []);
                        }}
                    />
                </div>
                <div key='parameters' style={{marginBottom: 25}}>
                    <label>Parameters</label>
                    {workflowParameters.length > 0 && <ParametersInput parameters={workflowParameters} onChange={setWorkflowParameters} />}
                    {parameters.length > 0 && <ParametersInput parameters={parameters} onChange={setParameters} />}
                    {workflowParameters.length === 0 && parameters.length === 0 ? (
                        <>
                            <br />
                            <label>No parameters</label>
                        </>
                    ) : (
                        <></>
                    )}
                </div>
                <div key='labels' style={{marginBottom: 25}}>
                    <label>Labels</label>
                    <TagsInput tags={labels} onChange={setLabels} />
                </div>
                <div key='submit'>
                    <button onClick={submit} className='argo-button argo-button--base' disabled={isSubmitting}>
                        <i className='fa fa-plus' /> {isSubmitting ? 'Loading...' : 'Submit'}
                    </button>
                </div>
            </div>
        </>
    );
}
