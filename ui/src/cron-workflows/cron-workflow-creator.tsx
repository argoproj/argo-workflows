import * as React from 'react';
import {useState} from 'react';

import {Button} from '../shared/components/button';
import {ErrorNotice} from '../shared/components/error-notice';
import {ExampleManifests} from '../shared/components/example-manifests';
import {UploadButton} from '../shared/components/upload-button';
import {exampleCronWorkflow} from '../shared/examples';
import {CronWorkflow} from '../shared/models';
import * as nsUtils from '../shared/namespaces';
import {services} from '../shared/services';
import {useEditableObject} from '../shared/use-editable-object';
import {CronWorkflowEditor} from './cron-workflow-editor';

export function CronWorkflowCreator({onCreate, namespace}: {namespace: string; onCreate: (cronWorkflow: CronWorkflow) => void}) {
    const {object: cronWorkflow, setObject: setCronWorkflow, serialization, lang, setLang} = useEditableObject(exampleCronWorkflow(nsUtils.getNamespaceWithDefault(namespace)));
    const [error, setError] = useState<Error>();
    return (
        <>
            <div>
                <UploadButton onUpload={setCronWorkflow} onError={setError} />
                <Button
                    icon='plus'
                    onClick={async () => {
                        try {
                            const newCronWorkflow = await services.cronWorkflows.create(cronWorkflow, nsUtils.getNamespaceWithDefault(cronWorkflow.metadata.namespace));
                            onCreate(newCronWorkflow);
                        } catch (err) {
                            setError(err);
                        }
                    }}>
                    Create
                </Button>
            </div>
            <ErrorNotice error={error} />
            <CronWorkflowEditor cronWorkflow={cronWorkflow} serialization={serialization} lang={lang} onLangChange={setLang} onChange={setCronWorkflow} onError={setError} />
            <p>
                <ExampleManifests />.
            </p>
        </>
    );
}
