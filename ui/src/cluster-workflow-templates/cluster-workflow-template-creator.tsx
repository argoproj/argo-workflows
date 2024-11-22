import * as React from 'react';
import {useState} from 'react';

import {Button} from '../shared/components/button';
import {ErrorNotice} from '../shared/components/error-notice';
import {ExampleManifests} from '../shared/components/example-manifests';
import {UploadButton} from '../shared/components/upload-button';
import {exampleClusterWorkflowTemplate} from '../shared/examples';
import {ClusterWorkflowTemplate} from '../shared/models';
import {services} from '../shared/services';
import {useEditableObject} from '../shared/use-editable-object';
import {WorkflowTemplateEditor} from '../workflow-templates/workflow-template-editor';

export function ClusterWorkflowTemplateCreator({onCreate}: {onCreate: (workflow: ClusterWorkflowTemplate) => void}) {
    const {object: template, setObject: setTemplate, serialization, lang, setLang} = useEditableObject(exampleClusterWorkflowTemplate());
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
            <WorkflowTemplateEditor template={template} serialization={serialization} lang={lang} onLangChange={setLang} onChange={setTemplate} onError={setError} />
            <div>
                <ExampleManifests />.
            </div>
        </>
    );
}
