import {Select} from 'argo-ui';
import React, {useState} from 'react';
import {Parameter, Template, Workflow} from '../../../models';
import {uiUrl} from '../../shared/base';
import {ErrorNotice} from '../../shared/components/error-notice';
import {ParametersInput} from '../../shared/components/parameters-input/parameters-input';
import {TagsInput} from '../../shared/components/tags-input/tags-input';
import {services} from '../../shared/services';
import {Utils} from '../../shared/utils';

interface Props {
    kind: string;
    namespace: string;
    name: string;
    entrypoint: string;
    templates: Template[];
    workflowParameters: Parameter[];
}

const workflowEntrypoint = '<default>';

export function SubmitWorkflowPanel(props: Props) {
    const defaultTemplate: Template = {
        name: workflowEntrypoint,
        inputs: {
            parameters: []
        }
    };

    const [entrypoint, setEntrypoint] = useState<string>(workflowEntrypoint);
    const [selectedTemplate, setSelectedTemplate] = useState<Template>(defaultTemplate);
    const [parameters, setParameters] = useState<Parameter[]>([]);
    const [workflowParameters, setWorkflowParameters] = useState<Parameter[]>(JSON.parse(JSON.stringify(props.workflowParameters)));
    const [templates] = useState<Template[]>([defaultTemplate].concat(props.templates));
    const [labels, setLabels] = useState<string[]>(['submit-from-ui=true']);
    const [error, setError] = useState<Error>();
    const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

    const getSelectedTemplate = (name: string): Template | null => {
        return templates.find(t => t.name === name) || null;
    };

    const submit = () => {
        setIsSubmitting(true);
        services.workflows
            .submit(props.kind, props.name, props.namespace, {
                entryPoint: entrypoint === workflowEntrypoint ? null : entrypoint,
                parameters: [
                    ...workflowParameters.filter(p => Utils.getValueFromParameter(p) !== undefined).map(p => p.name + '=' + Utils.getValueFromParameter(p)),
                    ...parameters.filter(p => Utils.getValueFromParameter(p) !== undefined).map(p => p.name + '=' + Utils.getValueFromParameter(p))
                ],
                labels: labels.join(',')
            })
            .then((submitted: Workflow) => (document.location.href = uiUrl(`workflows/${submitted.metadata.namespace}/${submitted.metadata.name}`)))
            .catch(err => {
                setError(err);
                setIsSubmitting(false);
            });
    };

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
                        options={templates.map(t => ({
                            value: t.name,
                            title: t.name
                        }))}
                        onChange={selected => {
                            const selectedTemp = getSelectedTemplate(selected.value);
                            setEntrypoint(selected.value);
                            setSelectedTemplate(selectedTemp || defaultTemplate);
                            setParameters((selectedTemp && selectedTemp.inputs.parameters) || []);
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
