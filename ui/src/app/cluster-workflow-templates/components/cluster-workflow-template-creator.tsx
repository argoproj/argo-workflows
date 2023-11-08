import * as React from 'react';
import {useState} from 'react';

import {ClusterWorkflowTemplate} from '../../../models';
import {Button} from '../../shared/components/button';
import {ErrorNotice} from '../../shared/components/error-notice';
import {ExampleManifests} from '../../shared/components/example-manifests';
import {UploadButton} from '../../shared/components/upload-button';
import {exampleClusterWorkflowTemplate} from '../../shared/examples';
import {services} from '../../shared/services';
import {ClusterWorkflowTemplateEditor} from './cluster-workflow-template-editor';

export function ClusterWorkflowTemplateCreator({onCreate}: {onCreate: (workflow: ClusterWorkflowTemplate) => void}) {
    const [template, setTemplate] = useState<ClusterWorkflowTemplate>(exampleClusterWorkflowTemplate());
    const [error, setError] = useState<Error>();
    return (
        <>
            <div>
                <UploadButton onUpload={setTemplate} onError={setError} />
                <Button
                    icon='plus'
                    onClick={async () => {
                        try {
                            const newTemplate = await services.clusterWorkflowTemplate.create(template);
                            onCreate(newTemplate);
                        } catch (err) {
                            setError(err);
                        }
                    }}>
                    Create
                </Button>
            </div>
            <ErrorNotice error={error} />
            <ClusterWorkflowTemplateEditor template={template} onChange={setTemplate} onError={setError} />
            <div>
                <ExampleManifests />.
            </div>
        </>
    );
}
