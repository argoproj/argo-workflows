import * as React from 'react';
import {useState} from 'react';

import {CronWorkflow} from '../../models';
import {Button} from '../shared/components/button';
import {ErrorNotice} from '../shared/components/error-notice';
import {ExampleManifests} from '../shared/components/example-manifests';
import {UploadButton} from '../shared/components/upload-button';
import {exampleCronWorkflow} from '../shared/examples';
import * as nsUtils from '../shared/namespaces';
import {services} from '../shared/services';
import {CronWorkflowEditor} from './cron-workflow-editor';

export function CronWorkflowCreator({onCreate, namespace}: {namespace: string; onCreate: (cronWorkflow: CronWorkflow) => void}) {
    const [cronWorkflow, setCronWorkflow] = useState<CronWorkflow>(exampleCronWorkflow(nsUtils.getNamespaceWithDefault(namespace)));
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
            <CronWorkflowEditor cronWorkflow={cronWorkflow} onChange={setCronWorkflow} onError={setError} />
            <p>
                <ExampleManifests />.
            </p>
        </>
    );
}
