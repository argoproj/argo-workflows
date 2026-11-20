import * as React from 'react';
import {useEffect, useState} from 'react';

import {Button} from '../shared/components/button';
import {ErrorNotice} from '../shared/components/error-notice';
import {ExampleManifests} from '../shared/components/example-manifests';
import {UploadButton} from '../shared/components/upload-button';
import {exampleWorkflowTemplate} from '../shared/examples';
import {WorkflowTemplate} from '../shared/models';
import * as nsUtils from '../shared/namespaces';
import {services} from '../shared/services';
import {useEditableObject} from '../shared/use-editable-object';
import {WorkflowTemplateEditor} from './workflow-template-editor';

export function WorkflowTemplateCreator({namespace, onCreate}: {namespace: string; onCreate: (workflow: WorkflowTemplate) => void}) {
    const {
        object: template,
        setObject: setTemplate,
        serialization,
        lang,
        setLang,
        resetObject
    } = useEditableObject(exampleWorkflowTemplate(nsUtils.getNamespaceWithDefault(namespace)));
    const [error, setError] = useState<Error>();

    // Sync template metadata.namespace when namespace prop changes (e.g., managed namespace becomes available)
    useEffect(() => {
        if (template.metadata.namespace !== namespace) {
            resetObject({
                ...template,
                metadata: {
                    ...template.metadata,
                    namespace
                }
            });
        }
    }, [namespace, template, resetObject]);

    return (
        <>
            <div>
                <UploadButton onUpload={setTemplate} onError={setError} />
                <Button
                    icon='plus'
                    onClick={async () => {
                        try {
                            const newTemplate = await services.workflowTemplate.create(template, nsUtils.getNamespaceWithDefault(template.metadata.namespace));
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
            <p>
                <ExampleManifests />.
            </p>
        </>
    );
}
