import { Select } from 'argo-ui/src/components/select/select';
import { History } from 'history';
import React, { useContext, useEffect, useMemo, useState } from 'react';

import { uiUrl } from '../../shared/base';
import { ArtifactsInput, ArtifactUploadResponse } from '../../shared/components/artifacts-input';
import { ErrorNotice } from '../../shared/components/error-notice';
import { getValueFromParameter, ParametersInput } from '../../shared/components/parameters-input';
import { TagsInput } from '../../shared/components/tags-input/tags-input';
import { Context } from '../../shared/context';
import { getWorkflowParametersFromQuery } from '../../shared/get_workflow_params';
import { Artifact, Parameter, Template } from '../../shared/models';
import { services } from '../../shared/services';

interface Props {
    kind: string;
    namespace: string;
    name: string;
    entrypoint: string;
    templates: Template[];
    workflowParameters: Parameter[];
    workflowArtifacts?: Artifact[];
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
    const { navigation } = useContext(Context);
    const [entrypoint, setEntrypoint] = useState(props.entrypoint || workflowEntrypoint);
    const [parameters, setParameters] = useState<Parameter[]>([]);
    const [workflowParameters, setWorkflowParameters] = useState<Parameter[]>(JSON.parse(JSON.stringify(props.workflowParameters)));
    const [labels, setLabels] = useState(['submit-from-ui=true']);
    const [error, setError] = useState<Error>();
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [uploadedArtifacts, setUploadedArtifacts] = useState<Record<string, ArtifactUploadResponse>>({});

    const handleArtifactUpload = (artifactName: string, response: ArtifactUploadResponse) => {
        setUploadedArtifacts(prev => ({ ...prev, [artifactName]: response }));
    };

    useEffect(() => {
        const templatePropertiesInQuery = getWorkflowParametersFromQuery(props.history);
        // Get the user arguments from the query params
        const updatedParams = workflowParameters.map(param => ({
            ...param,
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
            // Build artifacts array from uploaded artifacts
            const artifactOverrides = Object.entries(uploadedArtifacts).map(([name, response]) => {
                // Format: name=s3://bucket/key (the API expects this format)
                if (response.location?.s3) {
                    return `${name}=s3://${response.location.s3.bucket}/${response.location.s3.key}`;
                }
                // Fallback: use the key directly if location format is unknown
                return `${name}=${response.key}`;
            });

            const submitted = await services.workflows.submit(props.kind, props.name, props.namespace, {
                entryPoint: entrypoint === workflowEntrypoint ? null : entrypoint,
                parameters: [
                    ...workflowParameters.filter(p => getValueFromParameter(p) !== undefined).map(p => p.name + '=' + getValueFromParameter(p)),
                    ...parameters.filter(p => getValueFromParameter(p) !== undefined).map(p => p.name + '=' + getValueFromParameter(p))
                ],
                labels: labels.join(','),
                artifacts: artifactOverrides.length > 0 ? artifactOverrides : undefined
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
                <div key='entrypoint' title='Entrypoint' style={{ marginBottom: 25 }}>
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
                <div key='parameters' style={{ marginBottom: 25 }}>
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
                {props.workflowArtifacts && props.workflowArtifacts.length > 0 && (
                    <div key='artifacts' style={{ marginBottom: 25 }}>
                        <label>Input Artifacts</label>
                        {props.workflowArtifacts.map(artifact => (
                            <div key={artifact.name} style={{ marginTop: 10 }}>
                                <label style={{ fontWeight: 'normal', fontSize: '0.9em' }}>{artifact.name}</label>
                                <ArtifactsInput
                                    namespace={props.namespace}
                                    artifactName={artifact.name}
                                    onUploadComplete={response => handleArtifactUpload(artifact.name, response)}
                                    onError={setError}
                                />
                                {uploadedArtifacts[artifact.name] && (
                                    <small style={{ color: 'green' }}>✓ Uploaded: {uploadedArtifacts[artifact.name].key}</small>
                                )}
                            </div>
                        ))}
                    </div>
                )}
                <div key='labels' style={{ marginBottom: 25 }}>
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
