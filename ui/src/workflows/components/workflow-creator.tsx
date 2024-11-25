import {Select} from 'argo-ui/src/components/select/select';
import * as React from 'react';
import {useEffect, useState} from 'react';

import {Button} from '../../shared/components/button';
import {ErrorNotice} from '../../shared/components/error-notice';
import {ExampleManifests} from '../../shared/components/example-manifests';
import {UploadButton} from '../../shared/components/upload-button';
import {exampleWorkflow} from '../../shared/examples';
import {Workflow, WorkflowTemplate} from '../../shared/models';
import * as nsUtils from '../../shared/namespaces';
import {services} from '../../shared/services';
import {useEditableObject} from '../../shared/use-editable-object';
import {SubmitWorkflowPanel} from './submit-workflow-panel';
import {WorkflowEditor} from './workflow-editor';

type Stage = 'choose-method' | 'submit-workflow' | 'full-editor';

export function WorkflowCreator({namespace, onCreate}: {namespace: string; onCreate: (workflow: Workflow) => void}) {
    const [workflowTemplates, setWorkflowTemplates] = useState<WorkflowTemplate[]>();
    const [workflowTemplate, setWorkflowTemplate] = useState<WorkflowTemplate>();
    const [stage, setStage] = useState<Stage>('choose-method');
    const {object: workflow, setObject: setWorkflow, serialization, lang, setLang} = useEditableObject<Workflow>();
    const [error, setError] = useState<Error>();

    useEffect(() => {
        services.workflowTemplate
            .list(namespace, [])
            .then(list => list.items || [])
            .then(setWorkflowTemplates)
            .catch(setError);
    }, [namespace]);

    useEffect(() => {
        if (stage !== 'full-editor') return;
        if (!workflowTemplate) {
            setWorkflow(exampleWorkflow(nsUtils.getNamespaceWithDefault(namespace)));
            return;
        }

        setWorkflow({
            metadata: {
                generateName: workflowTemplate.metadata.name + '-',
                namespace,
                labels: {
                    'workflows.argoproj.io/workflow-template': workflowTemplate.metadata.name,
                    'submit-from-ui': 'true'
                }
            },
            spec: {
                arguments: workflowTemplate.spec.arguments,
                workflowTemplateRef: {
                    name: workflowTemplate.metadata.name
                }
            }
        });
    }, [stage]);

    useEffect(() => {
        if (workflowTemplate) {
            setStage('submit-workflow');
        }
    }, [workflowTemplate]);

    return (
        <>
            {stage === 'choose-method' && (
                <div className='white-box'>
                    <h4>Submit new workflow</h4>
                    <p>Either:</p>
                    <div style={{margin: 10, marginLeft: 20}}>
                        <Select
                            placeholder='Select a workflow template...'
                            options={workflowTemplates && workflowTemplates.length > 0 ? workflowTemplates.map(tmpl => tmpl.metadata.name) : []}
                            value={workflowTemplate && workflowTemplate.metadata.name}
                            onChange={templateName => setWorkflowTemplate((workflowTemplates || []).find(template => template.metadata.name === templateName.title))}
                        />
                    </div>
                    <p>Or:</p>
                    <div style={{margin: 10, marginLeft: 20}}>
                        <a onClick={() => setStage('full-editor')}>
                            Edit using full workflow options <i className='fa fa-caret-right' />
                        </a>
                    </div>
                </div>
            )}
            {stage === 'submit-workflow' && workflowTemplate && (
                <>
                    <SubmitWorkflowPanel
                        kind='WorkflowTemplate'
                        namespace={workflowTemplate.metadata.namespace}
                        name={workflowTemplate.metadata.name}
                        entrypoint={workflowTemplate.spec.entrypoint}
                        templates={workflowTemplate.spec.templates || []}
                        workflowParameters={workflowTemplate.spec.arguments.parameters || []}
                    />
                    <a onClick={() => setStage('full-editor')}>
                        Edit using full workflow options <i className='fa fa-caret-right' />
                    </a>
                </>
            )}
            {stage === 'full-editor' && workflow && (
                <>
                    <div>
                        <UploadButton onUpload={setWorkflow} onError={setError} />
                        <Button
                            icon='plus'
                            onClick={async () => {
                                try {
                                    const newWorkflow = await services.workflows.create(workflow, nsUtils.getNamespaceWithDefault(workflow.metadata.namespace));
                                    onCreate(newWorkflow);
                                } catch (err) {
                                    setError(err);
                                }
                            }}>
                            Create
                        </Button>
                    </div>
                    <ErrorNotice error={error} />
                    <WorkflowEditor workflow={workflow} serialization={serialization} lang={lang} onLangChange={setLang} onChange={setWorkflow} onError={setError} />
                    <div>
                        <ExampleManifests />.
                    </div>
                </>
            )}
        </>
    );
}
