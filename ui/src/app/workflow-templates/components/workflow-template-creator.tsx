import * as React from 'react';
import {useState} from 'react';
import {Workflow, WorkflowTemplate} from '../../../models';
import {Button} from '../../shared/components/button';
import {ErrorNotice} from '../../shared/components/error-notice';
import {ExampleManifests} from '../../shared/components/example-manifests';
import {UploadButton} from '../../shared/components/upload-button';
import {exampleWorkflowTemplate} from '../../shared/examples';
import {services} from '../../shared/services';
import {WorkflowTemplateEditor} from './workflow-template-editor';

export const WorkflowTemplateCreator = (props: {namespace: string; onCreate: (workflow: Workflow) => void}) => {
    const [template, setTemplate] = useState<WorkflowTemplate>(exampleWorkflowTemplate(props.namespace || 'default'));
    const [error, setError] = useState<Error>();
    return (
        <>
            <div>
                <UploadButton onUpload={setTemplate} onError={setError} />
                <Button
                    icon='plus'
                    onClick={() => {
                        services.workflowTemplate
                            .create(template, template.metadata.namespace)
                            .then(props.onCreate)
                            .catch(setError);
                    }}>
                    Create
                </Button>
            </div>
            <ErrorNotice error={error} />
            <WorkflowTemplateEditor template={template} onChange={setTemplate} onError={setError} />
            <p>
                <ExampleManifests />.
            </p>
        </>
    );
};
