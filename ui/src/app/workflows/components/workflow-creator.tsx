import * as React from 'react';
import {useState} from 'react';
import {Workflow} from '../../../models';
import {Button} from '../../shared/components/button';
import {ErrorNotice} from '../../shared/components/error-notice';
import {ExampleManifests} from '../../shared/components/example-manifests';
import {UploadButton} from '../../shared/components/upload-button';
import {exampleWorkflow} from '../../shared/examples';
import {services} from '../../shared/services';
import {Utils} from '../../shared/utils';
import {WorkflowEditor} from './workflow-editor';

export const WorkflowCreator = ({namespace, onCreate}: {namespace: string; onCreate: (workflow: Workflow) => void}) => {
    const [workflow, setWorkflow] = useState<Workflow>(exampleWorkflow(Utils.getNamespace(namespace)));
    const [error, setError] = useState<Error>();
    return (
        <>
            <div>
                <UploadButton onUpload={setWorkflow} onError={setError} />
                <Button
                    icon='plus'
                    onClick={() => {
                        services.workflows
                            .create(workflow, workflow.metadata.namespace)
                            .then(onCreate)
                            .catch(setError);
                    }}>
                    Create
                </Button>
            </div>
            <ErrorNotice error={error} />
            <WorkflowEditor template={workflow} onChange={setWorkflow} onError={setError} />
            <div>
                <ExampleManifests />.
            </div>
        </>
    );
};
